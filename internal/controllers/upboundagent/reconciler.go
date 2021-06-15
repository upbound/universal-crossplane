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

package upboundagent

import (
	"context"
	"time"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/yaml"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	internalmeta "github.com/upbound/universal-crossplane/internal/meta"
)

const (
	reconcileTimeout = 1 * time.Minute

	deploymentUpboundAgent       = "upbound-agent"
	configMapAgentDeploymentSpec = "upbound-agent-deployment-spec"
	keySpec                      = "spec.yaml"
	keyToken                     = "token"
)

const (
	errGetSecret              = "failed to get control plane token secret"
	errNoTokenInSecret        = "secret %s does not contain a token for key \"%s\""
	errGetSpecCM              = "failed to get agent spec configmap"
	errNoSpecInCM             = "configmap %s does not contain deployment spec for Upbound agent for key \"%s\""
	errFailedToSyncDeployment = "failed to sync agent deployment"
	errFailedToUnmarshall     = "failed to unmarshall as deployment spec"
)

var (
	secretsKind     = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
	configmapsKind  = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
	deploymentsKind = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
)

// ReconcilerOption is used to configure the Reconciler.
type ReconcilerOption func(*Reconciler)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(log logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

// Setup adds a controller that reconciles on control plane token secret and manages Upbound Agent deployment
func Setup(mgr ctrl.Manager, l logging.Logger, ts string) error {
	name := "upboundAgent"

	r := NewReconciler(mgr, ts,
		WithLogger(l.WithValues("controller", name)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}).
		WithEventFilter(resource.NewPredicates(resource.AnyOf(
			resource.AllOf(IsOfKind(secretsKind, mgr.GetScheme()), resource.IsNamed(ts)),
			resource.AllOf(IsOfKind(configmapsKind, mgr.GetScheme()), resource.IsNamed(configMapAgentDeploymentSpec)),
			resource.AllOf(IsOfKind(deploymentsKind, mgr.GetScheme()), resource.IsNamed(deploymentUpboundAgent)),
		))).
		Complete(r)
}

// Reconciler reconciles on control plane token secret and manages Upbound Agent deployment
type Reconciler struct {
	tokenSecret string
	client      client.Client
	log         logging.Logger
}

// NewReconciler returns a new reconciler
func NewReconciler(mgr manager.Manager, ts string, opts ...ReconcilerOption) *Reconciler {
	r := &Reconciler{
		client:      mgr.GetClient(),
		tokenSecret: ts,
		log:         logging.NewNopLogger(),
	}

	for _, f := range opts {
		f(r)
	}

	return r
}

// Reconcile reconciles on control plane token secret and manages Upbound Agent deployment
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("request", req)

	log.Debug("Reconciling...")
	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()

	ts := &corev1.Secret{}
	err := r.client.Get(ctx, types.NamespacedName{Name: r.tokenSecret, Namespace: req.Namespace}, ts)

	// We are using owner references to get agent deployment deleted when token secret is deleted.
	// Nothing to do here if token secret deleted.
	if kerrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, errGetSecret)
	}

	// Ensure secret has token
	t := ts.Data[keyToken]
	if string(t) == "" {
		err := errors.Errorf(errNoTokenInSecret, r.tokenSecret, keyToken)
		log.Info(err.Error())
		return reconcile.Result{}, err
	}

	if err := r.syncAgentDeployment(ctx, ts); err != nil {
		log.Info(err.Error())
		return reconcile.Result{}, err
	}

	log.Info("Successfully synced Upbound Agent deployment!")
	return reconcile.Result{}, nil
}

func (r *Reconciler) syncAgentDeployment(ctx context.Context, ts *corev1.Secret) error {
	cm := &corev1.ConfigMap{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: configMapAgentDeploymentSpec, Namespace: ts.Namespace}, cm); err != nil {
		return errors.Wrap(err, errGetSpecCM)
	}

	ds := cm.Data[keySpec]
	if ds == "" {
		return errors.Errorf(errNoSpecInCM, configMapAgentDeploymentSpec, keySpec)
	}

	agentDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentUpboundAgent,
			Namespace: ts.Namespace,
			Labels: map[string]string{
				internalmeta.LabelKeyManagedBy: internalmeta.LabelValueManagedBy,
			},
			OwnerReferences: []metav1.OwnerReference{meta.AsController(meta.TypedReferenceTo(ts, ts.GroupVersionKind()))},
		},
	}

	// crossplane runtime NewAPIUpdatingApplicator causes constant updates on the object
	// no matter it is really changed or not. This triggers another reconcile loop hence another
	// update. NewAPIPatchingApplicator does not cause above but we need update rather than
	// patch here (e.g. we removed an env var from agent deployment in an upcoming version).
	_, err := controllerutil.CreateOrUpdate(ctx, r.client, agentDeployment, func() error {
		if err := yaml.Unmarshal([]byte(ds), &agentDeployment.Spec); err != nil {
			return errors.Wrap(err, errFailedToUnmarshall)
		}
		return nil
	})
	return errors.Wrap(err, errFailedToSyncDeployment)
}

// IsOfKind accepts objects that are of the supplied managed resource kind.
// TODO(turkenh): move to crossplane-runtime?
func IsOfKind(k schema.GroupVersionKind, ot runtime.ObjectTyper) resource.PredicateFn {
	return func(obj runtime.Object) bool {
		gvk, err := resource.GetKind(obj, ot)
		if err != nil {
			return false
		}
		return gvk == k
	}
}
