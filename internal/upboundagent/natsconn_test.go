package upboundagent

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
)

var (
	errBoom = errors.New("boom")
)

func Test_fetchCA(t *testing.T) {
	endpoint := "https://foo.com"
	endpointToken := "platform-token"
	defaultResponse := map[string]string{
		"nats_ca": "test-ca",
	}

	type args struct {
		responderErr error
		responseCode int
		responseBody interface{}
	}
	type want struct {
		ca  string
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
				ca:  "test-ca",
				err: nil,
			},
		},
		"ServerError": {
			args: args{
				responseCode: http.StatusInternalServerError,
				responseBody: "some-error",
			},
			want: want{
				err: errors.New("ca bundle request failed with 500 - \"some-error\""),
			},
		},
		"UnexpectedResponseBody": {
			args: args{
				responseCode: http.StatusOK,
				responseBody: "test-ca",
			},
			want: want{
				//err: errors.New("failed to unmarshall nats ca bundle response: json: cannot unmarshal string into Go value of type map[string]string"),
				err: errors.WithStack(errors.New("failed to unmarshall nats ca bundle response: json: cannot unmarshal string into Go value of type map[string]string")),
			},
		},
		"EmptyToken": {
			args: args{
				responseCode: http.StatusOK,
				responseBody: map[string]string{
					"ca": "",
				},
			},
			want: want{
				err: errors.New("empty nats ca bundle received"),
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
				}, "failed to request ca bundle"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rc := newRestyClient(endpoint, false)

			httpmock.ActivateNonDefault(rc.GetClient())

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

			httpmock.RegisterResponder(http.MethodGet, endpoint+natsCAPath, responder)

			got, gotErr := fetchCABundle(rc, logging.NewNopLogger(), endpointToken)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("fetchCABundle(...): -want error, +got error: %s", diff)
			}
			if diff := cmp.Diff(tc.want.ca, got); diff != "" {
				t.Errorf("fetchCABundle(...): -want result, +got result: %s", diff)
			}
		})
	}
}

func Test_fetchNewJWT(t *testing.T) {
	endpoint := "https://foo.com"
	endpointToken := "platform-token"
	clusterID := uuid.New()
	defaultResponse := map[string]string{
		"token": "test-ca",
	}

	type args struct {
		responderErr error
		responseCode int
		responseBody interface{}
	}
	type want struct {
		jwt string
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
				jwt: "test-ca",
				err: nil,
			},
		},
		"ServerError": {
			args: args{
				responseCode: http.StatusInternalServerError,
				responseBody: "some-error",
			},
			want: want{
				err: errors.New("new token request failed with 500 - \"some-error\""),
			},
		},
		"UnexpectedResponseBody": {
			args: args{
				responseCode: http.StatusOK,
				responseBody: "test-ca",
			},
			want: want{
				err: errors.WithStack(errors.New("failed to unmarshall nats token response: json: cannot unmarshal string into Go value of type map[string]string")),
			},
		},
		"EmptyToken": {
			args: args{
				responseCode: http.StatusOK,
				responseBody: map[string]string{
					"token": "",
				},
			},
			want: want{
				err: errors.New("empty token received"),
			},
		},
		"RestyTransportErr": {
			args: args{
				responderErr: errBoom,
			},
			want: want{
				err: errors.Wrap(&url.Error{
					Op:  "Post",
					URL: "https://foo.com/v1/nats/token",
					Err: errBoom,
				}, "failed to request new token"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rc := newRestyClient(endpoint, false)

			httpmock.ActivateNonDefault(rc.GetClient())

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

			httpmock.RegisterResponder(http.MethodPost, endpoint+natsTokenPath, responder)

			got, gotErr := fetchNewJWTToken(rc, logging.NewNopLogger(), endpointToken, clusterID.String(), "some-public-key")
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("fetchNewJWTToken(...): -want error, +got error: %s", diff)
			}
			if diff := cmp.Diff(tc.want.jwt, got); diff != "" {
				t.Errorf("fetchNewJWTToken(...): -want result, +got result: %s", diff)
			}
		})
	}
}

func Test_isTokenValid(t *testing.T) {
	type args struct {
		jwt string
	}
	type want struct {
		valid bool
	}
	cases := map[string]struct {
		args
		want
	}{
		"Valid": {
			args: args{
				// Token with no expiration
				jwt: "eyJ0eXAiOiJqd3QiLCJhbGciOiJlZDI1NTE5In0.eyJqdGkiOiJNS1Q3TUdYSlFLVTRSNk1GUUg0QUdVR1NWNjJXQ1g1Q1NVQ1ZDTFZYN0ZMNkNITjZWSkFRIiwiaWF0IjoxNjEzNTY0NTk1LCJpc3MiOiJBREJKSEdZNEtYSjU1NVJDUEMySE9DTEpTSllIMlBGTVU0WllPR1JFWFBJRzJHRFNWQ1FIWFJWNyIsIm5hbWUiOiJ0ZXN0LXBsYXRmb3JtIiwic3ViIjoiVUE3U0U1SEs0TkxSVFRVVkxRM0NDSFMyVDcyNUhUQUNTRTRZSUhFWUpXT0RJSlBVQjRUS0NLUjIiLCJ0eXBlIjoidXNlciIsIm5hdHMiOnsicHViIjp7ImFsbG93IjpbInBsYXRmb3Jtcy50ZXN0LXBsYXRmb3JtLmhlYWx0aCJdfSwic3ViIjp7ImFsbG93IjpbInBsYXRmb3Jtcy50ZXN0LXBsYXRmb3JtLmdhdGV3YXkiXX0sInJlc3AiOnsibWF4IjozMDAsInR0bCI6NjAwMDAwMDAwMDAwfX19.5_cKm0CIQzRtklrI0UYYrtgrEZzd1rMU5XWWU8kS26ftOeE7HhX-CntyVFZbggmBR7cRJ7r-NM1N4TJgS2jVDw",
			},
			want: want{
				valid: true,
			},
		},
		"Expired": {
			args: args{
				// Token already expired
				jwt: "eyJ0eXAiOiJqd3QiLCJhbGciOiJlZDI1NTE5In0.eyJleHAiOjE2MTM0ODQwMDUsImp0aSI6IkdBU1gzSElJRDJNMlg3WUNPU1lYSFJWNkNYMlozR0YyM1JBUDQ1M0JVSjJIMlgzUUZTNVEiLCJpYXQiOjE2MTMzOTc2MDUsImlzcyI6IkFEVE1NRzdRVjdKVkFVNVpYSUlRSEFGWENDT1QzSkxVSko1TVFPSkE2RlFLWFhPUlNIV1ZQU05KIiwibmFtZSI6ImZhZTU3MzA3LTUyNTktNDc5Ni1iOGRiLWQwNWMwZjllOWE3NSIsInN1YiI6IlVCNUxISUVCNkhCNkhKNldJM1hRSVpERjc1NkpZNjJGVTRGWFc3NFVFQVlJWUQyNjU0RkpHWVpFIiwidHlwZSI6InVzZXIiLCJuYXRzIjp7InB1YiI6eyJhbGxvdyI6WyJwbGF0Zm9ybXMuZmFlNTczMDctNTI1OS00Nzk2LWI4ZGItZDA1YzBmOWU5YTc1LmhlYWx0aCJdfSwic3ViIjp7ImFsbG93IjpbInBsYXRmb3Jtcy5mYWU1NzMwNy01MjU5LTQ3OTYtYjhkYi1kMDVjMGY5ZTlhNzUuZ2F0ZXdheSJdfSwicmVzcCI6eyJtYXgiOjMwMCwidHRsIjo2MDAwMDAwMDAwMDB9fX0.CdOx8rPfLNHydi_4Cfyx9zzAH7k8GK39qzkVfTWBioH4jVqNAOM3tIILd9TB-HAOblLjkV2yGTp3Db0eRMlpAA",
			},
			want: want{
				valid: false,
			},
		},
		"Invalid": {
			args: args{
				jwt: "invalid",
			},
			want: want{
				valid: false,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			got := isJWTValid(tc.jwt, logging.NewNopLogger())
			if diff := cmp.Diff(tc.want.valid, got); diff != "" {
				t.Errorf("isJWTValid(...): -want result, +got result: %s", diff)
			}
		})
	}
}
