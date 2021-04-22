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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/upbound/universal-crossplane/internal/upboundagent"
	"github.com/upbound/universal-crossplane/internal/version"
)

const (
	prefixPlatformTokenSubject = "controlPlane|"
)

const (
	errMalformedCPToken          = "malformed control plane token"
	errCPTokenNoSubjectKey       = "failed to get value for key \"sub\""
	errCPTokenSubjectIsNotString = "failed to parse value for key \"sub\" as a string"
	errCPIDInTokenNotValidUUID   = "control plane id in token is not a valid UUID: %s"
	errFailedToGetKubeSystemNS   = "failed to get kube-system namespace"
	errKubeSystemUIDEmpty        = "metadata.uid of kube-system namespace is empty"
)

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

func main() { // nolint:gocyclo
	ctx := kong.Parse(&cli)
	a := cli.Agent

	if cli.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	cpID, err := readCPIDFromToken(a.ControlPlaneToken)
	if err != nil {
		ctx.FatalIfErrorf(errors.Wrap(err, "failed to read control plane id from token"))
	}

	pem, err := base64.StdEncoding.DecodeString(a.JWTPublicKey)
	if err != nil {
		ctx.FatalIfErrorf(errors.Wrap(err, "failed to base64 decode provided jwt public key"))
	}
	pk, err := jwt.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		ctx.FatalIfErrorf(errors.Wrap(err, "failed to parse public key"))
	}

	var graphqlCertPool *x509.CertPool
	if a.GraphqlCABundleFile != "" {
		b, err := ioutil.ReadFile(filepath.Clean(a.GraphqlCABundleFile))
		if err != nil {
			ctx.FatalIfErrorf(errors.Wrap(err, "failed to read graphql ca bundle file"))
		}
		graphqlCertPool, err = generateTrustedCertPool(b)
		if err != nil {
			ctx.FatalIfErrorf(errors.Wrap(err, "failed to generate graphql ca cert pool"))
		}
	}

	tgConfig := &upboundagent.Config{
		DebugMode:         cli.Debug,
		ControlPlaneID:    cpID,
		TokenRSAPublicKey: pk,
		GraphQLCACertPool: graphqlCertPool,
		NATS: &upboundagent.NATSClientConfig{
			Name:              a.PodName,
			Endpoint:          a.NATSEndpoint,
			JWTEndpoint:       a.NATSJwtEndpoint,
			ControlPlaneToken: a.ControlPlaneToken,
		},
	}

	restConfig, err := config.GetConfig()
	if err != nil {
		ctx.FatalIfErrorf(errors.Wrap(err, "failed to get rest config"))
	}
	kube, err := client.New(restConfig, client.Options{})
	if err != nil {
		ctx.FatalIfErrorf(errors.Wrap(err, "failed to initialize kubernetes client"))
	}
	kubeClusterID, err := readKubeClusterID(kube)
	if err != nil {
		ctx.FatalIfErrorf(errors.Wrap(err, "failed to read kube cluster ID"))
	}

	pxy := upboundagent.NewProxy(tgConfig, restConfig, kubeClusterID)

	logrus.Info("Starting Upbound Agent ",
		fmt.Sprintf("%s: %v, ", "version", version.Version),
		fmt.Sprintf("%s: %v, ", "control-plane-id", cpID),
		fmt.Sprintf("%s: %v, ", "debug", cli.Debug),
		fmt.Sprintf("%s: %v, ", "pod-name", a.PodName),
		fmt.Sprintf("%s: %v, ", "server-port", a.ServerPort),
		fmt.Sprintf("%s: %v, ", "tls-cert-file", a.TLSCertFile),
		fmt.Sprintf("%s: %v, ", "tls-private-key-file", a.TLSKeyFile),
		fmt.Sprintf("%s: %v, ", "graphql-cabundle-file", a.GraphqlCABundleFile),
		fmt.Sprintf("%s: %v, ", "nats-endpoint", a.NATSEndpoint),
		fmt.Sprintf("%s: %v", "nats-jwt-endpoint", a.NATSJwtEndpoint))

	addr := fmt.Sprintf(":%s", a.ServerPort)
	ctx.FatalIfErrorf(errors.Wrap(pxy.Run(addr, a.TLSCertFile, a.TLSKeyFile), "cannot run upbound agent proxy"))
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

func readKubeClusterID(kube client.Client) (string, error) {
	ns := &corev1.Namespace{}
	if err := kube.Get(context.Background(), types.NamespacedName{Name: "kube-system"}, ns); err != nil {
		return "", errors.Wrap(err, errFailedToGetKubeSystemNS)
	}
	if ns.GetUID() == "" {
		return "", errors.New(errKubeSystemUIDEmpty)
	}
	return string(ns.GetUID()), nil
}
