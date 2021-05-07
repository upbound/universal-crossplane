/*
Copyright 2021 Upbound Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package internal

import "github.com/dgrijalva/jwt-go"

// CrossplaneAccessor is the struct holding accessor info in JWT custom claims
type CrossplaneAccessor struct {
	// Groups is the list of groups that the agent should impersonate for this
	// given access.
	Groups []string `json:"groups"`

	// UpboundID is the identifier from Upbound that will be added to metadata
	// of the impersonation config.
	UpboundID string `json:"upboundID"`
}

// TokenClaims is the struct for custom claims of JWT token
type TokenClaims struct {
	Payload CrossplaneAccessor `json:"payload"`
	jwt.StandardClaims
}
