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

	corev1 "k8s.io/api/core/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	errBoom := errors.New("boom")
	errItemNotFound := kerrors.NewNotFound(schema.GroupResource{}, "mock resource")
	tokenSecret := "upbound-control-plane-token"

	type args struct {
		mgr manager.Manager
		req reconcile.Request
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
		"ErrNotTokenInSecret": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(nil)},
				},
			},
			want: want{
				err: errors.Errorf(errNoTokenInSecret, tokenSecret, keyToken),
			},
		},
		"ErrGetSpecConfigMap": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
						if s, ok := obj.(*corev1.Secret); ok {
							s.Data = map[string][]byte{
								keyToken: []byte("some-token"),
							}
							return nil
						}
						return errBoom
					})},
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errGetSpecCM),
			},
		},
		"ErrNoSpecInConfigMap": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
						if s, ok := obj.(*corev1.Secret); ok {
							s.Data = map[string][]byte{
								keyToken: []byte("some-token"),
							}
							return nil
						}
						return nil
					})},
				},
			},
			want: want{
				err: errors.Errorf(errNoSpecInCM, configMapAgentDeploymentSpec, keySpec),
			},
		},
		"ErrFailedToUnmarshallAsSpec": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
						if s, ok := obj.(*corev1.Secret); ok {
							s.Data = map[string][]byte{
								keyToken: []byte("some-token"),
							}
							return nil
						}
						if s, ok := obj.(*corev1.ConfigMap); ok {
							s.Data = map[string]string{
								keySpec: "not-a-valid-spec",
							}
							return nil
						}
						return nil
					})},
				},
			},
			want: want{
				err: errors.Wrap(errors.Wrap(errors.New("error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go value of type v1.DeploymentSpec"),
					errFailedToUnmarshall), errFailedToSyncDeployment),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewReconciler(tc.args.mgr, tokenSecret)
			_, err := r.Reconcile(context.Background(), tc.args.req)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n\nr.Reconcile(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}
