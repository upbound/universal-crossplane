package upboundagent

import (
	"crypto/rsa"
	"crypto/x509"
)

// NATSClientConfig is the configuration for a NATS Client
type NATSClientConfig struct {
	Name     string
	Endpoint string
	// JWTEndpoint is the Upbound API endpoint for fetching NATS JWT for control planes
	JWTEndpoint string
	// ControlPlaneToken is the token to authenticate against JWTEndpoint
	ControlPlaneToken string
	CABundleFile      string
}

// Config maintains the configurations for the Upbound Agent
type Config struct {
	// DebugMode enables debug level logging
	DebugMode         bool
	ControlPlaneID    string
	TokenRSAPublicKey *rsa.PublicKey
	GraphQLCACertPool *x509.CertPool
	NATS              *NATSClientConfig
}
