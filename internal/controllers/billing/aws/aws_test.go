package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	signature = "some sig"

	// You can use https://jwt.io for generating these tokens. The private key
	// used can be found in internal/upboundagent/proxy_test.go as comment.
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

var errBoom = errors.New("boom!")

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
			r := NewAWSMarketplace(tc.args.kube, tc.args.mcl, testPublicKey)
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
				err:      errors.Errorf(errProductCodeMatchFmt, differentProductCode, AWSProductCode),
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
			r := NewAWSMarketplace(nil, nil, testPublicKey)
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
