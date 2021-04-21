package billing

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/marketplacemetering"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// These constants are given by AWS Marketplace.
// TODO(muvaf): Consider fetching them from an Upbound API but keep the latest
// ones hard-coded as fallback for air-gapped environments.
const (
	AWSProductCode      = "1fszvu527waovqeuhpkyx2b5d"
	AWSPublicKey        = "-----BEGIN PUBLIC KEY-----\nMIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAyu7Xq7XTBRgFWCL+DXj8\nXyc/fPLWNQ1adPDf8zqkJ1H1JCTg6fUo7HUvNu0BAbPwIME4aDEzteJkhPq9IzS8\nHlrZT/7DqSPV9bXnR9OkqugfbFPyHGyd9afHyfDJfGwfqBP5r8oBuGwmCw5Ia088\nAcePfkVEisAo+8KiBAE16bqvDw0v5YzDrDVpHH9YdK1q9eG5WRTt0h7lYFj8dydr\nh+OyONGyWTkAWbs3JpsQLZgRdU6Klj5aZzO6FeUc2kOz2Hs+QvKgbNSpgV0000KK\n2on4L1+WJau7sj8EFquFdk2C0MhucIy6ceWXGB3YAOb8c0H9FT0eSY5rtX154otW\njmV9vMLLX1gajtQD0iOLBLRQ3WliP7fGc6o3StjMrbKh+ErXGVzzJnjK2eQhgkg/\n/DgcKjUptZ21gdbqbQBGwvfitBEJX7VCwF4VMhFM8JQiAxCVBZ7kkY5ZlGjvN2gO\nAMFKarvAWRwrZisxKWe+RFBU1EI5WS75X7owU/IehIabAgMBAAE=\n-----END PUBLIC KEY-----\n"
	AWSPublicKeyVersion = 1
)

// SecretKeyAWSMeteringSignature is the key whose value contains JWT signature returned
// from AWS Metering Service.
const (
	SecretKeyAWSMeteringSignature = "awsMeteringSignature"
)

type marketplaceClient interface {
	RegisterUsage(ctx context.Context, params *marketplacemetering.RegisterUsageInput, optFns ...func(*marketplacemetering.Options)) (*marketplacemetering.RegisterUsageOutput, error)
}

// NewAWSMarketplace returns a new AWSMarketplace object that can register usage.
func NewAWSMarketplace(cl client.Client, mcl marketplaceClient) *AWSMarketplace {
	return &AWSMarketplace{
		client:   cl,
		metering: mcl,
	}
}

// AWSMarketplace implements Registerer for AWS Marketplace API.
type AWSMarketplace struct {
	client   client.Client
	metering marketplaceClient
}

// Register makes sure user is entitled for this usage in an idempotent way.
func (am *AWSMarketplace) Register(ctx context.Context, s *v1.Secret, uid string) (string, error) {
	if len(s.Data[SecretKeyAWSMeteringSignature]) > 0 {
		return string(s.Data[SecretKeyAWSMeteringSignature]), nil
	}
	u := &marketplacemetering.RegisterUsageInput{
		ProductCode:      aws.String(AWSProductCode),
		PublicKeyVersion: aws.Int32(AWSPublicKeyVersion),
		Nonce:            aws.String(uid),
	}
	resp, err := am.metering.RegisterUsage(ctx, u)
	if err != nil {
		return "", errors.Wrap(err, "cannot register usage")
	}
	err = retry.OnError(retry.DefaultRetry, resource.IsAPIError, func() error {
		nn := types.NamespacedName{Name: s.Name, Namespace: s.Namespace}
		if err := am.client.Get(ctx, nn, s); err != nil {
			return err
		}
		if s.Data == nil {
			s.Data = map[string][]byte{}
		}
		s.Data[SecretKeyAWSMeteringSignature] = []byte(aws.ToString(resp.Signature))
		return am.client.Update(ctx, s)
	})
	return aws.ToString(resp.Signature), errors.Wrapf(err, "cannot update entitlement secret %s/%s", s.Namespace, s.Name)
}

// Verify makes sure the signature is signed by AWS Marketplace.
func (am *AWSMarketplace) Verify(token, uid string) (bool, error) {
	t, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(_ *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM([]byte(AWSPublicKey))
	})
	if err != nil {
		return false, errors.Wrap(err, "cannot parse token")
	}
	if !t.Valid {
		return false, errors.New("token is invalid")
	}
	claims := t.Claims.(jwt.MapClaims)
	switch {
	case claims["productCode"] != AWSProductCode:
		return false, errors.Errorf("productCode %s does not match expected %s", claims["productCode"], AWSProductCode)
	case claims["nonce"] != uid:
		return false, errors.Errorf("nonce %s does not match expected %s", claims["nonce"], uid)
	case claims["publicKeyVersion"] != float64(AWSPublicKeyVersion):
		return false, errors.Errorf("publicKeyVersion %s does not match expected %f", claims["publicKeyVersion"], float64(AWSPublicKeyVersion))
	}
	return true, nil
}
