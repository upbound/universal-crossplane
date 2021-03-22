package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/upbound/crossplane-distro/internal/controller/bootstrap/operations"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	reconcileTimeout = 1 * time.Minute
	secretNameConfig = "uxp-config"
)

const (
	errGetConfigSecret = "cannot get config secret"
	errRunOperation    = "failed to run operation %s"
)

type Operation interface {
	GetName() string
	Run(ctx context.Context, log logging.Logger, config map[string][]byte) error
}

// ReconcilerOption is used to configure the Reconciler.
type ReconcilerOption func(*Reconciler)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

type Reconciler struct {
	client     client.Client
	log        logging.Logger
	operations []Operation
}

// Setup adds a controller that runs bootstrap operations
func Setup(mgr ctrl.Manager, l logging.Logger) error {
	name := "bootstrap"

	r := NewReconciler(mgr,
		WithLogger(l.WithValues("controller", name)),
	)

	//TODO(hasan): watch secret with specific name
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&corev1.Secret{}).
		Complete(r)
}

func setupOperations() []Operation {
	ops := make([]Operation, 0, 1)
	return append(ops, operations.NewTLSSecretOperation())
}

func NewReconciler(mgr manager.Manager, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		client:     mgr.GetClient(),
		log:        logging.NewNopLogger(),
		operations: setupOperations(),
	}

	for _, f := range opts {
		f(r)
	}

	return r
}

func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", req)
	log.Info("Reconciling")
	if req.Name != secretNameConfig {
		return reconcile.Result{}, nil
	}
	log.Info("Reconciling for bootstrap")
	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()

	cfgSecret := &corev1.Secret{}
	if err := r.client.Get(ctx, req.NamespacedName, cfgSecret); err != nil {
		log.Debug(errGetConfigSecret, "error", err)
		return reconcile.Result{}, errors.Wrap(err, errGetConfigSecret)
	}

	for _, o := range r.operations {
		if err := o.Run(ctx, log, cfgSecret.Data); err != nil {
			log.Debug(fmt.Sprintf(errRunOperation, o.GetName()), "error", err)
			return reconcile.Result{}, errors.Wrap(err, fmt.Sprintf(errRunOperation, o.GetName()))
		}
	}

	return reconcile.Result{}, nil
}
