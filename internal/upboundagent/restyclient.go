package upboundagent

import (
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
)

// newRestyClient creates new resty client configured for logging and tracing
func newRestyClient(host string, debug bool) *resty.Client {
	c := resty.New().
		SetHostURL(host).
		SetDebug(debug).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
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

	return c
}
