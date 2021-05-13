// Copyright 2021 Upbound Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package upboundagent

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/upbound/nats-proxy/pkg/natsproxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/upbound/universal-crossplane/internal/clients/upbound"
	"github.com/upbound/universal-crossplane/internal/upboundagent/internal"
)

const (
	proxyPathArg = "*"

	readynessHandlerPath = "/readyz"
	livenessHandlerPath  = "/livez"
	k8sHandlerPath       = "/k8s/*"
	xgqlHandlerPath      = "/query"

	headerAuthorization      = "Authorization"
	groupSystemAuthenticated = "system:authenticated"

	impersonatorExtraKeyUpboundID = "upbound-id"
	impersonatorUserUpboundCloud  = "upbound-cloud-impersonator"

	serviceXgql = "xgql"

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	keepAliveInterval = 5 * time.Second
	drainTimeout      = 20 * time.Second
	shutdownTimeout   = 20 * time.Second
)

const (
	errUnableToValidateToken          = "unable to validate token"
	errUpboundIDMissing               = "upboundID is missing"
	errMissingAuthHeader              = "missing authorization header"
	errMissingBearer                  = "missing bearer token"
	errInvalidToken                   = "invalid token"
	errInvalidEnvID                   = "invalid environment id: %s, expecting: %s"
	errUnexpectedSigningMethod        = "unexpected signing method, expecting RS256 but found: %v"
	errFailedToGetImpersonationConfig = "failed to get impersonation config"
)

var (
	allowedHeaders = []string{
		"Content-Type",
		"X-Forwarded-For",
		"X-Forwarded-Host",
		"Accept-Encoding",
		"Accept",
		"User-Agent",
	}
)

// Proxy is an Upbound Agent Proxy
type Proxy struct {
	log           logging.Logger
	config        *Config
	kubeHost      *url.URL
	kubeTransport http.RoundTripper
	nc            *nats.Conn
	xgqlHost      *url.URL
	k8sBearer     string
	agent         *natsproxy.Agent
	server        *http.Server
	isReady       *atomic.Value
}

// NewProxy returns a new Proxy
func NewProxy(config *Config, restConfig *rest.Config, upClient upbound.Client, log logging.Logger, clusterID string) (*Proxy, error) {
	krt, err := roundTripperForRestConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build round tripper for rest config")
	}

	// get k8s API server url
	kubeHost, err := url.Parse(restConfig.Host)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse kube url")
	}

	xgqlHost, err := url.Parse(fmt.Sprintf("https://%s", serviceXgql))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse xgql url")
	}

	// TODO(turkenh): remove once nats-proxy starts using logging interface: https://github.com/upbound/nats-proxy/issues/3
	if config.DebugMode {
		// set log level for nats-proxy
		logrus.SetLevel(logrus.DebugLevel)
	}
	var nc *nats.Conn
	natsConn, err := newNATSConnManager(log, upClient, clusterID, config.NATS.ControlPlaneToken, config.NATS.CABundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new nats connection manager")
	}
	nopts := []nats.Option{nats.Name(fmt.Sprintf("%s-%s", config.ControlPlaneID, config.NATS.Name))}
	nopts = natsproxy.SetupConnOptions(nopts)
	nopts = append(nopts, natsConn.setupAuthOption(), natsConn.setupTLSOption())
	// Connect to NATS
	nc, err = nats.Connect(config.NATS.Endpoint, nopts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect NATS")
	}

	pxy := &Proxy{
		log:           log,
		nc:            nc,
		kubeHost:      kubeHost,
		kubeTransport: krt,
		config:        config,
		xgqlHost:      xgqlHost,
		k8sBearer:     restConfig.BearerToken,
		isReady:       &atomic.Value{},
	}

	return pxy, nil
}

// Run runs Upbound Agent Proxy.
func (p *Proxy) Run(addr, certFile, keyFile string) error {
	p.isReady.Store(true)

	e, err := p.setupRouter()
	if err != nil {
		return errors.Wrap(err, "failed to setup router")
	}

	s := &http.Server{
		Handler:           e,
		Addr:              addr,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		// Note(turkenh): WriteTimeout intentionally left as "0" since setting a write timeout breaks k8s watch requests.
		WriteTimeout: 0,
	}
	p.server = s
	go func() {
		if err := s.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			err = errors.Wrap(err, "service stopped unexpectedly")
			p.log.Info(err.Error())
			os.Exit(-1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	// interrupt signal sent from terminal
	signal.Notify(quit, syscall.SIGINT)
	// sigterm signal sent from kubernetes
	signal.Notify(quit, syscall.SIGTERM)
	<-quit

	return p.shutdown()
}

func (p *Proxy) shutdown() error {
	p.isReady.Store(false)

	p.log.Debug("proxy shutdown: draining nats agent")
	dtc, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	err := p.agent.Drain()
	if err != nil {
		return errors.Wrap(err, "error on drain")
	}

draining:
	for p.agent.IsDraining() {
		select {
		case <-dtc.Done():
			p.log.Info("error: proxy shutdown, drain timed out")
			break draining
		default:
			p.log.Debug("proxy shutdown: still draining")
			time.Sleep(100 * time.Millisecond)
		}
	}

	p.log.Info("proxy shutdown: shutting down server")
	stc, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return p.server.Shutdown(stc)
}

// setupRouter setup an echo instance as a router.
func (p *Proxy) setupRouter() (*echo.Echo, error) {
	e := echo.New()

	e.Logger.SetLevel(log.INFO)
	if p.config.DebugMode {
		e.Use(middleware.Logger())
		e.Logger.SetLevel(log.DEBUG)
	}

	e.Use(middleware.Recover())

	prm := prometheus.NewPrometheus("upbound_agent", nil)
	prm.Use(e)

	jt := jaegertracing.New(e, nil)
	defer jt.Close() // nolint:errcheck

	// TODO(turkenh): use different routers for nats agent and http server once graphql removed, which will let us
	// remove k8s from http server
	e.Any(k8sHandlerPath, p.k8s())
	e.Any(xgqlHandlerPath, p.xgql())
	e.Any(readynessHandlerPath, p.readyz())
	e.Any(livenessHandlerPath, p.livez())

	agentID, err := uuid.Parse(p.config.ControlPlaneID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse control plane id as uid")
	}
	agent := natsproxy.NewAgent(p.nc, agentID, e, getSubjectForAgent(agentID), keepAliveInterval)
	err = agent.Listen()

	if err != nil {
		return nil, errors.Wrap(err, "failed to listen to nats")
	}
	p.agent = agent
	return e, nil
}

func (p *Proxy) livez() echo.HandlerFunc {
	return func(c echo.Context) error {
		if p.nc.Status() == nats.CONNECTED {
			return c.JSON(http.StatusOK, echo.Map{"status": http.StatusOK, "nats-status": p.nc.Status()})
		}
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"status": http.StatusServiceUnavailable, "nats-status": p.nc.Status()})
	}
}

func (p *Proxy) readyz() echo.HandlerFunc {
	return func(c echo.Context) error {
		if p.isReady.Load().(bool) {
			return c.JSON(http.StatusOK, echo.Map{"status": http.StatusOK})
		}
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"status": http.StatusServiceUnavailable})
	}
}

func (p *Proxy) xgql() echo.HandlerFunc {
	return func(c echo.Context) error {
		p.log.Debug("incoming xgql request", "url", c.Request().URL.String())

		ic, err := p.getImpersonationConfig(c.Request().Header)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, echo.Map{"message": err.Error()})
		}

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
				RootCAs:            p.config.XGQLCACertPool,
				MinVersion:         tls.VersionTLS12,
			},
		}
		btr := transport.NewBearerAuthRoundTripper(p.k8sBearer, tr)
		itr := transport.NewImpersonatingRoundTripper(ic, btr)

		rp := httputil.NewSingleHostReverseProxy(p.xgqlHost)
		rp.Transport = itr
		rp.ErrorHandler = p.error

		reqCopy := sanitizeRequest(c.Request())
		reqCopy.URL.Host = p.xgqlHost.Host

		rp.ServeHTTP(c.Response(), reqCopy)
		p.log.Debug("response from xgql", "status", c.Response().Status)
		return nil
	}
}

func (p *Proxy) k8s() echo.HandlerFunc {
	return func(c echo.Context) error {
		p.log.Debug("incoming k8s request", "url", c.Request().URL.String())

		ic, err := p.getImpersonationConfig(c.Request().Header)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, echo.Map{"message": err.Error()})
		}

		irt := transport.NewImpersonatingRoundTripper(ic, p.kubeTransport)

		rp := httputil.NewSingleHostReverseProxy(p.kubeHost)
		rp.Transport = irt
		rp.ErrorHandler = p.error

		reqCopy := sanitizeRequest(c.Request())
		reqCopy.URL.Path = parseDestinationPath(c) // k8s/path -> path

		rp.ServeHTTP(c.Response(), reqCopy)
		p.log.Debug("response from k8s", "status", c.Response().Status)
		return nil
	}
}

func (p *Proxy) getImpersonationConfig(requestHeader http.Header) (transport.ImpersonationConfig, error) {
	var cfg transport.ImpersonationConfig

	tc, err := p.reviewToken(requestHeader)
	if err != nil {
		err = errors.Wrap(err, errUnableToValidateToken)
		p.log.Info(err.Error())
		return cfg, err
	}

	cid := tc.Audience
	if cid != p.config.ControlPlaneID {
		err = errors.Errorf(errInvalidEnvID, cid, p.config.ControlPlaneID)
		p.log.Info(err.Error())
		return cfg, err
	}

	p.log.Debug("token is valid")

	cfg, err = impersonationConfigForUser(tc.Payload, p.log)
	if err != nil {
		err = errors.Wrap(err, errFailedToGetImpersonationConfig)
		p.log.Info(err.Error())
		return cfg, err
	}
	return cfg, nil
}

func parseDestinationPath(c echo.Context) string {
	// We swallow the error because "/" with no additional info is valid.
	return c.Param(proxyPathArg)
}

// sanitizeRequest creates a shallow copy of the request along with a deep copy of the allowed Headers.
func sanitizeRequest(req *http.Request) *http.Request {
	r := new(http.Request)

	// shallow clone
	*r = *req

	// deep copy headers
	r.Header = cloneAllowedHeaders(req.Header)

	return r
}

// cloneAllowedHeaders deep copies allowed headers
func cloneAllowedHeaders(in http.Header) http.Header {
	out := http.Header{}
	var val string
	for i := 0; i < len(allowedHeaders); i++ {
		val = in.Get(allowedHeaders[i])
		if val != "" {
			out.Add(allowedHeaders[i], val)
		}
	}
	return out
}

func (p *Proxy) error(rw http.ResponseWriter, r *http.Request, err error) {
	p.log.Info("unknown error", "err", err, "remote-addr", r.RemoteAddr)
	http.Error(rw, "", http.StatusInternalServerError)
}

func (p *Proxy) reviewToken(requestHeaders http.Header) (*internal.TokenClaims, error) {
	auth := strings.TrimSpace(requestHeaders.Get(headerAuthorization))
	if auth == "" {
		return nil, errors.New(errMissingAuthHeader)
	}
	parts := strings.Split(auth, " ")
	if len(parts) < 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, errors.New(errMissingBearer)
	}

	tokenStr := parts[1]
	tcs := &internal.TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, tcs, func(token *jwt.Token) (interface{}, error) {
		if sm, ok := token.Method.(*jwt.SigningMethodRSA); !ok || sm.Name != "RS256" {
			return nil, errors.Errorf(errUnexpectedSigningMethod, token.Header["alg"])
		}
		return p.config.TokenRSAPublicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New(errInvalidToken)
	}
	return tcs, err
}

func roundTripperForRestConfig(config *rest.Config) (http.RoundTripper, error) {
	tlsConf, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, err
	}

	tlsTransport := &http.Transport{
		TLSClientConfig: tlsConf,
	}

	restTransportConfig, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}

	kubeRT, err := transport.HTTPWrappersForConfig(restTransportConfig, tlsTransport)
	if err != nil {
		return nil, err
	}

	return kubeRT, nil
}

func impersonationConfigForUser(ca internal.CrossplaneAccessor, log logging.Logger) (transport.ImpersonationConfig, error) {
	log.Debug("Impersonating user info", "upboundID", ca.UpboundID, "groups", ca.Groups)

	if ca.UpboundID == "" {
		return transport.ImpersonationConfig{}, errors.New(errUpboundIDMissing)
	}

	return transport.ImpersonationConfig{
		UserName: impersonatorUserUpboundCloud,
		Groups:   append(ca.Groups, groupSystemAuthenticated),
		Extra: map[string][]string{
			impersonatorExtraKeyUpboundID: {ca.UpboundID},
		},
	}, nil
}

// getSubjectForAgent returns the NATS subject for agent for a given control plane
func getSubjectForAgent(agentID uuid.UUID) string {
	return fmt.Sprintf("platforms.%s.gateway", agentID.String())
}
