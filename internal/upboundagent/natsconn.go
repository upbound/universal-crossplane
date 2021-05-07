package upboundagent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/go-resty/resty/v2"
	natsjwt "github.com/nats-io/jwt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/pkg/errors"
)

const (
	natsTokenPath = "/v1/nats/token"
	keyToken      = "token"
)

type natsConnManager struct {
	log                  logging.Logger
	restyClient          *resty.Client
	kp                   nkeys.KeyPair
	pubKey               string
	clusterID            string
	ubcNATSEndpointToken string
	jwtToken             string
	caFile               string
}

func newNATSConnManager(log logging.Logger, cID, ubcNATSEndpoint, ubcNATSEndpointToken string, caBundle string, debug bool) (*natsConnManager, error) {
	r := newRestyClient(ubcNATSEndpoint, debug)
	kp, err := nkeys.CreateUser()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create nats user")
	}
	pk, err := kp.PublicKey()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get public key for nats user nkey")
	}
	caFile, err := caBundleToFile(caBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get nats ca bundle file")
	}

	n := &natsConnManager{
		log:                  log,
		kp:                   kp,
		pubKey:               pk,
		restyClient:          r,
		clusterID:            cID,
		caFile:               caFile,
		ubcNATSEndpointToken: ubcNATSEndpointToken,
	}

	return n, nil
}
func (n *natsConnManager) setupAuthOption() nats.Option {
	return nats.UserJWT(n.userTokenRefresher, n.signatureHandler)
}

func (n *natsConnManager) setupTLSOption() nats.Option {
	return nats.RootCAs(n.caFile)
}

func (n *natsConnManager) userTokenRefresher() (string, error) {
	n.log.Debug("handling NATS user JWT")
	if !isJWTValid(n.jwtToken, n.log) {
		tk, err := fetchNewJWTToken(n.restyClient, n.log, n.ubcNATSEndpointToken, n.clusterID, n.pubKey)
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

func caBundleToFile(caBundle string) (string, error) {
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

func fetchNewJWTToken(client *resty.Client, log logging.Logger, ubcNATSEndpointToken, clusterID, publicKey string) (string, error) {
	log.Debug("fetching new NATS JWT")

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
	if respBody[keyToken] == "" {
		return "", errors.New("empty token received")
	}

	return respBody[keyToken], nil
}

func isJWTValid(token string, log logging.Logger) bool {
	if token == "" {
		return false
	}
	claims := &natsjwt.UserClaims{}
	err := natsjwt.Decode(token, claims)
	if err != nil {
		log.Info("failed to decode token", "error", err)
		return false
	}
	vr := &natsjwt.ValidationResults{}
	claims.Validate(vr)
	if len(vr.Issues) > 0 {
		log.Info("token is not valid with issues", "issues", vr.Issues)
		return false
	}
	log.Debug("existing NATS JWT is valid")
	return true
}
