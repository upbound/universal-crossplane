package operations

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/upbound/crossplane-distro/internal/clients/upbound"
	"github.com/upbound/crossplane-distro/internal/controller/bootstrap/meta"
)

type UBCCertsFetcher struct {
	client    client.Client
	ubcClient upbound.Client
	namespace string
}

const (
	keyJWTPublicKey = "jwtPublicKey"
	keyNATSCA       = "natsCA"

	keyCPTokenSecretName  = "cpTokenSecretName"
	keyToken              = "token"
	secretNamePublicCerts = "upbound-agent-public-certs"
)

func NewUBCCertsFetcher(c client.Client, ubcClient upbound.Client, namespace string) *UBCCertsFetcher {
	return &UBCCertsFetcher{
		client:    c,
		ubcClient: ubcClient,
		namespace: namespace,
	}
}

func (u *UBCCertsFetcher) Run(ctx context.Context, log logging.Logger, config map[string][]byte) error {
	log.Debug("Running UBCCertsFetcher")
	tsn := string(config[keyCPTokenSecretName])
	if tsn == "" {
		log.Debug("No control plane token secret provided in configuration secret, skipping fetching Upbound agent public certs")
		return nil
	}

	ts := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tsn,
			Namespace: u.namespace,
		},
	}

	if err := u.client.Get(ctx, types.NamespacedName{Name: tsn, Namespace: u.namespace}, ts); err != nil {
		return errors.Wrapf(err, "failed to get control plane token secret %s", tsn)
	}

	cpToken := string(ts.Data[keyToken])
	if cpToken == "" {
		return errors.Errorf("no token found for key %s in control plane token secret", keyToken)
	}

	log.Info("Fetching Upbound agent public certs...")
	j, n, err := u.ubcClient.GetGatewayCerts(cpToken)
	if err != nil {
		return errors.Wrap(err, "failed to fetch agent public keys")
	}

	js := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNamePublicCerts,
			Namespace: u.namespace,
			Labels: map[string]string{
				meta.LabelKeyManagedBy: meta.LabelValueManagedBy,
			},
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, u.client, js, func() error {
		d := map[string][]byte{
			keyJWTPublicKey: []byte(j),
			keyNATSCA:       []byte(n),
		}
		js.Data = d
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to create/update agent public certs secret")
	}
	log.Info("Fetching Upbound agent public certs completed")

	return nil
}
