package ubccerts

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/upbound/universal-crossplane/internal/clients/upbound"
	upboundmocks "github.com/upbound/universal-crossplane/internal/clients/upbound/mocks"
)

func TestReconcile(t *testing.T) {
	errBoom := errors.New("boom")

	type args struct {
		mgr           manager.Manager
		setupUpClient func(ctrl *gomock.Controller) upbound.Client
		req           reconcile.Request
	}
	type want struct {
		err error
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"ErrGetSecret": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(errBoom)},
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNamePublicCerts}},
			},
			want: want{
				err: errors.Wrap(errBoom, errGetSecret),
			},
		},
		"ErrNoTokenSecret": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						if key.Name == secretNameCPToken {
							return kerrors.NewNotFound(schema.GroupResource{}, secretNameCPToken)
						}
						return nil
					}},
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNamePublicCerts}},
			},
			want: want{
				err: errors.Wrapf(kerrors.NewNotFound(schema.GroupResource{}, secretNameCPToken), errGetCPTokenSecret, secretNameCPToken),
			},
		},
		"ErrTokenSecretHasNoToken": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(nil)},
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNamePublicCerts}},
			},
			want: want{
				err: errors.Errorf(errNoTokenInSecret, keyToken, secretNameCPToken),
			},
		},
		"ErrFailedToFetch": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						if key.Name == secretNameCPToken {
							s := obj.(*corev1.Secret)
							s.Data = map[string][]byte{
								keyToken: []byte("a-token"),
							}
						}
						return nil
					}},
				},
				setupUpClient: func(ctrl *gomock.Controller) upbound.Client {
					upc := upboundmocks.NewMockClient(ctrl)
					upc.EXPECT().GetGatewayCerts(gomock.Any()).Return(upbound.PublicCerts{}, errBoom)
					return upc
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNamePublicCerts}},
			},
			want: want{
				err: errors.Wrap(errBoom, errFetch),
			},
		},
		"ErrFailedToUpdateSecret": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{
						MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
							if key.Name == secretNameCPToken {
								s := obj.(*corev1.Secret)
								s.Data = map[string][]byte{
									keyToken: []byte("a-token"),
								}
							}
							return nil
						},
						MockUpdate: func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
							return errBoom
						},
					},
				},
				setupUpClient: func(ctrl *gomock.Controller) upbound.Client {
					upc := upboundmocks.NewMockClient(ctrl)
					upc.EXPECT().GetGatewayCerts(gomock.Any()).Return(upbound.PublicCerts{
						JWTPublicKey: "jwt-public-key",
						NATSCA:       "mock-nats-ca",
					}, nil)
					return upc
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNamePublicCerts}},
			},
			want: want{
				err: errors.Wrap(errBoom, errUpdateSecret),
			},
		},
		"SuccessfulFetch": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{
						MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
							if key.Name == secretNameCPToken {
								s := obj.(*corev1.Secret)
								s.Data = map[string][]byte{
									keyToken: []byte("a-token"),
								}
							}
							return nil
						},
						MockUpdate: func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
							s := obj.(*corev1.Secret)
							j := string(s.Data[keyJWTPublicKey])
							n := string(s.Data[keyNATSCA])
							if j != "jwt-public-key" {
								t.Errorf("unexpected secret content for jwt public key: %s", j)
							}
							if n != "nats-ca" {
								t.Errorf("unexpected secret content for nats ca: %s", n)
							}
							return nil
						},
					},
				},
				setupUpClient: func(ctrl *gomock.Controller) upbound.Client {
					upc := upboundmocks.NewMockClient(ctrl)
					upc.EXPECT().GetGatewayCerts(gomock.Any()).Return(upbound.PublicCerts{
						JWTPublicKey: "jwt-public-key",
						NATSCA:       "nats-ca",
					}, nil)
					return upc
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNamePublicCerts}},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var upc upbound.Client
			if tc.args.setupUpClient != nil {
				upc = tc.args.setupUpClient(ctrl)
			}
			r := NewReconciler(tc.args.mgr, upc)
			_, err := r.Reconcile(context.Background(), tc.args.req)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n\nr.Reconcile(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}
