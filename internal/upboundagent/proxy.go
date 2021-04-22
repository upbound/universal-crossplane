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

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/upbound/nats-proxy/pkg/natsproxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

	"github.com/upbound/universal-crossplane/internal/upboundagent/internal"
)

const (
	proxyPathArg = "*"

	readynessHandlerPath = "/readyz"
	livenessHandlerPath  = "/livez"
	k8sHandlerPath       = "/k8s/*"
	graphqlHandlerPath   = "/graphql"

	headerAuthorization      = "Authorization"
	groupCrossplaneOwner     = "crossplane:masters"
	groupSystemAuthenticated = "system:authenticated"

	keyUpboundUser   = "user-on-upbound-cloud"
	userUpboundCloud = "upbound-cloud-user"

	serviceGraphQL = "crossplane-graphql"

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	keepAliveInterval = 5 * time.Second
	drainTimeout      = 20 * time.Second
	shutdownTimeout   = 20 * time.Second
)

const (
	errUnableToValidateToken          = "unable to validate token"
	errUsernameMissing                = "username is missing"
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
	log      logging.Logger
	config   *Config
	kubeHost *url.URL
	http.Header
	kubeTransport http.RoundTripper
	nc            *nats.Conn
	graphQLHost   *url.URL
	agent         *natsproxy.Agent
	server        *http.Server
	isReady       *atomic.Value
}

// NewProxy returns a new Proxy
func NewProxy(config *Config, restConfig *rest.Config, log logging.Logger, clusterID string) (*Proxy, error) {
	krt, err := roundTripperForRestConfig(restConfig)
	if err != nil {
		panic(err)
	}

	// get k8s API server url
	kubeHost, err := url.Parse(restConfig.Host)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse kube url")
	}

	graphQLHost, err := url.Parse(fmt.Sprintf("https://%s", serviceGraphQL))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse graphql url")
	}

	var nc *nats.Conn
	natsConn, err := newNATSConnManager(log, clusterID, config.NATS.JWTEndpoint, config.NATS.ControlPlaneToken, true)
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
		graphQLHost:   graphQLHost,
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
			p.log.Info("Error: proxy shutdown, drain timed out")
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

	e.Any(k8sHandlerPath, p.k8s())
	e.Any(graphqlHandlerPath, p.graphql())
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

func (p *Proxy) graphql() echo.HandlerFunc {
	return func(c echo.Context) error {
		gqlProxy := httputil.NewSingleHostReverseProxy(p.graphQLHost)

		gqlProxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
				RootCAs:            p.config.GraphQLCACertPool,
				MinVersion:         tls.VersionTLS12,
			},
		}

		p.log.Debug(fmt.Sprintf("graphql.proxy: %s", c.Request().URL.String()))
		c.Request().URL.Host = p.graphQLHost.Host

		gqlProxy.ServeHTTP(c.Response(), c.Request())
		p.log.Debug(fmt.Sprintf("graphql.proxy: %d", c.Response().Status))
		return nil
	}
}

func (p *Proxy) k8s() echo.HandlerFunc {
	return func(c echo.Context) error {
		p.log.Debug(fmt.Sprintf("incoming request url: %s", c.Request().URL.String()))

		tc, err := p.reviewToken(c.Request().Header)
		if err != nil {
			err = errors.Wrap(err, errUnableToValidateToken)
			p.log.Info(err.Error())
			return echo.NewHTTPError(http.StatusBadRequest, echo.Map{"message": err.Error()})
		}

		// Work on a copy of the request, RoundTrip should never modify the request:
		// https://golang.org/src/net/http/client.go#L103
		reqCopy := sanitizeRequest(c.Request())
		reqCopy.URL.Path = parseDestinationPath(c) // k8s/path -> path

		cid := tc.Audience
		if cid != p.config.ControlPlaneID {
			err = errors.New(fmt.Sprintf(errInvalidEnvID, cid, p.config.ControlPlaneID))
			p.log.Info(err.Error())
			return echo.NewHTTPError(http.StatusBadRequest, echo.Map{"message": err.Error()})
		}

		p.log.Debug("Token is valid")

		iCfg, err := impersonationConfigForUser(tc.User, p.log)
		if err != nil {
			err = errors.Wrap(err, errFailedToGetImpersonationConfig)
			p.log.Info(err.Error())
			return echo.NewHTTPError(http.StatusBadRequest, echo.Map{"message": err.Error()})
		}

		irt := transport.NewImpersonatingRoundTripper(iCfg, p.kubeTransport)
		k8sProxy := httputil.NewSingleHostReverseProxy(p.kubeHost)
		k8sProxy.Transport = irt
		k8sProxy.ErrorHandler = p.error

		k8sProxy.ServeHTTP(c.Response(), reqCopy)
		return nil
	}
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
			return nil, errors.New(fmt.Sprintf(errUnexpectedSigningMethod, token.Header["alg"]))
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

func impersonationConfigForUser(u internal.CrossplaneAccessor, log logging.Logger) (transport.ImpersonationConfig, error) {
	user := u.Identifier
	groups := u.TeamIDs
	isOwner := u.IsOwner

	log.Debug(fmt.Sprintf("User info: isowner %v user %s groups %v", isOwner, user, groups))

	if user == "" {
		return transport.ImpersonationConfig{}, errors.New(errUsernameMissing)
	}

	groups = append(groups, groupSystemAuthenticated)
	if isOwner {
		groups = append(groups, groupCrossplaneOwner)
	}

	return transport.ImpersonationConfig{
		UserName: userUpboundCloud,
		Groups:   groups,
		Extra: map[string][]string{
			keyUpboundUser: {user},
		},
	}, nil
}

// getSubjectForAgent returns the NATS subject for agent for a given control plane
func getSubjectForAgent(agentID uuid.UUID) string {
	return fmt.Sprintf("platforms.%s.gateway", agentID.String())
}
