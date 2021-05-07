package upbound

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
)

func Test_GetGatewayCerts(t *testing.T) {
	errBoom := errors.New("boom")

	endpoint := "https://foo.com"
	endpointToken := "platform-token"
	defaultResponse := map[string]string{
		keyNATSCA:       "test-ca",
		keyJWTPublicKey: "test-jwt-public-key",
	}

	type args struct {
		responderErr error
		responseCode int
		responseBody interface{}
	}
	type want struct {
		out PublicCerts
		err error
	}
	cases := map[string]struct {
		args
		want
	}{
		"Success": {
			args: args{
				responseCode: http.StatusOK,
				responseBody: defaultResponse,
			},
			want: want{
				out: PublicCerts{
					JWTPublicKey: "test-jwt-public-key",
					NATSCA:       "test-ca",
				},
				err: nil,
			},
		},
		"ServerError": {
			args: args{
				responseCode: http.StatusInternalServerError,
				responseBody: "some-error",
			},
			want: want{
				err: errors.New("gateway certs request failed with 500 - \"some-error\""),
			},
		},
		"UnexpectedResponseBody": {
			args: args{
				responseCode: http.StatusOK,
				responseBody: "test-ca",
			},
			want: want{
				err: errors.WithStack(errors.New("failed to unmarshall gw certs response: json: cannot unmarshal string into Go value of type map[string]string")),
			},
		},
		"EmptyCerts": {
			args: args{
				responseCode: http.StatusOK,
				responseBody: map[string]string{},
			},
			want: want{
				err: errors.New("empty jwt public key received"),
			},
		},
		"RestyTransportErr": {
			args: args{
				responderErr: errBoom,
			},
			want: want{
				err: errors.Wrap(&url.Error{
					Op:  "Get",
					URL: "https://foo.com/v1/gw/certs",
					Err: errBoom,
				}, "failed to request gateway certs"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rc := NewClient(endpoint, false)

			httpmock.ActivateNonDefault(rc.(*client).resty.GetClient())

			b, err := json.Marshal(tc.responseBody)
			if err != nil {
				t.Errorf("cannot unmarshal tc.responseBody %v", err)
			}

			var responder httpmock.Responder
			if tc.responderErr != nil {
				responder = httpmock.NewErrorResponder(tc.responderErr)
			} else {
				responder = httpmock.NewStringResponder(tc.responseCode, string(b))
			}

			httpmock.RegisterResponder(http.MethodGet, endpoint+gwCertsPath, responder)

			got, gotErr := rc.GetGatewayCerts(endpointToken)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("GetGatewayCerts(...): -want error, +got error: %s", diff)
			}
			if diff := cmp.Diff(tc.want.out, got); diff != "" {
				t.Errorf("GetGatewayCerts(...): -want result, +got result: %s", diff)
			}
		})
	}
}
