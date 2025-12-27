package security

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT-specific errors
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrRevokedToken = errors.New("token revoked")
)

// JWTClaims contains the claims for a JWT token.
type JWTClaims struct {
	// AgentID is the identifier of the agent this token authenticates.
	AgentID string `json:"agent_id"`

	// Roles are the roles granted to this token.
	Roles []string `json:"roles,omitempty"`

	// Permissions are specific permissions for this token.
	Permissions []string `json:"permissions,omitempty"`

	jwt.RegisteredClaims
}

// JWTAuth provides JWT token authentication.
type JWTAuth struct {
	mu      sync.RWMutex
	secret  []byte
	revoked map[string]time.Time // token -> expiration time
}

// NewJWTAuth creates a new JWT authenticator with the given secret.
func NewJWTAuth(secret string) *JWTAuth {
	return &JWTAuth{
		secret:  []byte(secret),
		revoked: make(map[string]time.Time),
	}
}

// Generate creates a new JWT token with the given claims and duration.
func (a *JWTAuth) Generate(claims *JWTClaims, duration time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(duration)

	// Set standard claims
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}

// Validate validates a JWT token and returns the claims if valid.
func (a *JWTAuth) Validate(ctx context.Context, tokenString string) (*JWTClaims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	// Check if token is revoked
	a.mu.RLock()
	_, revoked := a.revoked[tokenString]
	a.mu.RUnlock()

	if revoked {
		return nil, ErrRevokedToken
	}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return a.secret, nil
	})

	if err != nil {
		// Check for specific errors
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Refresh validates an existing token and generates a new one with the same claims
// but a new expiration time.
func (a *JWTAuth) Refresh(ctx context.Context, tokenString string, duration time.Duration) (string, error) {
	// Validate the existing token
	claims, err := a.Validate(ctx, tokenString)
	if err != nil {
		return "", err
	}

	// Generate a new token with the same claims but new expiration
	newClaims := &JWTClaims{
		AgentID:     claims.AgentID,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	}

	return a.Generate(newClaims, duration)
}

// Revoke adds a token to the revocation list.
// The token will be rejected until it naturally expires and is cleaned up.
func (a *JWTAuth) Revoke(tokenString string) {
	// Parse the token to get expiration time (don't validate signature for revocation)
	token, _ := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return a.secret, nil
	})

	var expiresAt time.Time
	if token != nil {
		if claims, ok := token.Claims.(*JWTClaims); ok && claims.ExpiresAt != nil {
			expiresAt = claims.ExpiresAt.Time
		}
	}

	// If we couldn't get expiration, set a default (24 hours)
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	a.mu.Lock()
	a.revoked[tokenString] = expiresAt
	a.mu.Unlock()
}

// CleanupRevoked removes expired tokens from the revocation list.
// This should be called periodically to prevent the revocation list from growing indefinitely.
func (a *JWTAuth) CleanupRevoked() {
	now := time.Now()

	a.mu.Lock()
	defer a.mu.Unlock()

	for token, expiresAt := range a.revoked {
		if now.After(expiresAt) {
			delete(a.revoked, token)
		}
	}
}
