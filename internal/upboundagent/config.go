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
	CABundle          string
}

// Config maintains the configurations for the Upbound Agent
type Config struct {
	// DebugMode enables debug level logging
	DebugMode         bool
	ControlPlaneID    string
	TokenRSAPublicKey *rsa.PublicKey
	XGQLCACertPool    *x509.CertPool
	NATS              *NATSClientConfig
}
