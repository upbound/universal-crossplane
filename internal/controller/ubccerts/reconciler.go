package ubccerts

import (
	"context"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/upbound/crossplane-distro/internal/clients/upbound"
	"github.com/upbound/crossplane-distro/internal/meta"
)

const (
	reconcileTimeout = 1 * time.Minute

	keyJWTPublicKey       = "jwtPublicKey"
	keyNATSCA             = "natsCA"
	keyToken              = "token"
	secretNameCPToken     = "upbound-control-plane-token"
	secretNamePublicCerts = "upbound-agent-public-certs"
)

const (
	errGetSecret        = "failed to get ubc public certs secret"
	errGetCPTokenSecret = "failed to get control plane token secret %s"
	errNoTokenInSecret  = "No token found for key %s in control plane token secret %s, skipping fetching Upbound agent public certs"
	errFetch            = "failed to fetch agent public keys"
	errUpdateSecret     = "failed to update agent public certs secret"
)

// Event reasons.
const (
	reasonToken  event.Reason = "ReadToken"
	reasonFetch  event.Reason = "FetchFromUpbound"
	reasonUpdate event.Reason = "UpdatingSecret"
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

// Setup adds a controller that reconciles on ubc cert secret
func Setup(mgr ctrl.Manager, l logging.Logger, ubcClient upbound.Client) error {
	name := "ubcCertsFetcher"

	r := NewReconciler(mgr,
		ubcClient,
		WithLogger(l.WithValues("controller", name)),
		WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&corev1.Secret{}).
		WithEventFilter(resource.NewPredicates(resource.IsNamed(secretNamePublicCerts))).
		Complete(r)
}

// Reconciler reconciles on ubc cert secret
type Reconciler struct {
	client    client.Client
	log       logging.Logger
	ubcClient upbound.Client
	record    event.Recorder
}

// NewReconciler returns a new reconciler
func NewReconciler(mgr ctrl.Manager, ubcClient upbound.Client, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		client:    mgr.GetClient(),
		log:       logging.NewNopLogger(),
		record:    event.NewNopRecorder(),
		ubcClient: ubcClient,
	}

	for _, f := range opts {
		f(r)
	}

	return r
}

// Reconcile reconciles on ubc public certs secret for uxp and fills the secret data with fetched public certs
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

	ts := &corev1.Secret{}
	err = r.client.Get(ctx, types.NamespacedName{Name: secretNameCPToken, Namespace: req.Namespace}, ts)
	if err != nil {
		err = errors.Wrapf(err, errGetCPTokenSecret, secretNameCPToken)
		log.Info(err.Error())
		r.record.Event(s, event.Warning(reasonToken, err))
		return reconcile.Result{}, err
	}

	cpToken := string(ts.Data[keyToken])
	if cpToken == "" {
		err = errors.Errorf(errNoTokenInSecret, keyToken, secretNameCPToken)
		log.Info(err.Error())
		r.record.Event(s, event.Warning(reasonToken, err))
		return reconcile.Result{}, err
	}

	log.Info("Fetching Upbound agent public certs...")
	certs, err := r.ubcClient.GetGatewayCerts(cpToken)
	if err != nil {
		err = errors.Wrap(err, errFetch)
		log.Info(err.Error())
		r.record.Event(s, event.Warning(reasonFetch, err))
		return reconcile.Result{}, err
	}

	s.Labels = map[string]string{
		meta.LabelKeyManagedBy: meta.LabelValueManagedBy,
	}
	s.Data = map[string][]byte{
		keyJWTPublicKey: []byte(certs.JWTPublicKey),
		keyNATSCA:       []byte(certs.NATSCA),
	}

	if err = r.client.Update(ctx, s); err != nil {
		err = errors.Wrap(err, errUpdateSecret)
		log.Info(err.Error())
		r.record.Event(s, event.Warning(reasonUpdate, err))
		return reconcile.Result{}, err
	}

	m := "Successfully fetched Upbound agent public certs secret!"
	log.Info(m)
	r.record.Event(s, event.Normal(reasonUpdate, m))
	return reconcile.Result{}, nil
}
