package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

func Test_readPlatformIDFromToken(t *testing.T) {
	type args struct {
		t string
	}
	type want struct {
		id  string
		err error
	}
	cases := map[string]struct {
		args
		want
	}{
		"HappyPath": {
			args: args{
				// payload:
				// {
				//   "sub": "controlPlane|b0075060-a0d0-4948-80a3-ffdb0c28ef71"
				// }
				t: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJjb250cm9sUGxhbmV8YjAwNzUwNjAtYTBkMC00OTQ4LTgwYTMtZmZkYjBjMjhlZjcxIn0.gw4XC9O8XJbeoMUcw2tg4YR88tY6OkiTK0qQXvoT1OU",
			},
			want: want{
				id:  "b0075060-a0d0-4948-80a3-ffdb0c28ef71",
				err: nil,
			},
		},
		"MalformedToken": {
			args: args{
				t: "not-a-valid-jwt-token",
			},
			want: want{
				err: errors.WithStack(errors.New(fmt.Sprintf("%s: token contains an invalid number of segments", errMalformedCPToken))),
			},
		},
		"NoSubjectKey": {
			args: args{
				// payload:
				// {
				//   "name": "Test"
				// }
				t: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiVGVzdCJ9.E7_2g0YuTeNMwAS55izHnYVg_gmPkaQq6efZVR_lQDk",
			},
			want: want{
				err: errors.New(errCPTokenNoSubjectKey),
			},
		},
		"SubjectIsNotString": {
			args: args{
				// payload:
				// {
				//   "sub": 123
				// }
				t: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOjEyM30.cEfWW2xCUQoGJSGz5ORY6tqJQA1eI-HDMkcrfGQeLNI",
			},
			want: want{
				err: errors.New(errCPTokenSubjectIsNotString),
			},
		},
		"IdInTokenIsNotUUID": {
			args: args{
				// payload:
				// {
				//   "sub": "controlPlane|123"
				// }
				t: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJjb250cm9sUGxhbmV8MTIzIn0.jQf1BFdzAq7i0L5RiQhkhvwt8yZ-RVwftYJLdNbg4CY",
			},
			want: want{
				err: errors.Wrap(errors.New("invalid UUID length: 3"), fmt.Sprintf(errCPIDInTokenNotValidUUID, "123")),
				id:  "123",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, gotErr := readCPIDFromToken(tc.args.t)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("readCPIDFromToken(...): -want error, +got error: %s", diff)
			}
			if diff := cmp.Diff(tc.want.id, got); diff != "" {
				t.Errorf("readCPIDFromToken(...): -want result, +got result: %s", diff)
			}
		})
	}
}

func Test_readKubeClusterID(t *testing.T) {
	errBoom := errors.New("boom")
	uid := uuid.New().String()

	type args struct {
		kube client.Client
	}
	type want struct {
		id  string
		err error
	}
	cases := map[string]struct {
		args
		want
	}{
		"FailedToGet": {
			args{
				kube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						return errBoom
					},
				},
			},
			want{
				err: errors.Wrap(errBoom, errFailedToGetKubeSystemNS),
			},
		},
		"EmptyUID": {
			args{
				kube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						return nil
					},
				},
			},
			want{
				err: errors.New(errKubeSystemUIDEmpty),
			},
		},
		"HappyPath": {
			args{
				kube: &test.MockClient{
					MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
						if key.Name == "kube-system" {
							ns := obj.(*corev1.Namespace)
							ns.SetUID(types.UID(uid))
						}
						return nil
					},
				},
			},
			want{
				id:  uid,
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, gotErr := readKubeClusterID(tc.args.kube)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("readCPIDFromToken(...): -want error, +got error: %s", diff)
			}
			if diff := cmp.Diff(tc.want.id, got); diff != "" {
				t.Errorf("readCPIDFromToken(...): -want result, +got result: %s", diff)
			}
		})
	}
}
