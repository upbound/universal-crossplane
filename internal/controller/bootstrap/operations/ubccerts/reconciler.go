package ubccerts

import (
	"context"
	"fmt"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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

const (
	reconcileTimeout = 1 * time.Minute

	keyJWTPublicKey       = "jwtPublicKey"
	keyNATSCA             = "natsCA"
	keyToken              = "token"
	secretNameCPToken     = "upbound-control-plane-token"
	secretNamePublicCerts = "upbound-agent-public-certs"
)

// ReconcilerOption is used to configure the Reconciler.
type ReconcilerOption func(*Reconciler)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

func Setup(mgr ctrl.Manager, l logging.Logger, ubcClient upbound.Client) error {
	name := "fetchingUBCCerts"

	r := NewReconciler(mgr,
		ubcClient,
		WithLogger(l.WithValues("controller", name)),
	)

	//TODO(hasan): watch secret with specific name
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&corev1.Secret{}).
		Complete(r)
}

type Reconciler struct {
	client    client.Client
	log       logging.Logger
	ubcClient upbound.Client
}

func NewReconciler(mgr ctrl.Manager, ubcClient upbound.Client, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		client:    mgr.GetClient(),
		log:       logging.NewNopLogger(),
		ubcClient: ubcClient,
	}

	for _, f := range opts {
		f(r)
	}

	return r
}

func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", req)
	if req.Name != secretNamePublicCerts {
		return reconcile.Result{}, nil
	}

	log.Debug("Reconciling...")
	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()

	s := &corev1.Secret{}
	err := r.client.Get(ctx, types.NamespacedName{Name: secretNameCPToken, Namespace: req.Namespace}, s)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to get control plane token secret %s", req.Name)
	}

	cpToken := string(s.Data[keyToken])
	if cpToken == "" {
		log.Debug(fmt.Sprintf("No token found for key %s in control plane token secret %s, skipping fetching Upbound agent public certs", keyToken, secretNameCPToken))
		return reconcile.Result{}, nil
	}

	log.Info("Fetching Upbound agent public certs...")
	j, n, err := r.ubcClient.GetGatewayCerts(cpToken)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to fetch agent public keys")
	}

	js := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNamePublicCerts,
			Namespace: req.Namespace,
			Labels: map[string]string{
				meta.LabelKeyManagedBy: meta.LabelValueManagedBy,
			},
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, js, func() error {
		d := map[string][]byte{
			keyJWTPublicKey: []byte(j),
			keyNATSCA:       []byte(n),
		}
		js.Data = d
		return nil
	})
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to create/update agent public certs secret")
	}
	log.Info("Fetching Upbound agent public certs completed")

	return reconcile.Result{}, nil
}
