package upbound

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	gwCertsPath = "/v1/gw/certs"

	keyJWTPublicKey = "jwt_public_key"
	keyNATSCA       = "nats_ca"
)

// PublicCerts keeps the public certificates/keys to interact with Upbound Cloud.
type PublicCerts struct {
	JWTPublicKey string
	NATSCA       string
}

// Client is the client for upbound api
//go:generate mockgen -destination ./mocks/upbound.go -package mocks github.com/upbound/universal-crossplane/internal/clients/upbound Client
type Client interface {
	GetGatewayCerts(cpToken string) (PublicCerts, error)
}

type client struct {
	resty *resty.Client
}

// NewClient returns a new Upbound client
func NewClient(host string, debug bool) Client {
	r := resty.New().
		SetHostURL(host).
		SetDebug(debug).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetLogger(logrus.StandardLogger())

	return &client{
		resty: r,
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
