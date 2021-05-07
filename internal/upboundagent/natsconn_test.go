/*
Copyright 2021 Upbound Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package upboundagent

import (
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/google/go-cmp/cmp"
)

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
