package operations

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

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/upbound/crossplane-distro/internal/controller/bootstrap/meta"
)

const (
	certificateBlockType = "CERTIFICATE"
	rsaKeySize           = 2048
	certificateValidity  = time.Hour * 24 * 365 * 10

	keyCACert  = "ca.crt"
	keyTLSCert = "tls.crt"
	keyTLSKey  = "tls.key"

	nameUpbound          = "upbound"
	cnGateway            = "upbound-agent-gateway"
	cnGraphql            = "upbound-agent-graphql"
	secretNameCA         = "upbound-ca"
	secretNameGatewayTLS = "upbound-agent-gateway-tls"
	secretNameGraphqlTLS = "upbound-agent-graphql-tls"
)

var (
	caConfig = &certutil.Config{
		CommonName:   nameUpbound,
		Organization: []string{nameUpbound},
	}
	certConfigs = map[string]*certutil.Config{
		secretNameGatewayTLS: {
			CommonName: cnGateway,
			AltNames: certutil.AltNames{
				DNSNames: []string{cnGateway},
			},
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		secretNameGraphqlTLS: {
			CommonName: cnGraphql,
			AltNames: certutil.AltNames{
				DNSNames: []string{cnGraphql},
			},
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
	}
)

type TLSSecretGeneration struct {
	client    client.Client
	namespace string
	caCert    *x509.Certificate
	caKey     crypto.Signer
}

func NewTLSSecretGeneration(c client.Client, namespace string) *TLSSecretGeneration {
	return &TLSSecretGeneration{
		client:    c,
		namespace: namespace,
	}
}

func (t *TLSSecretGeneration) createOrLoadCA(ctx context.Context) error {
	cas := &corev1.Secret{}
	err := t.client.Get(ctx, types.NamespacedName{Name: secretNameCA, Namespace: t.namespace}, cas)
	if resource.IgnoreNotFound(err) != nil {
		return errors.Wrap(err, "failed get ca secret")
	}
	if err == nil {
		// load ca from existing secret
		c, k, _, err := certFromTLSSecretData(cas.Data)
		if err != nil {
			return errors.Wrap(err, "failed to parts existing ca secret")
		}
		t.caCert = c
		t.caKey = k
		return nil
	}

	// ca secret does not exist, generate and save
	c, k, err := newCertificateAuthority(caConfig)
	if err != nil {
		return errors.Wrap(err, "failed to generate new ca")
	}
	t.caCert = c
	t.caKey = k
	d, err := tlsSecretDataFromCertAndKey(c, k, c)
	cas = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNameCA,
			Namespace: t.namespace,
			Labels: map[string]string{
				meta.LabelKeyManagedBy: meta.LabelValueManagedBy,
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: d,
	}
	return errors.Wrap(t.client.Create(ctx, cas), "failed to create ca secret")
}

func (t *TLSSecretGeneration) Run(ctx context.Context, log logging.Logger, config map[string][]byte) error {
	log.Debug("Running TLSSecretGeneration")
	if t.caCert == nil {
		if err := t.createOrLoadCA(ctx); err != nil {
			return errors.Wrap(err, "failed to initialize ca")
		}
	}

	for n, cfg := range certConfigs {
		s := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      n,
				Namespace: t.namespace,
				Labels: map[string]string{
					meta.LabelKeyManagedBy: meta.LabelValueManagedBy,
				},
			},
		}
		err := t.client.Get(ctx, types.NamespacedName{Name: n, Namespace: t.namespace}, s)
		if resource.IgnoreNotFound(err) != nil {
			return errors.Wrapf(err, "failed to get cert secret %s", n)
		}
		if err == nil {
			log.Debug(fmt.Sprintf("Certificate secret %s already exists, skipping generation", n))
			continue
		}

		log.Info(fmt.Sprintf("Generating certificate for %s...", n))
		_, err = controllerutil.CreateOrUpdate(ctx, t.client, s, func() error {
			cert, key, err := newSignedCertAndKey(cfg, t.caCert, t.caKey)
			if err != nil {
				return err
			}
			d, err := tlsSecretDataFromCertAndKey(cert, key, t.caCert)
			if err != nil {
				return err
			}
			s.Data = d
			s.Type = corev1.SecretTypeTLS
			return nil
		})
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Certificate generation completed for %s", n))
	}

	return nil
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
			err = errors.New(fmt.Sprintf("private key is not in recognized type, expecting RSA"))
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
