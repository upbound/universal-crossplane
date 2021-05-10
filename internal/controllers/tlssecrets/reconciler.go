// Copyright 2021 Upbound Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tlssecrets

import (
	"context"
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/upbound/universal-crossplane/internal/meta"
)

const (
	reconcileTimeout = 1 * time.Minute

	certificateBlockType = "CERTIFICATE"
	rsaKeySize           = 2048
	certificateValidity  = time.Hour * 24 * 365 * 10

	keyCACert  = "ca.crt"
	keyTLSCert = "tls.crt"
	keyTLSKey  = "tls.key"

	nameUpbound          = "upbound"
	cnAgent              = "upbound-agent"
	cnXgql               = "xgql"
	secretNameCA         = "uxp-ca"
	secretNameGatewayTLS = "upbound-agent-tls"
	secretNameXgqlTLS    = "xgql-tls"
)

const (
	errGetSecret        = "failed to get cert secret"
	errGetCASecret      = "failed get ca secret"
	errInitCA           = "failed to initialize ca"
	errUpdateCASecret   = "failed to update ca secret"
	errUpdateCertSecret = "failed to update secret for certificate data"
)

// Event reasons.
const (
	reasonCA       event.Reason = "InitializingCA"
	reasonGenerate event.Reason = "GeneratingCertificates"
	reasonSync     event.Reason = "SyncCerts"
	reasonUpdate   event.Reason = "UpdatingSecret"
)

var (
	caConfig = &certutil.Config{
		CommonName:   nameUpbound,
		Organization: []string{nameUpbound},
	}
	certConfigs = map[string]*certutil.Config{
		secretNameGatewayTLS: {
			CommonName: cnAgent,
			AltNames: certutil.AltNames{
				// TODO(hasan): drop "tenant-gateway" once we stop using legacy service
				DNSNames: []string{cnAgent, "tenant-gateway"},
			},
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		secretNameXgqlTLS: {
			CommonName: cnXgql,
			AltNames: certutil.AltNames{
				DNSNames: []string{cnXgql},
			},
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
	}
)

// ReconcilerOption is used to configure the Reconciler.
type ReconcilerOption func(*Reconciler)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

// WithRecorder specifies how the Reconciler should record Kubernetes events.
func WithRecorder(er event.Recorder) ReconcilerOption {
	return func(r *Reconciler) {
		r.record = er
	}
}

// Setup adds a controller that reconciles on tls secrets
func Setup(mgr ctrl.Manager, l logging.Logger) error {
	name := "tlsSecretGeneration"

	r := NewReconciler(mgr,
		WithLogger(l.WithValues("controller", name)),
		WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&corev1.Secret{}).
		WithEventFilter(resource.NewPredicates(resource.AnyOf(
			resource.IsNamed(secretNameGatewayTLS),
			resource.IsNamed(secretNameXgqlTLS),
		))).
		Complete(r)
}

// Reconciler reconciles on tls secrets
type Reconciler struct {
	client client.Client
	log    logging.Logger
	record event.Recorder
	caCert *x509.Certificate
	caKey  crypto.Signer
}

// NewReconciler returns a new reconciler
func NewReconciler(mgr manager.Manager, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		client: mgr.GetClient(),
		log:    logging.NewNopLogger(),
		record: event.NewNopRecorder(),
	}

	for _, f := range opts {
		f(r)
	}

	return r
}

// Reconcile reconciles on tls secrets for uxp and fills the secret data with generated certificates
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", req)

	log.Debug("Reconciling...")
	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()

	s := &corev1.Secret{}
	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, s)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, errGetSecret)
	}

	// Check if secret has data
	cert := s.Data[keyTLSCert]
	if string(cert) != "" {
		m := fmt.Sprintf("Secret %s already contains certificate, skipping generation", req.Name)
		log.Debug(m)
		r.record.Event(s, event.Normal(reasonSync, m))
		return reconcile.Result{}, nil
	}

	if err = r.createOrLoadCA(ctx, req.Namespace); err != nil {
		err = errors.Wrap(err, errInitCA)
		log.Info(err.Error())
		r.record.Event(s, event.Warning(reasonCA, err))
		return reconcile.Result{}, err
	}

	log.Info("Generating certificate...")

	c, k, err := newSignedCertAndKey(certConfigs[req.Name], r.caCert, r.caKey)
	if err != nil {
		err = errors.Wrap(err, "failed to generate signed certificate")
		log.Info(err.Error())
		r.record.Event(s, event.Warning(reasonGenerate, err))
		return reconcile.Result{}, err
	}
	d, err := tlsSecretDataFromCertAndKey(c, k, r.caCert)
	if err != nil {
		err = errors.Wrap(err, "failed to build secret data from generated certificate")
		log.Info(err.Error())
		r.record.Event(s, event.Warning(reasonGenerate, err))
		return reconcile.Result{}, err
	}

	s.Labels = map[string]string{
		meta.LabelKeyManagedBy: meta.LabelValueManagedBy,
	}
	s.Data = d

	if err = r.client.Update(ctx, s); err != nil {
		err = errors.Wrap(err, errUpdateCertSecret)
		log.Info(err.Error())
		r.record.Event(s, event.Warning(reasonUpdate, err))
		return reconcile.Result{}, err
	}

	m := "Successfully generated certificate!"
	log.Info(m)
	r.record.Event(s, event.Normal(reasonUpdate, m))
	return reconcile.Result{}, nil
}

func (r *Reconciler) createOrLoadCA(ctx context.Context, namespace string) error {
	cas := &corev1.Secret{}
	err := r.client.Get(ctx, types.NamespacedName{Name: secretNameCA, Namespace: namespace}, cas)
	if resource.IgnoreNotFound(err) != nil {
		return errors.Wrap(err, errGetCASecret)
	}
	if err == nil && string(cas.Data[keyTLSKey]) != "" {
		// load ca from existing secret
		c, k, _, err := certFromTLSSecretData(cas.Data)
		if err != nil {
			return errors.Wrap(err, "failed to parts existing ca secret")
		}
		r.caCert = c
		r.caKey = k
		return nil
	}

	// ca secret does not exist, generate and save
	c, k, err := newCertificateAuthority(caConfig)
	if err != nil {
		return errors.Wrap(err, "failed to generate new ca")
	}
	r.caCert = c
	r.caKey = k
	d, err := tlsSecretDataFromCertAndKey(c, k, c)
	if err != nil {
		return errors.Wrap(err, "failed to build tls secret data from generated ca")
	}
	cas = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNameCA,
			Namespace: namespace,
			Labels: map[string]string{
				meta.LabelKeyManagedBy: meta.LabelValueManagedBy,
			},
		},
		Type: corev1.SecretTypeTLS,
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, cas, func() error {
		cas.Data = d
		return nil
	})
	return errors.Wrap(err, errUpdateCASecret)
}

// newCertificateAuthority creates new certificate and private key for the certificate authority
func newCertificateAuthority(config *certutil.Config) (*x509.Certificate, crypto.Signer, error) {
	key, err := rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key while generating CA certificate")
	}

	cert, err := certutil.NewSelfSignedCACert(*config, key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create self-signed CA certificate")
	}

	return cert, key, nil
}

// newSignedCertAndKey creates new certificate and key by passing the certificate authority certificate and key
func newSignedCertAndKey(config *certutil.Config, caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, crypto.Signer, error) {
	key, err := rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key")
	}

	cert, err := newSignedCert(config, key, caCert, caKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to sign certificate")
	}

	return cert, key, nil
}

// newSignedCert creates a signed certificate using the given CA certificate and key
func newSignedCert(cfg *certutil.Config, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, error) {
	serial, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(certificateValidity).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// encodeCertPEM returns PEM-encoded certificate data
func encodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  certificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

func tlsSecretDataFromCertAndKey(cert *x509.Certificate, key crypto.Signer, ca *x509.Certificate) (map[string][]byte, error) {
	d := make(map[string][]byte)
	d[keyTLSKey] = []byte{}
	if key != nil {
		keyEncoded, err := keyutil.MarshalPrivateKeyToPEM(key)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode tls key as PEM")
		}
		d[keyTLSKey] = keyEncoded
	}
	if cert != nil {
		certEncoded := encodeCertPEM(cert)
		d[keyTLSCert] = certEncoded
	}

	if ca != nil {
		caEncoded := encodeCertPEM(ca)
		d[keyCACert] = caEncoded
	}

	return d, nil
}

func certFromTLSSecretData(data map[string][]byte) (cert *x509.Certificate, key crypto.Signer, ca *x509.Certificate, err error) {
	keyEncoded, ok := data[keyTLSKey]
	if !ok {
		err = errors.New(fmt.Sprintf("could not find key %s in ca secret", keyTLSKey))
		return
	}
	// Not all tls secrets contain private key, i.e. etcd ca cert to trust
	if len(keyEncoded) > 0 {
		var k interface{}
		k, err = keyutil.ParsePrivateKeyPEM(keyEncoded)
		if err != nil {
			err = errors.Wrap(err, "failed to parse private key as PEM")
			return
		}
		key, ok = k.(*rsa.PrivateKey)
		if !ok {
			err = errors.New("private key is not in recognized type, expecting RSA")
			return
		}
	}

	certEncoded, ok := data[keyTLSCert]
	if !ok {
		err = errors.New(fmt.Sprintf("could not find key %s in ca secret", keyTLSCert))
		return
	}
	certs, err := certutil.ParseCertsPEM(certEncoded)
	if err != nil {
		err = errors.Wrap(err, "failed to parse cert as PEM")
		return
	}
	cert = certs[0]

	caEncoded, ok := data[keyCACert]
	if !ok {
		return
	}
	cas, err := certutil.ParseCertsPEM(caEncoded)
	if err != nil {
		err = errors.Wrap(err, "failed to parse ca cert as PEM")
		return
	}
	ca = cas[0]

	return cert, key, ca, err
}
