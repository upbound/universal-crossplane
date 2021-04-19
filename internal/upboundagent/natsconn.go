package upboundagent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-resty/resty/v2"
	natsjwt "github.com/nats-io/jwt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	uhttp "github.com/upbound/upbound-runtime/pkg/http"
)

const (
	natsTokenPath = "/v1/nats/token"
	natsCAPath    = "/v1/gw/certs"
)

type natsConnManager struct {
	restyClient          *resty.Client
	kp                   nkeys.KeyPair
	pubKey               string
	clusterID            string
	ubcNATSEndpointToken string
	jwtToken             string
	caFile               string
}

func newNATSConnManager(cID, ubcNATSEndpoint, ubcNATSEndpointToken string, debug bool) (*natsConnManager, error) {
	r := uhttp.NewRestyClient(ubcNATSEndpoint, false, debug)
	kp, err := nkeys.CreateUser()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create nats user")
	}
	pk, err := kp.PublicKey()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get public key for nats user nkey")
	}

	caFile, err := getCABundleFile(r, ubcNATSEndpointToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get nats ca bundle file")
	}

	return &natsConnManager{
		kp:                   kp,
		pubKey:               pk,
		restyClient:          r,
		clusterID:            cID,
		ubcNATSEndpointToken: ubcNATSEndpointToken,
		caFile:               caFile,
	}, nil
}
func (n *natsConnManager) setupAuthOption() nats.Option {
	return nats.UserJWT(n.userTokenRefresher, n.signatureHandler)
}

func (n *natsConnManager) setupTLSOption() nats.Option {
	return nats.RootCAs(n.caFile)
}

func (n *natsConnManager) userTokenRefresher() (string, error) {
	logrus.Debug("handling NATS user JWT")
	if !isJWTValid(n.jwtToken) {
		tk, err := fetchNewJWTToken(n.restyClient, n.ubcNATSEndpointToken, n.clusterID, n.pubKey)
		if err != nil {
			return "", err
		}
		n.jwtToken = tk
	}
	return n.jwtToken, nil
}

func (n *natsConnManager) signatureHandler(nonce []byte) ([]byte, error) {
	return n.kp.Sign(nonce)
}

func getCABundleFile(client *resty.Client, ubcNATSEndpointToken string) (string, error) {
	caBundle, err := fetchCABundle(client, ubcNATSEndpointToken)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch ca bundle")
	}
	b, err := base64.StdEncoding.DecodeString(caBundle)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode ca bundle")
	}

	caFile, err := ioutil.TempFile("", "nats-ca")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file for nats ca")
	}
	if _, err = caFile.Write(b); err != nil {
		return "", errors.Wrap(err, "failed to write nats ca to file")
	}
	return caFile.Name(), nil
}

func fetchCABundle(client *resty.Client, ubcNATSEndpointToken string) (string, error) {
	logrus.Debug("fetching NATS CA bundle")

	req := client.R()
	req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", ubcNATSEndpointToken))

	resp, err := req.Get(natsCAPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to request ca bundle")
	}
	if resp.StatusCode() != http.StatusOK {
		return "", errors.Errorf("ca bundle request failed with %s - %s", resp.Status(), string(resp.Body()))
	}
	respBody := map[string]string{}

	if err := json.Unmarshal(resp.Body(), &respBody); err != nil {
		return "", errors.Wrap(err, "failed to unmarshall nats ca bundle response")
	}
	if respBody["nats_ca"] == "" {
		return "", errors.New("empty nats ca bundle received")
	}

	return respBody["nats_ca"], nil
}

func fetchNewJWTToken(client *resty.Client, ubcNATSEndpointToken, clusterID, publicKey string) (string, error) {
	logrus.Debug("fetching new NATS JWT")

	body := map[string]string{
		"clusterID":    clusterID,
		"clientPubKey": publicKey,
	}
	mBody, err := json.Marshal(body)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshall body to json")
	}

	req := client.R()
	req.SetBody(mBody)
	req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", ubcNATSEndpointToken))

	resp, err := req.Post(natsTokenPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to request new token")
	}

	if resp.StatusCode() != http.StatusOK {
		return "", errors.Errorf("new token request failed with %s - %s", resp.Status(), string(resp.Body()))
	}

	respBody := map[string]string{}

	if err := json.Unmarshal(resp.Body(), &respBody); err != nil {
		return "", errors.Wrap(err, "failed to unmarshall nats token response")
	}
	if respBody["token"] == "" {
		return "", errors.New("empty token received")
	}

	return respBody["token"], nil
}

func isJWTValid(token string) bool {
	if token == "" {
		return false
	}
	claims := &natsjwt.UserClaims{}
	err := natsjwt.Decode(token, claims)
	if err != nil {
		logrus.Debugf("failed to decode token: %v", err)
		return false
	}
	vr := &natsjwt.ValidationResults{}
	claims.Validate(vr)
	if len(vr.Issues) > 0 {
		logrus.Debugf("token is not valid with issues: %v", vr.Issues)
		return false
	}
	logrus.Debug("existing NATS JWT is valid")
	return true
}
