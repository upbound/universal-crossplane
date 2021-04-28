package internal

import "github.com/dgrijalva/jwt-go"

// CrossplaneAccessor is the struct holding accessor info in JWT custom claims
type CrossplaneAccessor struct {
	Groups    []string `json:"groups"`
	UpboundID string   `json:"upboundID"`
}

// TokenClaims is the struct for custom claims of JWT token
type TokenClaims struct {
	Payload CrossplaneAccessor `json:"payload"`
	jwt.StandardClaims
}
