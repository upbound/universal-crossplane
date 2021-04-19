package upboundagent

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v4"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/transport"

	"github.com/upbound/universal-crossplane/internal/upboundagent/internal"
)

const (
	validPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnzyis1ZjfNB0bBgKFMSv
vkTtwlvBsaJq7S5wA+kzeVOVpVWwkWdVha4s38XM/pa/yr47av7+z3VTmvDRyAHc
aT92whREFpLv9cj5lTeJSibyr/Mrm/YtjCZVWgaOYIhwrXwKLqPr/11inWsAkfIy
tvHWTxZYEcXLgAXFuUuaS3uF9gEiNQwzGTU1v0FqkqTBr4B8nW3HCN47XUu0t8Y0
e+lf4s4OxQawWD79J9/5d3Ry0vbV3Am1FtGJiJvOwRsIfVChDpYStTcHTCMqtvWb
V6L11BWkpzGXSW4Hv43qa+GSYOD2QU68Mb59oSk2OB+BtOLpJofmbGEGgvmwyCI9
MwIDAQAB
-----END PUBLIC KEY-----`
	validJWTToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3RVc2VyIFRlc3RVc2VybmFtZSIsImV4cCI6MTA0MTM3OTU2MDAsImF1ZCI6ImMyMTU2MWRhLTA4N2ItNGVmYy1hZjZiLTcxOGU5OWJmZDg1ZiIsInBheWxvYWQiOnsiaXNPd25lciI6ZmFsc2UsInRlYW1JZHMiOlsidGVzdDEiLCJ0ZXN0MiJdLCJpZGVudGlmaWVyIjoidGVzdCIsImlkZW50aWZpZXJLaW5kIjoidXNlcklEIn19.OdDMFf54hd9BrK12FfF13VQax32plIrOoEoJr0h7LmtGNofXXljoa6uZL9MB65CQ3a_KacFpyeP-YPvYbIJngq6QK4w2gQnERZiiW9oypihu-f_sTeo3N-HFn4ZC5i4HFJl5c0JHacWRwpotLJovQGCi0IWrp6HvWlpMROQYEGLGzG67TNpYZlNc6AIqd4jnhZGmiEvbebYCBow8HwZ7i1bwsf9cskSdNuPIMFQqW8f5KYmXqw9PYYg9_b3inES3qt8IQXZf_PfigsN9ffh5pR3ybiQNUwKhrtpi0HYWvXiYY8viyW5_xJ09QkHDvP7unTkpXjQ7B9Wz9mtQ6-nFMw"
)

//   Private key for future reference
//  -----BEGIN RSA PRIVATE KEY-----
//  MIIEogIBAAKCAQEAnzyis1ZjfNB0bBgKFMSvvkTtwlvBsaJq7S5wA+kzeVOVpVWw
//  kWdVha4s38XM/pa/yr47av7+z3VTmvDRyAHcaT92whREFpLv9cj5lTeJSibyr/Mr
//  m/YtjCZVWgaOYIhwrXwKLqPr/11inWsAkfIytvHWTxZYEcXLgAXFuUuaS3uF9gEi
//  NQwzGTU1v0FqkqTBr4B8nW3HCN47XUu0t8Y0e+lf4s4OxQawWD79J9/5d3Ry0vbV
//  3Am1FtGJiJvOwRsIfVChDpYStTcHTCMqtvWbV6L11BWkpzGXSW4Hv43qa+GSYOD2
//  QU68Mb59oSk2OB+BtOLpJofmbGEGgvmwyCI9MwIDAQABAoIBACiARq2wkltjtcjs
//  kFvZ7w1JAORHbEufEO1Eu27zOIlqbgyAcAl7q+/1bip4Z/x1IVES84/yTaM8p0go
//  amMhvgry/mS8vNi1BN2SAZEnb/7xSxbflb70bX9RHLJqKnp5GZe2jexw+wyXlwaM
//  +bclUCrh9e1ltH7IvUrRrQnFJfh+is1fRon9Co9Li0GwoN0x0byrrngU8Ak3Y6D9
//  D8GjQA4Elm94ST3izJv8iCOLSDBmzsPsXfcCUZfmTfZ5DbUDMbMxRnSo3nQeoKGC
//  0Lj9FkWcfmLcpGlSXTO+Ww1L7EGq+PT3NtRae1FZPwjddQ1/4V905kyQFLamAA5Y
//  lSpE2wkCgYEAy1OPLQcZt4NQnQzPz2SBJqQN2P5u3vXl+zNVKP8w4eBv0vWuJJF+
//  hkGNnSxXQrTkvDOIUddSKOzHHgSg4nY6K02ecyT0PPm/UZvtRpWrnBjcEVtHEJNp
//  bU9pLD5iZ0J9sbzPU/LxPmuAP2Bs8JmTn6aFRspFrP7W0s1Nmk2jsm0CgYEAyH0X
//  +jpoqxj4efZfkUrg5GbSEhf+dZglf0tTOA5bVg8IYwtmNk/pniLG/zI7c+GlTc9B
//  BwfMr59EzBq/eFMI7+LgXaVUsM/sS4Ry+yeK6SJx/otIMWtDfqxsLD8CPMCRvecC
//  2Pip4uSgrl0MOebl9XKp57GoaUWRWRHqwV4Y6h8CgYAZhI4mh4qZtnhKjY4TKDjx
//  QYufXSdLAi9v3FxmvchDwOgn4L+PRVdMwDNms2bsL0m5uPn104EzM6w1vzz1zwKz
//  5pTpPI0OjgWN13Tq8+PKvm/4Ga2MjgOgPWQkslulO/oMcXbPwWC3hcRdr9tcQtn9
//  Imf9n2spL/6EDFId+Hp/7QKBgAqlWdiXsWckdE1Fn91/NGHsc8syKvjjk1onDcw0
//  NvVi5vcba9oGdElJX3e9mxqUKMrw7msJJv1MX8LWyMQC5L6YNYHDfbPF1q5L4i8j
//  8mRex97UVokJQRRA452V2vCO6S5ETgpnad36de3MUxHgCOX3qL382Qx9/THVmbma
//  3YfRAoGAUxL/Eu5yvMK8SAt/dJK6FedngcM3JEFNplmtLYVLWhkIlNRGDwkg3I5K
//  y18Ae9n7dHVueyslrb6weq7dTkYDi3iOYRW8HRkIQh06wEdbxt0shTzAJvvCQfrB
//  jg/3747WSsf/zBTcHihTRBdAv6OmdhV4/dD5YBfLAkLrd+mX7iE=
//  -----END RSA PRIVATE KEY-----

// TestMain - Configure gin to not output debug output during tests.
func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.FatalLevel)
	os.Exit(m.Run())
}

func Test_impersonationConfigForUser(t *testing.T) {
	type args struct {
		u internal.CrossplaneAccessor
	}
	type want struct {
		out transport.ImpersonationConfig
		err error
	}
	cases := map[string]struct {
		args
		want
	}{
		"owner": {
			args: args{
				u: internal.CrossplaneAccessor{
					IsOwner:    true,
					TeamIDs:    nil,
					Identifier: "test",
				},
			},
			want: want{
				out: transport.ImpersonationConfig{
					UserName: userUpboundCloud,
					Groups:   []string{groupSystemAuthenticated, groupCrossplaneOwner},
					Extra: map[string][]string{
						keyUpboundUser: {"test"},
					},
				},
				err: nil,
			},
		},
		"userWithOneGroup": {
			args: args{
				u: internal.CrossplaneAccessor{
					IsOwner:    false,
					TeamIDs:    []string{"test-group"},
					Identifier: "test",
				},
			},
			want: want{
				out: transport.ImpersonationConfig{
					UserName: userUpboundCloud,
					Groups:   []string{"test-group", groupSystemAuthenticated},
					Extra: map[string][]string{
						keyUpboundUser: {"test"},
					},
				},
				err: nil,
			},
		},
		"userWithMultipleGroups": {
			args: args{
				u: internal.CrossplaneAccessor{
					IsOwner:    false,
					TeamIDs:    []string{"test-group-1", "test-group-2", "test-group-3"},
					Identifier: "test",
				},
			},
			want: want{
				out: transport.ImpersonationConfig{
					UserName: userUpboundCloud,
					Groups:   []string{"test-group-1", "test-group-2", "test-group-3", groupSystemAuthenticated},
					Extra: map[string][]string{
						keyUpboundUser: {"test"},
					},
				},
				err: nil,
			},
		},
		"ownerWithMultipleGroups": {
			args: args{
				u: internal.CrossplaneAccessor{
					IsOwner:    true,
					TeamIDs:    []string{"test-group-1", "test-group-2", "test-group-3"},
					Identifier: "test",
				},
			},
			want: want{
				out: transport.ImpersonationConfig{
					UserName: userUpboundCloud,
					Groups:   []string{"test-group-1", "test-group-2", "test-group-3", groupSystemAuthenticated, groupCrossplaneOwner},
					Extra: map[string][]string{
						keyUpboundUser: {"test"},
					},
				},
				err: nil,
			},
		},
		"missingUserName": {
			args: args{
				u: internal.CrossplaneAccessor{
					IsOwner:    false,
					TeamIDs:    nil,
					Identifier: "",
				},
			},
			want: want{
				err: errors.New(errUsernameMissing),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, gotErr := impersonationConfigForUser(tc.u)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("impersonationConfigForUser(...): -want error, +got error: %s", diff)
			}
			if diff := cmp.Diff(tc.want.out, got); diff != "" {
				t.Errorf("impersonationConfigForUser(...): -want result, +got result: %s", diff)
			}
		})
	}
}

func TestProxy_roundTrip(t *testing.T) {
	testEnvID := "c21561da-087b-4efc-af6b-718e99bfd85f"
	kubeURL, _ := url.Parse("https://kubehost")
	type args struct {
		publicKey []byte
		req       *http.Request
	}
	type want struct {
		respBody string
		respCode int
	}
	cases := map[string]struct {
		args
		want
	}{
		"UnableToValidateToken_SignedWithWrongKey": {
			args: args{
				publicKey: []byte(`-----BEGIN PUBLIC KEY-----
MIGeMA0GCSqGSIb3DQEBAQUAA4GMADCBiAKBgGVaeGQGnkXJYK8RrBYcbIlrF35X
rOBbDIc8/+IeC/jzkxaOGl7Se3Nx/ewIe8bE24RIWCLeWZO+X4OFHIKWqiRhOD2h
quhz7dONQ0iAI/C8d3iCIi9I6DVWE+7JjZnViEYBjCm830SzUnFDWxGSllxhGrp4
WNF1xiFz8ZOCiTgLAgMBAAE=
-----END PUBLIC KEY-----`),
				req: &http.Request{
					Header: map[string][]string{
						headerAuthorization: {
							fmt.Sprintf("Bearer %s", validJWTToken),
						},
					},
				},
			},
			want: want{
				respCode: http.StatusBadRequest,
				respBody: fmt.Sprintf(`{"message":"%s - %s"}`, errUnableToValidateToken, errors.New("crypto/rsa: verification error")),
			},
		},
		"WrongEnvironmentId": {
			args: args{
				publicKey: []byte(validPublicKey),
				req: &http.Request{
					Header: map[string][]string{
						headerAuthorization: {
							fmt.Sprintf("Bearer %s", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3RVc2VyIFRlc3RVc2VybmFtZSIsImV4cCI6MTA0MTM3OTU2MDAsImF1ZCI6ImFiYy1hYmMiLCJwYXlsb2FkIjp7ImlzT3duZXIiOmZhbHNlLCJ0ZWFtSWRzIjpbInRlc3QxIiwidGVzdDIiXSwidXNlcklkZW50aWZpZXIiOiJ0ZXN0In19.X0_aYr14BcYu-iznkGsxhTYtusQsYhIoQdi5B7QJ3w5-c0Ar2ewBiYlY7gugg_Cy5XXrHgjLbA3Lvj3yUascmCW4AYuZY1frbc4BSPy6LbIUiEJhPMt2VkPNBfyUMvzPUxuC3a3SthxmEO1yJ2k2cUeUWyHN6ODMZSmlj5FbCBw4SlQEZObYZ1xuBPOq3peVli5LYVhdpxZQt37gaHBuGF1dstgZN5hSAC1HudUmKqvS9hvhjs_adqwxyXIAPG8hh2j-OhJIBuZr5tZ2oyDMOGtutf50nHWO5mhM3PHt7WpTU8x6hlEDSJxT8xa0LrO39GscjlgQ3FwKRvx56ccZEA"),
						},
					},
				},
			},
			want: want{
				respCode: http.StatusBadRequest,
				respBody: fmt.Sprintf(`{"message":"%s"}`, fmt.Sprintf(errInvalidEnvID, "abc-abc", testEnvID)),
			},
		},
		"FailedToGetImpersonationConfig": {
			args: args{
				publicKey: []byte(validPublicKey),
				req: &http.Request{
					Header: map[string][]string{
						headerAuthorization: {
							fmt.Sprintf("Bearer %s", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3RVc2VyIFRlc3RVc2VybmFtZSIsImV4cCI6MTA0MTM3OTU2MDAsImF1ZCI6ImMyMTU2MWRhLTA4N2ItNGVmYy1hZjZiLTcxOGU5OWJmZDg1ZiIsInBheWxvYWQiOnsiaXNPd25lciI6ZmFsc2UsInRlYW1JZHMiOlsidGVzdDEiLCJ0ZXN0MiJdfX0.Cb5d92IqM60zl6EBgz3IGoIJ3JVN93RPCZTB0zMUcaJRdDxFi3ppyGhJEgVm_v5ynPuzcc7ejdKrrh_C6wzcPHZqwVlVl-RbBMTCukySHd2VXJgDOOLWSdh8fHJJJCZ1vWH8OxtqdwyWPBEYYZbAj6qdzdWUSxYLVuHailc0G6ABU9OWoc1HvUSqxwZMhDoLz7wYtMgUozQivIixq9ssFFm7_gXyFzcGgRxBx1uLVLRtM2k4tAxLI5229Kf47ZBBfHQVlyThZOocvyhsPVjXZ96HpAatd5UezhyQKskyqt4VCoGPFK00f-cdx2PxbvWV_ZjB0wad57u8yZ15Cj8yvQ"),
						},
					},
				},
			},
			want: want{
				respCode: http.StatusBadRequest,
				respBody: fmt.Sprintf(`{"message":"%s: %s"}`, errFailedToGetImpersonationConfig, errUsernameMissing),
			},
		},
		"Success": {
			args: args{
				publicKey: []byte(validPublicKey),
				req: &http.Request{
					Header: map[string][]string{
						headerAuthorization: {
							fmt.Sprintf("Bearer %s", validJWTToken),
						},
					},
				},
			},
			want: want{
				respBody: "mock success - proxied to: https://kubehost/proxypathinfo",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := &Proxy{
				kubeTransport: mockRoundTripper{},
				config:        &Config{EnvID: testEnvID},
				kubeHost:      kubeURL, // No function just to avoid nil ref
			}
			if tc.publicKey != nil {
				k, err := jwt.ParseRSAPublicKeyFromPEM(tc.publicKey)
				if err != nil {
					t.Fatalf("invalid input public key: %v", err)
				}
				p.config = &Config{
					EnvID:             testEnvID,
					TokenRSAPublicKey: k,
				}
			}
			g := NewGomegaWithT(t)
			rec := httptest.NewRecorder()
			// We need close notify to support the proxy.
			w := CloseNotifyWrapper{rec}

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "https://proxy.tgw.upbound.io/k8s/proxypathinfo", nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			e.Any(k8sHandlerPath, p.k8s())

			req.Header = tc.req.Header
			e.ServeHTTP(rec, req)

			if tc.want.respCode == 0 {
				tc.want.respCode = http.StatusOK
			}

			if tc.want.respBody != "" {
				wantBody := tc.want.respBody
				if w.Code != http.StatusOK {
					wantBody += "\n" // Proxy adds `/n` for internal responses / errors
				}
				g.Expect(w.Body.String()).To(Equal(wantBody))
			}
			g.Expect(w.Code).To(Equal(tc.want.respCode))
		})
	}
}

type CloseNotifyWrapper struct {
	*httptest.ResponseRecorder
}

func (CloseNotifyWrapper) CloseNotify() <-chan bool {
	closer := make(chan bool, 1)
	return closer
}

func TestProxy_reviewToken(t *testing.T) {
	type args struct {
		publicKey []byte
		req       *http.Request
	}
	type want struct {
		out *TokenClaims
		err error
	}
	cases := map[string]struct {
		args
		want
	}{
		"MissingAuthHeader": {
			args: args{
				req: &http.Request{},
			},
			want: want{
				err: errors.New(errMissingAuthHeader),
			},
		},
		"MissingBearer": {
			args: args{
				req: &http.Request{
					Header: map[string][]string{
						headerAuthorization: {
							"Basic YWxhZGRpbjpvcGVuc2VzYW1l",
						},
					},
				},
			},
			want: want{
				err: errors.New(errMissingBearer),
			},
		},
		"UnexpectedSigningMethod": {
			args: args{
				req: &http.Request{
					Header: map[string][]string{
						headerAuthorization: {
							"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJwYXlsb2FkIjp7ImlzT3duZXIiOmZhbHNlLCJ0ZWFtSURzIjpbInRlYW0tdXVpZC0xIiwidGVhbS11dWlkLTIiXSwidXNlcklkZW50aWZpZXIiOiJ1c2VybmFtZS1vci1yb2JvdG5hbWUifX0.gaKQk4Ysq7GHjuFd9xRSy4GrASSLfQ6U1-T1414Bnkg",
						},
					},
				},
			},
			want: want{
				err: &jwt.ValidationError{
					Inner: errors.New(fmt.Sprintf(errUnexpectedSigningMethod, "HS256")),
				},
			},
		},
		"SignedWithWrongKey": {
			args: args{
				publicKey: []byte(`-----BEGIN PUBLIC KEY-----
MIGeMA0GCSqGSIb3DQEBAQUAA4GMADCBiAKBgGVaeGQGnkXJYK8RrBYcbIlrF35X
rOBbDIc8/+IeC/jzkxaOGl7Se3Nx/ewIe8bE24RIWCLeWZO+X4OFHIKWqiRhOD2h
quhz7dONQ0iAI/C8d3iCIi9I6DVWE+7JjZnViEYBjCm830SzUnFDWxGSllxhGrp4
WNF1xiFz8ZOCiTgLAgMBAAE=
-----END PUBLIC KEY-----`),
				req: &http.Request{
					Header: map[string][]string{
						headerAuthorization: {
							fmt.Sprintf("Bearer %s", validJWTToken),
						},
					},
				},
			},
			want: want{
				err: &jwt.ValidationError{
					Inner: errors.New("crypto/rsa: verification error"),
				},
			},
		},
		"Success": {
			args: args{
				publicKey: []byte(validPublicKey),
				req: &http.Request{
					Header: map[string][]string{
						headerAuthorization: {
							fmt.Sprintf("Bearer %s", validJWTToken),
						},
					},
				},
			},
			want: want{
				err: nil,
				out: &TokenClaims{
					User: internal.CrossplaneAccessor{
						TeamIDs:        []string{"test1", "test2"},
						Identifier:     "test",
						IsOwner:        false,
						IdentifierKind: "userID",
					},
					StandardClaims: jwt.StandardClaims{
						Audience:  "c21561da-087b-4efc-af6b-718e99bfd85f",
						ExpiresAt: 10413795600,
						Subject:   "1234567890",
					},
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := &Proxy{}
			if tc.publicKey != nil {
				k, err := jwt.ParseRSAPublicKeyFromPEM(tc.publicKey)
				if err != nil {
					t.Fatalf("invalid input public key: %v", err)
				}
				p.config = &Config{
					TokenRSAPublicKey: k,
				}
			}
			got, gotErr := p.reviewToken(tc.args.req.Header)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("reviewToken(...): -want error, +got error: %s", diff)
			}
			if diff := cmp.Diff(tc.want.out, got); diff != "" {
				t.Errorf("reviewToken(...): -want result, +got result: %s", diff)
			}
		})
	}
}

type mockRoundTripper struct {
}

func (mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	body := ioutil.NopCloser(bytes.NewReader([]byte(fmt.Sprintf("mock success - proxied to: %+v", r.URL))))
	return &http.Response{StatusCode: http.StatusOK, Body: body, Request: r}, nil
}
