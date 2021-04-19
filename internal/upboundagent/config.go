package upboundagent

import (
	"crypto/rsa"
	"crypto/x509"
)

// NATSClientConfig is the configuration for a NATS Client
type NATSClientConfig struct {
	Name     string
	Endpoint string
	// JWTEndpoint is the Upbound api endpoint for fetching NATS JWT for platforms
	JWTEndpoint string
	// ControlPlaneToken is the token to authenticate against JWTEndpoint
	ControlPlaneToken string
	CABundleFile      string
}

// Config maintains the configurations for the upbound agent
type Config struct {
	// DebugMode enables debug level logging
	DebugMode         bool
	EnvID             string
	TokenRSAPublicKey *rsa.PublicKey
	GraphQLHost       string
	GraphQLCACertPool *x509.CertPool
	NATS              *NATSClientConfig
}
