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

package upbound

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
)

const (
	gwCertsPath   = "/v1/gw/certs"
	natsTokenPath = "/v1/nats/token"

	keyToken        = "token"
	keyJWTPublicKey = "jwt_public_key"
	keyNATSCA       = "nats_ca"
)

// PublicCerts keeps the public certificates/keys to interact with Upbound Cloud.
type PublicCerts struct {
	JWTPublicKey string
	NATSCA       string
}

// Client is the client for upbound api
//go:generate go run github.com/golang/mock/mockgen -copyright_file ../../../hack/boilerplate.txt -destination ./mocks/upbound.go -package mocks github.com/upbound/universal-crossplane/internal/clients/upbound Client
type Client interface {
	GetGatewayCerts(cpToken string) (PublicCerts, error)
	FetchNewJWTToken(cpToken, clusterID, publicKey string) (string, error)
}

type client struct {
	resty  *resty.Client
	logger logging.Logger
}

// NewClient returns a new Upbound client
func NewClient(host string, log logging.Logger, debug bool) Client {
	c := resty.New().
		SetHostURL(host).
		SetDebug(debug).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		// Could not use crossplane runtime logger here, since:
		// Cannot use 'log' (type Logger) as type Logger Type does not implement 'Logger' as some methods are missing:
		// Debugf(format string, v ...interface{})
		// Errorf(format string, v ...interface{})
		// Warnf(format string, v ...interface{})
		SetLogger(logrus.StandardLogger())

	c.SetTransport(&ochttp.Transport{})

	c.OnRequestLog(func(r *resty.RequestLog) error {
		// masking authorization header
		r.Header.Set("Authorization", "[REDACTED]")
		r.Body = "[REDACTED]"
		return nil
	})

	c.OnResponseLog(func(r *resty.ResponseLog) error {
		r.Body = "[REDACTED]"
		return nil
	})

	return &client{
		resty:  c,
		logger: log,
	}
}

// GetGatewayCerts function returns public certificates to interact with Upbound Cloud.
func (c *client) GetGatewayCerts(cpToken string) (PublicCerts, error) {
	req := c.resty.R()
	req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", cpToken))

	resp, err := req.Get(gwCertsPath)
	if err != nil {
		return PublicCerts{}, errors.Wrap(err, "failed to request gateway certs")
	}
	if resp.StatusCode() != http.StatusOK {
		return PublicCerts{}, errors.Errorf("gateway certs request failed with %s - %s", resp.Status(), string(resp.Body()))
	}
	respBody := map[string]string{}

	if err := json.Unmarshal(resp.Body(), &respBody); err != nil {
		return PublicCerts{}, errors.Wrap(err, "failed to unmarshall gw certs response")
	}
	j := respBody[keyJWTPublicKey]
	n := respBody[keyNATSCA]
	if j == "" {
		return PublicCerts{}, errors.New("empty jwt public key received")
	}
	if n == "" {
		return PublicCerts{}, errors.New("empty nats ca received")
	}

	return PublicCerts{
		JWTPublicKey: j,
		NATSCA:       n,
	}, nil
}

func (c *client) FetchNewJWTToken(cpToken, clusterID, publicKey string) (string, error) {
	req := c.resty.R()
	body := map[string]string{
		"clusterID":    clusterID,
		"clientPubKey": publicKey,
	}
	mBody, err := json.Marshal(body)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshall body to json")
	}

	req.SetBody(mBody)
	req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", cpToken))

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
