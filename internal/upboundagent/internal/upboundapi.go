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
