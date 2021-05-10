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
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"

	"github.com/google/go-cmp/cmp"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/pkg/test"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errBoom = errors.New("boom")

type MockRegisterer struct {
	MockRegister func(ctx context.Context, secret *corev1.Secret, uid string) (string, error)
	MockVerify   func(token, uid string) (bool, error)
}

func (m *MockRegisterer) Register(ctx context.Context, secret *corev1.Secret, uid string) (string, error) {
	return m.MockRegister(ctx, secret, uid)
}

func (m *MockRegisterer) Verify(token, uid string) (bool, error) {
	return m.MockVerify(token, uid)
}

func TestReconcile(t *testing.T) {
	type args struct {
		kube client.Client
		reg  Registerer
	}
	type want struct {
		err error
		rec reconcile.Result
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"SecretGetError": {
			reason: "We should requeue if entitlement Secret cannot be fetched",
			args: args{
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errGetSecret),
			},
		},
		"KubesystemGetError": {
			reason: "We should requeue if kube-system namespace cannot be fetched",
			args: args{
				kube: &test.MockClient{
					MockGet: func(_ context.Context, key client.ObjectKey, obj client.Object) error {
						switch obj.(type) {
						case *corev1.Secret:
							return nil
						case *corev1.Namespace:
							return errBoom
						}
						return nil
					},
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errGetKubesystemNS),
			},
		},
		"RegisterError": {
			reason: "We should requeue if registration fails",
			args: args{
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
				reg: &MockRegisterer{
					MockRegister: func(_ context.Context, _ *corev1.Secret, _ string) (string, error) {
						return "", errBoom
					},
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errRegister),
			},
		},
		"VerifyError": {
			reason: "We should requeue if verification fails",
			args: args{
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
				reg: &MockRegisterer{
					MockRegister: func(_ context.Context, _ *corev1.Secret, _ string) (string, error) {
						return "", nil
					},
					MockVerify: func(_, _ string) (bool, error) {
						return false, errBoom
					},
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errVerify),
			},
		},
		"CannotVerify": {
			reason: "We should sync again after a while if the token cannot be verified",
			args: args{
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
				reg: &MockRegisterer{
					MockRegister: func(_ context.Context, _ *corev1.Secret, _ string) (string, error) {
						return "", nil
					},
					MockVerify: func(_, _ string) (bool, error) {
						return false, nil
					},
				},
			},
			want: want{
				rec: reconcile.Result{RequeueAfter: syncPeriod},
			},
		},
		"Success": {
			reason: "We should not reconcile if we successfully registered and verified the entitlement",
			args: args{
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
				reg: &MockRegisterer{
					MockRegister: func(_ context.Context, _ *corev1.Secret, _ string) (string, error) {
						return "", nil
					},
					MockVerify: func(_, _ string) (bool, error) {
						return true, nil
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewReconciler(&fake.Manager{Client: tc.args.kube}, WithRegisterer(tc.args.reg))
			_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{}})

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\nReason: %s\nr.Reconcile(...): -want error, +got error:\n%s", tc.reason, diff)
			}
		})
	}
}
