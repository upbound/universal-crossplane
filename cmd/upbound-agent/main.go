package main

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	flagDebug               = "debug"
	flagPodName             = "pod-name"
	flagServerPort          = "server-port"
	flagTLSCertFile         = "tls-cert-file"
	flagTLSKeyFile          = "tls-private-key-file"
	flagGraphqlCABundleFile = "graphql-cabundle-file"
	flagControlPlaneToken   = "control-plane-token"
	flagNATSEndpoint        = "nats-endpoint"
	flagNATSJWTEndpoint     = "nats-jwt-endpoint"
	flagJWTPublicKey        = "jwt-public-key"
	flagEnableMetrics       = "enable-metrics"
	flagTraceSampleFraction = "trace-sample-fraction"

	defaultServerPort          = "6443"
	prefixPlatformTokenSubject = "controlPlane|"
)

const (
	errMalformedCPToken          = "malformed control plane token"
	errCPTokenNoSubjectKey       = "failed to get value for key \"sub\""
	errCPTokenSubjectIsNotString = "failed to parse value for key \"sub\" as a string"
	errCPIDInTokenNotValidUUID   = "control plane id in token is not a valid UUID: %s"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error { // nolint:gocyclo
	var cmd = &cobra.Command{
		Use:   serviceName,
		Short: "Upbound agent",
		Run: func(command *cobra.Command, args []string) {
			debug := viper.GetBool(flagDebug)
			podName := viper.GetString(flagPodName)
			serverPort := viper.GetString(flagServerPort)
			tlsCertFile := viper.GetString(flagTLSCertFile)
			tlsKeyFile := viper.GetString(flagTLSKeyFile)
			graphqlCABundleFile := viper.GetString(flagGraphqlCABundleFile)

			natsEndpoint := viper.GetString(flagNATSEndpoint)
			natsJWTEndpoint := viper.GetString(flagNATSJWTEndpoint)

			cpToken := viper.GetString(flagControlPlaneToken)
			jwtPublicKey := viper.GetString(flagJWTPublicKey)

			enableMetrics := viper.GetBool(flagEnableMetrics)
			traceSampleFraction := viper.GetFloat64(flagTraceSampleFraction)

			// Logging config
			logrus.SetFormatter(monitoring.NewFormatterWithServiceVersion(serviceName, version.Version))
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}

			logrus.Debug("Command flag values: ",
				fmt.Sprintf("%s: %v, ", flagDebug, debug),
				fmt.Sprintf("%s: %v, ", flagPodName, podName),
				fmt.Sprintf("%s: %v, ", flagServerPort, serverPort),
				fmt.Sprintf("%s: %v, ", flagTLSCertFile, tlsCertFile),
				fmt.Sprintf("%s: %v, ", flagTLSKeyFile, tlsKeyFile),
				fmt.Sprintf("%s: %v, ", flagGraphqlCABundleFile, graphqlCABundleFile),
				fmt.Sprintf("%s: %v, ", flagNATSEndpoint, natsEndpoint),
				fmt.Sprintf("%s: %v, ", flagNATSJWTEndpoint, natsJWTEndpoint),
				fmt.Sprintf("%s: %v, ", flagEnableMetrics, enableMetrics),
				fmt.Sprintf("%s: %v", flagTraceSampleFraction, traceSampleFraction))

			cpID, err := readCPIDFromToken(cpToken)
			if err != nil {
				logrus.Fatal(errors.Wrap(err, "failed to read control plane id from token"))
			}

			pem, err := base64.StdEncoding.DecodeString(jwtPublicKey)
			if err != nil {
				logrus.Fatal(errors.Wrap(err, "failed to base64 decode provided jwt public key"))
			}
			pk, err := jwt.ParseRSAPublicKeyFromPEM(pem)
			if err != nil {
				logrus.Fatal(errors.Wrap(err, "failed to parse public key"))
			}

			var nc *upboundagent.NATSClientConfig
			if viper.GetString(flagNATSEndpoint) != "" {
				logrus.Info("Enabling NATS")
				nc = &upboundagent.NATSClientConfig{
					Name:              podName,
					Endpoint:          natsEndpoint,
					JWTEndpoint:       natsJWTEndpoint,
					ControlPlaneToken: cpToken,
				}
			}
			var graphqlCertPool *x509.CertPool
			if graphqlCABundleFile != "" {
				b, err := ioutil.ReadFile(filepath.Clean(graphqlCABundleFile))
				if err != nil {
					logrus.Fatal(errors.Wrap(err, "failed to read graphql ca bundle file"))
				}
				graphqlCertPool, err = generateTrustedCertPool(b)
				if err != nil {
					logrus.Fatal(errors.Wrap(err, "failed to generate graphql ca cert pool"))
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
				logrus.Fatal(errors.Wrap(err, "failed to get rest config"))
			}
			kubeClusterID, err := readKubeClusterID(restConfig)
			if err != nil {
				logrus.Fatal(errors.Wrap(err, "failed to read kube cluster ID"))
			}
			metricsAddr := "0"
			if enableMetrics {
				metricsAddr = ""
			}
			tc := metrics.TelemetryConfig{
				MetricServerAddr:  metricsAddr,
				HTTPClientMetrics: true,
				HTTPServerMetrics: true,
			}
			pxy := upboundagent.NewProxy(tgConfig, restConfig, kubeClusterID)

			logrus.Info("Starting the Upbound agent", "controlPlaneID", cpID)

			serverAddr := fmt.Sprintf(":%s", serverPort)
			s := service.NewService(pxy, debug, serviceName, "", serverAddr,
				"", version.Version, traceSampleFraction, 200, 400,
				false, tlsCertFile, tlsKeyFile, true, tc)
			s.Run()
		},
	}
	flags := cmd.Flags()
	flagData := []struct {
		key          string
		usage        string
		defaultValue string
		required     bool
	}{
		{
			key:   flagDebug,
			usage: "true or false",
		},
		{
			key:      flagPodName,
			usage:    "name of the agent pod",
			required: true,
		},
		{
			key:          flagServerPort,
			defaultValue: defaultServerPort,
			usage:        "port to serve agent service",
		},
		{
			key:      flagTLSCertFile,
			usage:    "file containing the default x509 Certificate for HTTPS.",
			required: true,
		},
		{
			key:      flagTLSKeyFile,
			usage:    "file containing the default x509 private key matching provided cert",
			required: true,
		},
		{
			key:   flagGraphqlCABundleFile,
			usage: "ca bundle file for graphql server",
		},
		{
			key:   flagControlPlaneToken,
			usage: "platform token to access Upbound Cloud connect endpoint",
		},
		{
			key:   flagNATSEndpoint,
			usage: "endpoint for nats",
		},
		{
			key:   flagNATSJWTEndpoint,
			usage: "endpoint for nats jwt tokens",
		},
		{
			key:      flagJWTPublicKey,
			usage:    "base64 encoded rsa public key to validate jwt tokens",
			required: true,
		},
		{
			key:      flagEnableMetrics,
			usage:    "enable exporting metrics to Prometheus",
			required: false,
		},
		{
			key:      flagTraceSampleFraction,
			usage:    "tracing sample fraction for trace.ProbabilitySampler. Default value = 0, i.e. tracing disabled",
			required: false,
		},
	}

	viper.AutomaticEnv()
	for _, data := range flagData {
		flags.String(data.key, data.defaultValue, data.usage)
		flag := flags.Lookup(data.key)
		if err := viper.BindPFlag(data.key, flag); err != nil {
			return err
		}
		if data.required && viper.GetString(flag.Name) == "" {
			if err := cmd.MarkFlagRequired(flag.Name); err != nil {
				return errors.Wrapf(err, "failed to set flag %s as required", flag.Name)
			}

		}
	}
	return cmd.Execute()
}

func generateTrustedCertPool(b []byte) (*x509.CertPool, error) {
	rootCAs := x509.NewCertPool()

	if ok := rootCAs.AppendCertsFromPEM(b); !ok {
		return nil, errors.New("no certs appended, ca cert should have been appended")
	}

	return rootCAs, nil
}

func readCPIDFromToken(t string) (string, error) {
	// Read env id from platform token
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
