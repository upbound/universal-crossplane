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

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/test"
)

const (
	signature = "some sig"

	testPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnzyis1ZjfNB0bBgKFMSv
vkTtwlvBsaJq7S5wA+kzeVOVpVWwkWdVha4s38XM/pa/yr47av7+z3VTmvDRyAHc
aT92whREFpLv9cj5lTeJSibyr/Mrm/YtjCZVWgaOYIhwrXwKLqPr/11inWsAkfIy
tvHWTxZYEcXLgAXFuUuaS3uF9gEiNQwzGTU1v0FqkqTBr4B8nW3HCN47XUu0t8Y0
e+lf4s4OxQawWD79J9/5d3Ry0vbV3Am1FtGJiJvOwRsIfVChDpYStTcHTCMqtvWb
V6L11BWkpzGXSW4Hv43qa+GSYOD2QU68Mb59oSk2OB+BtOLpJofmbGEGgvmwyCI9
MwIDAQAB
-----END PUBLIC KEY-----`
	correctToken         = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3RVc2VyIFRlc3RVc2VybmFtZSIsImV4cCI6MTc0NjM5NzkxNSwiYXVkIjoiYzIxNTYxZGEtMDg3Yi00ZWZjLWFmNmItNzE4ZTk5YmZkODVmIiwicHJvZHVjdENvZGUiOiIxZnN6dnU1Mjd3YW92cWV1aHBreXgyYjVkIiwibm9uY2UiOiJjMjE1NjFkYS0wODdiLTRlZmMtYWY2Yi1zZGFzMjMyYXNkIiwicHVibGljS2V5VmVyc2lvbiI6MX0.V-b6HOHhmTANLpZObDK_XIbozI9WXGT-9yeMASgr3yke1PQiVLyToSQK2KAsTFJW1KHj3Vm4wxTUFNVxYJ6D163wUlb4ZLb3fewj1jBTeCx3VYm4jnZRvWoF45wMYjEVxzMhYk-T5MIzdu1SAdifTuyiXQpiCStpmPw9ZzmCDnVkmoEcrCdFj_b25c5Jgx4uHqaageATAePqqj75AXOjsZeeZ8DeCY7EjrliUP3Z48nGR9yJ01SXK_g48eAvTAG_cVPN7BQcPYJdQeKo9XuWa1VbrUJVrlOIqJspOJwza1CbEhaeoTT37Dy1fZYY41YZllW7c4laRtkK4Tua3vAypA"
	differentToken       = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3RVc2VyIFRlc3RVc2VybmFtZSIsImV4cCI6MTc0NjM5NzkxNSwiYXVkIjoiYzIxNTYxZGEtMDg3Yi00ZWZjLWFmNmItNzE4ZTk5YmZkODVmIiwicHJvZHVjdENvZGUiOiJyYW5kIiwibm9uY2UiOiIyMzEiLCJwdWJsaWNLZXlWZXJzaW9uIjo5OTl9.KmICGZYewaAPEauOjRkv_yGuNUjVDzoqUNLXchMKGTgSnQaoP1GmQJBMWVdG6Lx2ZsrJQqK4qXYU4QN7PhAWF8GDBSl7KBLfyvspTpJMNJduGv49TBlgK9Hd8DMUEQA-MeaFfwM4amc7ESvb0lJoDlgl2l7uZhsrSd55aQRQbMGqIMQv3Ggn7sEAvoZ021ZxiYLq55fkpf68hCk1LDH6HaNJO43bAMregNYByHjBdDzsm6H96YTMk8AfT9yE7RvfJLzigKKmFVsPdA4Jzj43_pHdblgPzIWxPptb7WMrA3tum3ySVG_nMi7gSaUaelPe1XlVrNo7UTkztI1XIfIa9g"
	differentProductCode = "rand"

	testNonce = "c21561da-087b-4efc-af6b-sdas232asd"
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

var errBoom = errors.New("boom")

type MockAWSMarketplace struct {
	MockRegisterUsage func(ctx context.Context, params *marketplacemetering.RegisterUsageInput, optFns ...func(*marketplacemetering.Options)) (*marketplacemetering.RegisterUsageOutput, error)
}

func (m *MockAWSMarketplace) RegisterUsage(ctx context.Context, params *marketplacemetering.RegisterUsageInput, optFns ...func(*marketplacemetering.Options)) (*marketplacemetering.RegisterUsageOutput, error) {
	return m.MockRegisterUsage(ctx, params, optFns...)
}

func TestAWSMarketplace_Register(t *testing.T) {
	type args struct {
		s    *corev1.Secret
		uid  string
		mcl  marketplaceClient
		kube client.Client
	}
	type want struct {
		token string
		err   error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"SignatureAlreadyExists": {
			reason: "we should not do any registration if we already have a signature",
			args: args{
				s: &corev1.Secret{
					Data: map[string][]byte{
						SecretKeyAWSMeteringSignature: []byte(signature),
					},
				},
			},
			want: want{
				token: signature,
			},
		},
		"RegisterFails": {
			reason: "We should return error if we cannot register usage",
			args: args{
				s: &corev1.Secret{},
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
				mcl: &MockAWSMarketplace{
					MockRegisterUsage: func(_ context.Context, _ *marketplacemetering.RegisterUsageInput, _ ...func(*marketplacemetering.Options)) (*marketplacemetering.RegisterUsageOutput, error) {
						return nil, errBoom
					},
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errRegisterUsage),
			},
		},
		"ApplyFails": {
			reason: "We should return error if we cannot apply the entitlement secret with signature",
			args: args{
				s: &corev1.Secret{},
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
				mcl: &MockAWSMarketplace{
					MockRegisterUsage: func(_ context.Context, _ *marketplacemetering.RegisterUsageInput, _ ...func(*marketplacemetering.Options)) (*marketplacemetering.RegisterUsageOutput, error) {
						return &marketplacemetering.RegisterUsageOutput{Signature: aws.String(signature)}, nil
					},
				},
			},
			want: want{
				token: signature,
				err:   errors.Wrap(errors.Wrap(errBoom, "cannot get object"), errApplySecret),
			},
		},
		"Success": {
			reason: "We should return the signature after a successful registration",
			args: args{
				s: &corev1.Secret{},
				kube: &test.MockClient{
					MockGet:   test.NewMockGetFn(nil),
					MockPatch: test.NewMockPatchFn(nil),
				},
				mcl: &MockAWSMarketplace{
					MockRegisterUsage: func(_ context.Context, _ *marketplacemetering.RegisterUsageInput, _ ...func(*marketplacemetering.Options)) (*marketplacemetering.RegisterUsageOutput, error) {
						return &marketplacemetering.RegisterUsageOutput{Signature: aws.String(signature)}, nil
					},
				},
			},
			want: want{
				token: signature,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewMarketplace(tc.args.kube, tc.args.mcl, testPublicKey)
			token, err := r.Register(context.Background(), tc.args.s, tc.args.uid)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\nReason: %s\nr.Register(...): -want token, +got token:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.token, token); diff != "" {
				t.Errorf("\nReason: %s\nr.Register(...): -want error, +got error:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestAWSMarketplace_Verify(t *testing.T) {
	type args struct {
		token string
		uid   string
	}
	type want struct {
		verified bool
		err      error
	}
	cases := map[string]struct {
		reason string
		args
		want
	}{
		"CorruptToken": {
			reason: "We should return error if the token is not valid JWT",
			args: args{
				token: "random",
			},
			want: want{
				err: errors.Wrap(errors.New("token contains an invalid number of segments"), errParseToken),
			},
		},
		"WrongProductCode": {
			reason: "We should return error if the product code does not match",
			args: args{
				token: differentToken,
			},
			want: want{
				verified: false,
				err:      errors.Errorf(errProductCodeMatchFmt, differentProductCode, MarketplaceProductCode),
			},
		},
		"Success": {
			reason: "We should return true if we are able to verify the token",
			args: args{
				token: correctToken,
				uid:   testNonce,
			},
			want: want{
				verified: true,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewMarketplace(nil, nil, testPublicKey)
			verified, err := r.Verify(tc.args.token, tc.args.uid)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\nReason: %s\nr.Verify(...): -want verified, +got verified:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.verified, verified); diff != "" {
				t.Errorf("\nReason: %s\nr.Verify(...): -want error, +got error:\n%s", tc.reason, diff)
			}
		})
	}
}
