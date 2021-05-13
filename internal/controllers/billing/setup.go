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

package billing

import (
	"context"

	"github.com/upbound/universal-crossplane/internal/controllers/billing/aws"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/upbound/universal-crossplane/internal/meta"
)

// SetupAWSMarketplace adds the AWS Marketplace controller that registers this
// instance with AWS Marketplace.
func SetupAWSMarketplace(mgr ctrl.Manager, l logging.Logger) error {
	name := "aws-marketplace"
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithEC2IMDSRegion())
	if err != nil {
		return errors.Wrap(err, "cannot load default AWS config")
	}
	reg := aws.NewMarketplace(mgr.GetClient(), marketplacemetering.NewFromConfig(cfg), aws.MarketplacePublicKey)

	r := NewReconciler(mgr,
		WithLogger(l.WithValues("controller", name)),
		WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		WithRegisterer(reg),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&corev1.Secret{}).
		WithEventFilter(resource.NewPredicates(resource.IsNamed(meta.SecretNameEntitlement))).
		Complete(r)
}
