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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	errBoom := errors.New("boom")
	errItemNotFound := kerrors.NewNotFound(schema.GroupResource{}, "mock resource")
	tokenSecret := "upbound-control-plane-token"
	deploymentSpec := appsv1.DeploymentSpec{}

	type args struct {
		mgr manager.Manager
	}
	type want struct {
		err error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ErrGetTokenSecret": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						if key.Name == tokenSecret {
							return errBoom
						}
						return nil
					}},
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errGetSecret),
			},
		},
		"TokenSecretNotFound": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						if key.Name == tokenSecret {
							return errItemNotFound
						}
						return nil
					}},
				},
			},
			want: want{
				err: nil,
			},
		},
		"ErrNoTokenInSecret": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(nil)},
				},
			},
			want: want{
				err: errors.Errorf(errNoTokenInSecret, tokenSecret, keyToken),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewReconciler(tc.args.mgr, deploymentSpec, tokenSecret)
			_, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "upbound-system",
					Name:      tokenSecret,
				},
			})

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n\nr.Reconcile(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}
