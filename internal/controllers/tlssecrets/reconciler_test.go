package tlssecrets

import (
	"context"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testCACert = `
-----BEGIN CERTIFICATE-----
MIIDBTCCAe2gAwIBAgIBADANBgkqhkiG9w0BAQsFADAkMRAwDgYDVQQKEwd1cGJv
dW5kMRAwDgYDVQQDEwd1cGJvdW5kMB4XDTIxMDMyOTIwNTkzOFoXDTMxMDMyNzIw
NTkzOFowJDEQMA4GA1UEChMHdXBib3VuZDEQMA4GA1UEAxMHdXBib3VuZDCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMApbX2UETPhjDd662M+B0TndpQg
CazZlgo392pwa5tv/KJQIMp31looxMt33YQ++1zNxcwtvIvvQGbrY4JMCx8BDj3F
MIVac1NLdnQgtANTWVQYDw4BINx+b7OS9z5oiRjIvz84bAnLTgnpGr0Us9nRLJRF
9MXW8gVXL5HpIX17Q/mTPEFbDb48dgAoaCiWVH0uUrJfp1XBa42HuuoCCYJt/ZbI
EuI7nXfo8P0Ty8spVL28hdGp9B6R9CxiapvmSB/z0K9jB7A9AeGYMyK2aECweTwv
m+ZVYXfDk/yXNC7uStexHU0riCXLbYDB5TU3Dxh9nMjYLwgqm1HWok+nKzECAwEA
AaNCMEAwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYE
FLvXLxDo8bXsZHQ1UJkBQ69wX5OPMA0GCSqGSIb3DQEBCwUAA4IBAQC84vlise+7
FhNZ078tmCElUTwS5+9bGv6Q4GuXEXamCyEtKHo6tMRs25vIJO/8oTnYkvizCQ2t
czil3iZMRB5umlVWDaW1Ii1v+44d2nJ+txCDD6v9BD2Xpcq7SUUGvij5JGw/ZDzd
wcZX2/Gpo14tN5qwomL/rMCa1e218G3OsFQW9/ZrKVu33FzytK5y+MNhyxyBOlmm
sb+VTZdrWHUNH2upN6JUaOxBdgaKH1lG7smH6e4WK0DMjylN73GBFdBGcxAHEizi
5R1Q4wydyNbxVnTaYbxoNvr6NV0hfVASfJg/JL3Kr7EWyV2mrmJ7WpxnjmyiwYHi
x6lEqoLcaFMQ
-----END CERTIFICATE-----
`
	testCAKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAwCltfZQRM+GMN3rrYz4HROd2lCAJrNmWCjf3anBrm2/8olAg
ynfWWijEy3fdhD77XM3FzC28i+9AZutjgkwLHwEOPcUwhVpzU0t2dCC0A1NZVBgP
DgEg3H5vs5L3PmiJGMi/PzhsCctOCekavRSz2dEslEX0xdbyBVcvkekhfXtD+ZM8
QVsNvjx2AChoKJZUfS5Ssl+nVcFrjYe66gIJgm39lsgS4judd+jw/RPLyylUvbyF
0an0HpH0LGJqm+ZIH/PQr2MHsD0B4ZgzIrZoQLB5PC+b5lVhd8OT/Jc0Lu5K17Ed
TSuIJcttgMHlNTcPGH2cyNgvCCqbUdaiT6crMQIDAQABAoIBAQC1szaNxMFTblUY
bMlAqPlUpQzR2U1svL2L4gm4Ap8tdgHLNLsc152+2MfkoO27y5YA1a3Pd/vN0afy
6WbJYMAvS988d0V/At0DiNpzyiyM7HYN90Xc9yIse/2BLllNEKl53vA/hklaJXwg
EOOwoG/DaW+esFtX6vwkIqGfdXKuY4lBKEuFWzA/m29HbCHPUapomXrDUSzVMN1v
2HGWYpAPZxQVZuiVFf5kp527E8vOXes8y/lK1tDkEe1zmdZLKdR0+RYgFaPAGnEM
yjNSIS/qMKo1UResPGKmPguRWstTlSJlaHwNKOfVBQwQWfAX0QXiUQzKuYZmHAaA
1xj57/s1AoGBAOcRc/ttjlZQv738BGzhrXjMFq91i03voEhgKhhXTj4VoxeaJBKr
5YR0iIsQSQMWDGZPlCtP6Z728hLRco+HORBy7thL+E0xycVTieZQToFuKU5zUk7d
n3Kb5xVkQJrkUkJlN1O+2yUS+yn+9hp61cmVqeupCU8KQylmrOqnaz+LAoGBANTl
TnkXF6K3F55x/ScD44rmFzTeL7D4+pRLzu26CKypLBBdprcL4O8ElTI0c7z2S2uV
RR41yckj1Cx4AqpEiO52EtV1jrpBYiqM9cgUgWYKx1zco4cNzubhDA5+/yLyQouV
Hp7QGyZK/zi3BnbMLBBxCediDZax/DJ1DWlxZNezAoGABdFIrEHb3Yx251+a9OrR
pULuJ0i8Ux//VxMkvCwmiiWdT5DP67BsPON6NJYaYHuDoGfMgTKn3Rq2iYbAbaCn
7SQXo1Z2T+s6+z7ZL/VBpLyTSahZoCawRwBp1v4JKl0pPQazV+ZsOgi6ThpfM9d3
3nVoK8i7tUO64SX2oInKh3UCgYARjgb2fSz5wdc0vXl+ahetMGPhfCC6mw0uhUG+
4IQumJSFlPNWTKhzjREwXprcjgKSEHDumMjWyRmJwSuXFqej4iCTcWofeZy6nXz2
zpoM6/6cbaUeUckpyIzR9S7cltVd5SHtPoO+mJiK+KyTxyorAOcsKS2tq2d8UaKV
e0AxeQKBgEAzxl3zSqp64sgf/6N8thO13KQgI+9p8O8XqMj5xjInFo4Rt/qPyhMl
DvjepcqqCb9z2XqYfgeoJD4bIWIfshGBvIHug13V1Ui9dKEsA4++PrpI8eC6rmhK
oKJYxoAih+ITK2h8IKIJXWJsikaYRzbu9BW3vE8U7nM2yN5t5+E/
-----END RSA PRIVATE KEY-----
`
)

func TestReconcile(t *testing.T) {
	errBoom := errors.New("boom")

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
		"ErrGetSecret": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(errBoom)},
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNameGatewayTLS}},
			},
			want: want{
				err: errors.Wrap(errBoom, errGetSecret),
			},
		},
		"SecretHasCert": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
						s := obj.(*corev1.Secret)
						s.Data = map[string][]byte{
							keyTLSCert: []byte("cert-key-data"),
						}
						return nil
					})},
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNameGatewayTLS}},
			},
		},
		"ErrGetCASecret": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						if key.Name == secretNameCA {
							return errBoom
						}
						return nil
					}},
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNameGatewayTLS}},
			},
			want: want{
				err: errors.Wrap(errors.Wrap(errBoom, errGetCASecret), errInitCA),
			},
		},
		"ErrUpdateCASecret": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{
						MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
							return nil
						},
						MockUpdate: func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
							return errBoom
						},
					},
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNameGatewayTLS}},
			},
			want: want{
				err: errors.Wrap(errors.Wrap(errBoom, errUpdateCASecret), errInitCA),
			},
		},
		"FailedToUpdate": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{
						MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
							if key.Name == secretNameCA {
								s := obj.(*corev1.Secret)
								s.Data = map[string][]byte{
									keyCACert:  []byte(testCACert),
									keyTLSCert: []byte(testCACert),
									keyTLSKey:  []byte(testCAKey),
								}
							}
							return nil
						},
						MockUpdate: func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
							return errBoom
						},
					},
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNameGatewayTLS}},
			},
			want: want{
				err: errors.Wrap(errBoom, errUpdateCertSecret),
			},
		},
		"SuccessfulGenerate": {
			args: args{
				mgr: &fake.Manager{
					Client: &test.MockClient{
						MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
							if key.Name == secretNameCA {
								s := obj.(*corev1.Secret)
								s.Data = map[string][]byte{
									keyCACert:  []byte(testCACert),
									keyTLSCert: []byte(testCACert),
									keyTLSKey:  []byte(testCAKey),
								}
							}
							return nil
						},
						MockUpdate: func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
							s := obj.(*corev1.Secret)
							ca := string(s.Data[keyCACert])
							cert := string(s.Data[keyTLSCert])
							key := string(s.Data[keyTLSKey])

							diff := cmp.Diff(strings.TrimSpace(ca), strings.TrimSpace(testCACert))
							if diff != "" {
								t.Errorf("unexpected ca cert: %s", diff)
							}
							if cert == "" {
								t.Errorf("missing tls cert")
							}
							if key == "" {
								t.Errorf("missing tls private key")
							}
							return nil
						},
					},
				},
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: secretNameGatewayTLS}},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewReconciler(tc.args.mgr)
			_, err := r.Reconcile(context.Background(), tc.args.req)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n\nr.Reconcile(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}
