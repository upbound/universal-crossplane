package main

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/upbound/upbound-runtime/pkg/monitoring"
	"github.com/upbound/upbound-runtime/pkg/monitoring/metrics"
	"github.com/upbound/upbound-runtime/pkg/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/upbound/universal-crossplane/internal/upboundagent"
	"github.com/upbound/universal-crossplane/internal/version"
)

const (
	serviceName = "upbound-agent"

	prefixPlatformTokenSubject = "controlPlane|"
)

const (
	errMalformedCPToken          = "malformed control plane token"
	errCPTokenNoSubjectKey       = "failed to get value for key \"sub\""
	errCPTokenSubjectIsNotString = "failed to parse value for key \"sub\" as a string"
	errCPIDInTokenNotValidUUID   = "control plane id in token is not a valid UUID: %s"
)

// Context represents a cli context
type Context struct {
	Debug bool
}

// AgentCmd represents the "upbound-agent" command
type AgentCmd struct {
	PodName             string `help:"Name of the agent pod."`
	ServerPort          string `default:"6443" help:"Port to serve agent service."`
	TLSCertFile         string `help:"File containing the default x509 Certificate for HTTPS."`
	TLSKeyFile          string `help:"File containing the default x509 private key matching provided cert"`
	GraphqlCABundleFile string `help:"CA bundle file for graphql server"`
	NATSEndpoint        string `help:"Endpoint for nats"`
	NATSJwtEndpoint     string `help:"Endpoint for nats jwt tokens"`
	ControlPlaneToken   string `help:"Platform token to access Upbound Cloud connect endpoint"`
	JWTPublicKey        string `help:"BASE64 encoded rsa public key to validate jwt tokens."`
}

var cli struct {
	Debug bool `help:"Enable debug mode"`

	Agent AgentCmd `cmd:"" help:"Runs Upbound Agent"`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{Debug: cli.Debug})
	ctx.FatalIfErrorf(err)
}

// Run runs the agent command
func (a *AgentCmd) Run(ctx *Context) error {
	debug := ctx.Debug
	podName := a.PodName
	serverPort := a.ServerPort
	tlsCertFile := a.TLSCertFile
	tlsKeyFile := a.TLSKeyFile
	graphqlCABundleFile := a.GraphqlCABundleFile

	natsEndpoint := a.NATSEndpoint
	natsJWTEndpoint := a.NATSJwtEndpoint

	cpToken := a.ControlPlaneToken
	jwtPublicKey := a.JWTPublicKey

	// Logging config
	logrus.SetFormatter(monitoring.NewFormatterWithServiceVersion(serviceName, version.Version))
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.Debug("Command flag values: ",
		fmt.Sprintf("%s: %v, ", "debug", debug),
		fmt.Sprintf("%s: %v, ", "pod-name", podName),
		fmt.Sprintf("%s: %v, ", "server-port", serverPort),
		fmt.Sprintf("%s: %v, ", "tls-cert-file", tlsCertFile),
		fmt.Sprintf("%s: %v, ", "tls-private-key-file", tlsKeyFile),
		fmt.Sprintf("%s: %v, ", "graphql-cabundle-file", graphqlCABundleFile),
		fmt.Sprintf("%s: %v, ", "nats-endpoint", natsEndpoint),
		fmt.Sprintf("%s: %v, ", "nats-jwt-endpoint", natsJWTEndpoint))

	cpID, err := readCPIDFromToken(cpToken)
	if err != nil {
		return errors.Wrap(err, "failed to read control plane id from token")
	}

	pem, err := base64.StdEncoding.DecodeString(jwtPublicKey)
	if err != nil {
		return errors.Wrap(err, "failed to base64 decode provided jwt public key")
	}
	pk, err := jwt.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		return errors.Wrap(err, "failed to parse public key")
	}

	nc := &upboundagent.NATSClientConfig{
		Name:              podName,
		Endpoint:          natsEndpoint,
		JWTEndpoint:       natsJWTEndpoint,
		ControlPlaneToken: cpToken,
	}

	var graphqlCertPool *x509.CertPool
	if graphqlCABundleFile != "" {
		b, err := ioutil.ReadFile(filepath.Clean(graphqlCABundleFile))
		if err != nil {
			return errors.Wrap(err, "failed to read graphql ca bundle file")
		}
		graphqlCertPool, err = generateTrustedCertPool(b)
		if err != nil {
			return errors.Wrap(err, "failed to generate graphql ca cert pool")
		}
	}

	tgConfig := &upboundagent.Config{
		DebugMode:         debug,
		EnvID:             cpID,
		TokenRSAPublicKey: pk,
		GraphQLHost:       "https://crossplane-graphql",
		GraphQLCACertPool: graphqlCertPool,
		NATS:              nc,
	}

	restConfig, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get rest config")
	}
	kubeClusterID, err := readKubeClusterID(restConfig)
	if err != nil {
		return errors.Wrap(err, "failed to read kube cluster ID")
	}
	metricsAddr := "0"
	tc := metrics.TelemetryConfig{
		MetricServerAddr:  metricsAddr,
		HTTPClientMetrics: true,
		HTTPServerMetrics: true,
	}
	pxy := upboundagent.NewProxy(tgConfig, restConfig, kubeClusterID)

	logrus.Info("Starting the Upbound agent", "controlPlaneID", cpID)

	serverAddr := fmt.Sprintf(":%s", serverPort)
	s := service.NewService(pxy, debug, serviceName, "", serverAddr,
		"", version.Version, 0.0, 200, 400,
		false, tlsCertFile, tlsKeyFile, true, tc)
	s.Run()

	return nil
}

func generateTrustedCertPool(b []byte) (*x509.CertPool, error) {
	rootCAs := x509.NewCertPool()

	if ok := rootCAs.AppendCertsFromPEM(b); !ok {
		return nil, errors.New("no certs appended, ca cert should have been appended")
	}

	return rootCAs, nil
}

func readCPIDFromToken(t string) (string, error) {
	// Read control-plane id from the token
	token, err := jwt.Parse(t, nil)
	if err.(*jwt.ValidationError).Errors == jwt.ValidationErrorMalformed {
		return "", errors.Wrap(err, errMalformedCPToken)
	}

	cl := token.Claims.(jwt.MapClaims)
	v, ok := cl["sub"]
	if !ok {
		return "", errors.New(errCPTokenNoSubjectKey)
	}
	s, ok := v.(string)
	if !ok {
		return "", errors.New(errCPTokenSubjectIsNotString)
	}

	envID := strings.TrimPrefix(s, prefixPlatformTokenSubject)
	_, err = uuid.Parse(envID)
	return envID, errors.Wrapf(err, errCPIDInTokenNotValidUUID, envID)
}

func readKubeClusterID(restConfig *rest.Config) (string, error) {
	kube, err := client.New(restConfig, client.Options{})
	if err != nil {
		return "", errors.Wrap(err, "failed to initialize kubernetes client")
	}
	ns := &corev1.Namespace{}
	if err := kube.Get(context.Background(), types.NamespacedName{Name: "kube-system"}, ns); err != nil {
		return "", errors.Wrap(err, "failed to get kube-system namespace")
	}
	if ns.GetUID() == "" {
		return "", errors.New("metadata.uid of kube-system namespace is empty")
	}
	return string(ns.GetUID()), nil
}
