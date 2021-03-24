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

type Client interface {
	GetGatewayCerts(cpToken string) (string, string, error)
}

type client struct {
	resty *resty.Client
}

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

func (c *client) GetGatewayCerts(cpToken string) (string, string, error) {
	req := c.resty.R()
	req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", cpToken))

	resp, err := req.Get(gwCertsPath)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to request gateway certs")
	}
	if resp.StatusCode() != http.StatusOK {
		return "", "", errors.Errorf("gateway certs request failed with %s - %s", resp.Status(), string(resp.Body()))
	}
	respBody := map[string]string{}

	if err := json.Unmarshal(resp.Body(), &respBody); err != nil {
		return "", "", errors.Wrap(err, "failed to unmarshall gw certs response")
	}
	j := respBody[keyJWTPublicKey]
	n := respBody[keyNATSCA]
	if j == "" {
		return "", "", errors.New("empty jwt public key received")
	}
	if n == "" {
		return "", "", errors.New("empty nats ca received")
	}

	return j, n, nil
}
