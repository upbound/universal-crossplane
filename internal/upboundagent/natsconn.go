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
	"encoding/base64"
	"io/ioutil"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	natsjwt "github.com/nats-io/jwt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/pkg/errors"

	"github.com/upbound/universal-crossplane/internal/clients/upbound"
)

type natsConnManager struct {
	log       logging.Logger
	upClient  upbound.Client
	kp        nkeys.KeyPair
	pubKey    string
	clusterID string
	cpToken   string
	jwtToken  string
	caFile    string
}

func newNATSConnManager(log logging.Logger, upClient upbound.Client, cID, cpToken string, caBundle string) (*natsConnManager, error) {
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
		log:       log,
		upClient:  upClient,
		kp:        kp,
		pubKey:    pk,
		clusterID: cID,
		caFile:    caFile,
		cpToken:   cpToken,
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
		tk, err := n.upClient.FetchNewJWTToken(n.cpToken, n.clusterID, n.pubKey)
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
