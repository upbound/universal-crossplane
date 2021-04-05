package billing

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

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
const (
	AWSProductCode      = "todo"
	AWSPublicKey        = "BEGIN RSA public key"
	AWSPublicKeyVersion = 1
)

// Constants used for in-cluster operations.
const (
	SecretNameAWSMarketplace = "upbound-aws-marketplace"
	SecretKeyAWSUsageToken   = "token"
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
func (am *AWSMarketplace) Register(ctx context.Context, namespace, uid string) (string, error) {
	s := &v1.Secret{}
	nn := types.NamespacedName{Name: SecretNameAWSMarketplace, Namespace: namespace}
	if err := am.client.Get(ctx, nn, s); err != nil {
		return "", errors.Wrap(err, "cannot get aws marketplace secret")
	}
	if len(s.Data[SecretKeyAWSUsageToken]) > 0 {
		return string(s.Data[SecretKeyAWSUsageToken]), nil
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
		if err := am.client.Get(ctx, nn, s); err != nil {
			return err
		}
		s.Data[SecretKeyAWSUsageToken] = []byte(aws.ToString(resp.Signature))
		return am.client.Update(ctx, s)
	})
	return aws.ToString(resp.Signature), errors.Wrap(err, "cannot update aws marketplace secret")
}

// Verify makes sure the signature is signed by AWS Marketplace.
func (am *AWSMarketplace) Verify(token, uid string) (bool, error) {
	l := strings.Split(token, ".")
	if len(l) != 3 {
		return false, errors.New("jwt token has to be made up of 3 parts separated by periods")
	}
	t, err := jwt.Parse(token, func(_ *jwt.Token) (interface{}, error) {
		return AWSPublicKey, nil
	})
	if err != nil {
		return false, errors.Wrap(err, "cannot parse token")
	}
	if !t.Valid {
		return false, nil
	}
	p, err := base64.URLEncoding.DecodeString(l[1])
	if err != nil {
		return false, errors.Wrap(err, "cannot decode jwt payload")
	}
	payload := map[string]string{}
	if err := json.Unmarshal(p, &payload); err != nil {
		return false, errors.Wrap(err, "cannot unmarshal jwt payload into string map")
	}
	switch {
	case payload["productCode"] != AWSProductCode:
		return false, nil
	case payload["nonce"] != uid:
		return false, nil
	case payload["publicKeyVersion"] != strconv.Itoa(AWSPublicKeyVersion):
		return false, nil
	}
	return true, nil
}
