package upboundagent

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/upbound/nats-proxy/pkg/natsproxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

	"github.com/upbound/universal-crossplane/internal/upboundagent/internal"
)

const (
	proxyPathArg             = "*"
	k8sHandlerPath           = "/k8s/*"
	graphqlHandlerPath       = "/graphql"
	natsLivenessHandlerPath  = "/natz"
	headerAuthorization      = "Authorization"
	groupCrossplaneOwner     = "crossplane:masters"
	groupSystemAuthenticated = "system:authenticated"

	keyUpboundUser   = "user-on-upbound-cloud"
	userUpboundCloud = "upbound-cloud-user"

	errUnableToValidateToken          = "unable to validate token"
	errUsernameMissing                = "username is missing"
	errMissingAuthHeader              = "missing authorization header"
	errMissingBearer                  = "missing bearer token"
	errInvalidToken                   = "invalid token"
	errInvalidEnvID                   = "invalid environment id: %s, expecting: %s"
	errUnexpectedSigningMethod        = "unexpected signing method, expecting RS256 but found: %v"
	errFailedToGetImpersonationConfig = "failed to get impersonation config"

	// serviceGraphQL is the name of the graphql kubernetes service
	serviceGraphQL = "crossplane-graphql"

	// watchFlushInterval the time for the ReverseProxy to flush it's data to the response, -1 = immediate.
	watchFlushInterval = -1
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

// Proxy is a Kubernetes Apiserver proxy
type Proxy struct {
	config   *Config
	kubeHost *url.URL
	http.Header
	kubeTransport http.RoundTripper
	nc            *nats.Conn
	graphQLHost   *url.URL
	agent         *natsproxy.Agent
}

// TokenClaims is the struct for custom claims of JWT token
type TokenClaims struct {
	User internal.CrossplaneAccessor `json:"payload"`
	jwt.StandardClaims
}

// NewProxy returns a new Proxy
func NewProxy(config *Config, restConfig *rest.Config, clusterID string) *Proxy { // nolint:gocyclo
	logrus.Info("starting upbound agent proxy server...")

	if config.DebugMode {
		logrus.SetLevel(logrus.DebugLevel)
	}

	krt, err := roundTripperForRestConfig(restConfig)
	if err != nil {
		panic(err)
	}

	// get API server url
	kubeHost, err := url.Parse(restConfig.Host)
	if err != nil {
		logrus.WithError(err).Panic("failed to parse kube url")
	}

	graphQLHost, err := url.Parse(fmt.Sprintf("https://%s", serviceGraphQL))
	if err != nil {
		logrus.WithError(err).Panic("failed to parse graphql url")
	}

	var nc *nats.Conn

	natsConn, err := newNATSConnManager(clusterID, config.NATS.JWTEndpoint, config.NATS.ControlPlaneToken, true)
	if err != nil {
		logrus.Fatal(err)
	}

	opts := []nats.Option{nats.Name(fmt.Sprintf("%s-%s", config.EnvID, config.NATS.Name))}
	opts = natsproxy.SetupConnOptions(opts)
	opts = append(opts, natsConn.setupAuthOption(), natsConn.setupTLSOption())

	// Connect to NATS
	nc, err = nats.Connect(config.NATS.Endpoint, opts...)
	if err != nil {
		// pod restarts, will retry with exponential backoff
		logrus.Fatal(err)
	}

	pxy := &Proxy{
		nc:            nc,
		kubeHost:      kubeHost,
		kubeTransport: krt,
		config:        config,
		graphQLHost:   graphQLHost,
	}

	return pxy
}

// getSubjectForAgent returns the NATS subject for agent for a given control plane
func getSubjectForAgent(agentID uuid.UUID) string {
	return fmt.Sprintf("platforms.%s.gateway", agentID.String())
}

// SetupRoutes satisfies RouteInitializer Interface from Upbound Runtime to setup a service.
func (p *Proxy) SetupRoutes(engine *echo.Echo) {
	// TODO: plumb this in a more intuitive way. In routeNatsRequest we use this reference to ServeHTTP response.
	// This is here because UpboundAgent serves incoming nats requests via echo.ServeHTTP
	// We need to do cleanup on subscription returned from us.Listen and this seems like a bad place.

	agentID, err := uuid.Parse(p.config.EnvID)
	if err != nil {
		logrus.Fatal(err)
	}
	keepAliveInterval := time.Second * 5

	agent := natsproxy.NewAgent(p.nc, agentID, engine, getSubjectForAgent(agentID), keepAliveInterval)
	err = agent.Listen()

	if err != nil {
		logrus.WithError(err).Panic("failed to listen to nats")
	}

	p.agent = agent

	engine.Any(k8sHandlerPath, p.k8s())
	engine.Any(graphqlHandlerPath, p.graphql())
	engine.Any(natsLivenessHandlerPath, p.natz())
}

// Drain the agent
func (p *Proxy) Drain() error {
	logrus.Debug("Proxy.Drain() will drain agent")
	return p.agent.Drain()
}

// IsDraining checks if agent is still draining
func (p *Proxy) IsDraining() bool {
	logrus.Debug("Proxy.IsDraining()")
	return p.agent.IsDraining()
}

// TODO(hasan): needs testing as a liveness/readyness check
func (p *Proxy) natz() echo.HandlerFunc {
	logrus.Debug("Proxy.nc.Status()")

	return func(c echo.Context) error {
		if p.nc.Status() == nats.CLOSED || p.nc.Status() == nats.DISCONNECTED {
			return c.JSON(http.StatusServiceUnavailable, echo.Map{"status": http.StatusServiceUnavailable, "nats-status": p.nc.Status()})
		}
		return c.JSON(http.StatusOK, echo.Map{"status": http.StatusOK, "nats-status": p.nc.Status()})
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

		logrus.Debugf("graphql.proxy: %s", c.Request().URL.String())
		c.Request().URL.Host = p.graphQLHost.Host

		gqlProxy.ServeHTTP(c.Response(), c.Request())
		logrus.Debugf("graphql.proxy: %d", c.Response().Status)
		return nil
	}
}

func (p *Proxy) k8s() echo.HandlerFunc {
	return func(c echo.Context) error {
		logrus.Debugf("incoming request url: %s", c.Request().URL.String())

		tc, err := p.reviewToken(c.Request().Header)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to fetch ca ca token")
			return echo.NewHTTPError(http.StatusBadRequest, echo.Map{"message": fmt.Sprintf("%s - %s", errUnableToValidateToken, err)})
		}

		// Work on a copy of the request, RoundTrip should never modify the request:
		// https://golang.org/src/net/http/client.go#L103
		reqCopy := SanitizeRequest(c.Request())
		reqCopy.URL.Path = parseDestinationPath(c) // k8s/path -> path

		cid := tc.Audience
		if cid != p.config.EnvID {
			message := fmt.Sprintf(errInvalidEnvID, cid, p.config.EnvID)
			logrus.Warnf(message)
			return echo.NewHTTPError(http.StatusBadRequest, echo.Map{"message": message})
		}

		logrus.Debug("Token is valid")

		iCfg, err := impersonationConfigForUser(tc.User)
		if err != nil {
			logrus.WithError(err).Warn(errFailedToGetImpersonationConfig)
			return echo.NewHTTPError(http.StatusBadRequest, echo.Map{"message": fmt.Sprintf(errFailedToGetImpersonationConfig+": %s", err)})
		}

		irt := transport.NewImpersonatingRoundTripper(iCfg, p.kubeTransport)
		k8sProxy := httputil.NewSingleHostReverseProxy(p.kubeHost)
		k8sProxy.Transport = irt
		k8sProxy.ErrorHandler = p.error
		if isWatchRequest(c) {
			// ?watch needs headers returned to client because it's an http streaming connection.
			// https://golang.org/src/net/http/httputil/reverseproxy.go - Line 374 shows -1 for
			// Conent-Type = "text/event-stream"
			// TODO: confirm the exact ideal behavior here for network. Should it be 200 ms like
			// https://github.com/kubernetes/kubernetes/blob/323f34858de18b862d43c40b2cced65ad8e24052/staging/src/k8s.io/apimachinery/pkg/util/proxy/upgradeaware.go#L85
			k8sProxy.FlushInterval = watchFlushInterval
		}

		k8sProxy.ServeHTTP(c.Response(), reqCopy)
		return nil
	}
}

func parseDestinationPath(c echo.Context) string {
	// We swallow the error because "/" with no additional info is valid.
	return c.Param(proxyPathArg)
}

// SanitizeRequest creates a shallow copy of the request along with a deep copy of the allowed Headers.
func SanitizeRequest(req *http.Request) *http.Request {
	r := new(http.Request)

	// shallow clone
	*r = *req

	// deep copy headers
	r.Header = CloneAllowedHeaders(req.Header)

	return r
}

// CloneAllowedHeaders deep copies allowed headers
func CloneAllowedHeaders(in http.Header) http.Header {
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
	// TODO(soorena776): there is not a `Error` method in the new logging
	// interface. We might want to change this and use a different logger.
	logrus.Warn("unknown error", "err", err, "remote-addr", r.RemoteAddr)
	http.Error(rw, "", http.StatusInternalServerError)
}

func (p *Proxy) reviewToken(requestHeaders http.Header) (*TokenClaims, error) {
	auth := strings.TrimSpace(requestHeaders.Get(headerAuthorization))
	if auth == "" {
		return nil, errors.New(errMissingAuthHeader)
	}
	parts := strings.Split(auth, " ")
	if len(parts) < 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, errors.New(errMissingBearer)
	}

	tokenStr := parts[1]
	tcs := &TokenClaims{}
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

func impersonationConfigForUser(u internal.CrossplaneAccessor) (transport.ImpersonationConfig, error) {
	user := u.Identifier
	groups := u.TeamIDs
	isOwner := u.IsOwner

	logrus.Debug("User info", "isowner", isOwner, "user", user, "groups", groups)

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

// isWatchRequest detects k8s requests that include the ?watch=true param indicating that we should stream response data
func isWatchRequest(c echo.Context) bool {
	if queryParam := c.QueryParam("watch"); queryParam != "" {
		if isWatch, _ := strconv.ParseBool(queryParam); isWatch {
			return true
		}
	}
	return false
}
