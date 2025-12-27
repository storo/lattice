package security

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// Auth-specific errors
var (
	ErrNoCredentials = errors.New("no credentials provided")
)

// AuthClaims is the unified claims structure returned by Auth.
type AuthClaims struct {
	// AgentID is the authenticated agent's identifier.
	AgentID string

	// Roles are the roles granted to this agent.
	Roles []string

	// Permissions are specific permissions.
	Permissions []string

	// ExpiresAt is when the credentials expire (0 = never).
	ExpiresAt int64

	// Method is the authentication method used (api_key or jwt).
	Method string
}

// HasRole checks if the claims include a specific role.
func (c *AuthClaims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the claims include any of the specified roles.
func (c *AuthClaims) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if c.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if the claims include all of the specified roles.
func (c *AuthClaims) HasAllRoles(roles ...string) bool {
	for _, role := range roles {
		if !c.HasRole(role) {
			return false
		}
	}
	return true
}

// HasPermission checks if the claims include a specific permission.
func (c *AuthClaims) HasPermission(perm string) bool {
	for _, p := range c.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// Auth provides a unified authentication interface supporting multiple methods.
type Auth struct {
	apiKey       *APIKeyAuth
	jwt          *JWTAuth
	apiKeyHeader string
	queryParam   string
}

// AuthOption configures the Auth instance.
type AuthOption func(*Auth)

// NewAuth creates a new unified authenticator.
func NewAuth(opts ...AuthOption) *Auth {
	a := &Auth{
		apiKeyHeader: "X-API-Key",
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// WithAPIKeyAuth enables API key authentication.
func WithAPIKeyAuth(apiKey *APIKeyAuth) AuthOption {
	return func(a *Auth) {
		a.apiKey = apiKey
	}
}

// WithJWTAuth enables JWT authentication.
func WithJWTAuth(jwt *JWTAuth) AuthOption {
	return func(a *Auth) {
		a.jwt = jwt
	}
}

// WithAPIKeyHeader sets a custom header name for API keys.
func WithAPIKeyHeader(header string) AuthOption {
	return func(a *Auth) {
		a.apiKeyHeader = header
	}
}

// WithQueryParamAuth enables API key authentication via query parameter.
func WithQueryParamAuth(param string) AuthOption {
	return func(a *Auth) {
		a.queryParam = param
	}
}

// AuthenticateRequest attempts to authenticate an HTTP request.
// It checks for credentials in the following order:
// 1. Authorization header (Bearer token for JWT)
// 2. API Key header
// 3. Query parameter (if enabled)
func (a *Auth) AuthenticateRequest(ctx context.Context, req *http.Request) (*AuthClaims, error) {
	// Try JWT from Authorization header first
	if a.jwt != nil {
		authHeader := req.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := a.jwt.Validate(ctx, token)
			if err != nil {
				return nil, err
			}
			return &AuthClaims{
				AgentID:     claims.AgentID,
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
				ExpiresAt:   claims.ExpiresAt.Unix(),
				Method:      "jwt",
			}, nil
		}
	}

	// Try API Key from header
	if a.apiKey != nil {
		apiKey := req.Header.Get(a.apiKeyHeader)
		if apiKey != "" {
			claims, err := a.apiKey.Authenticate(ctx, apiKey)
			if err != nil {
				return nil, err
			}
			return &AuthClaims{
				AgentID:     claims.AgentID,
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
				ExpiresAt:   claims.ExpiresAt,
				Method:      "api_key",
			}, nil
		}
	}

	// Try API Key from query parameter
	if a.apiKey != nil && a.queryParam != "" {
		apiKey := req.URL.Query().Get(a.queryParam)
		if apiKey != "" {
			claims, err := a.apiKey.Authenticate(ctx, apiKey)
			if err != nil {
				return nil, err
			}
			return &AuthClaims{
				AgentID:     claims.AgentID,
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
				ExpiresAt:   claims.ExpiresAt,
				Method:      "api_key",
			}, nil
		}
	}

	return nil, ErrNoCredentials
}

// Authenticate validates credentials directly (without HTTP request).
// Tries JWT first if the credential looks like a JWT, otherwise tries API key.
func (a *Auth) Authenticate(ctx context.Context, credential string) (*AuthClaims, error) {
	if credential == "" {
		return nil, ErrNoCredentials
	}

	// JWT tokens have a specific format (three parts separated by dots)
	if a.jwt != nil && strings.Count(credential, ".") == 2 {
		claims, err := a.jwt.Validate(ctx, credential)
		if err == nil {
			return &AuthClaims{
				AgentID:     claims.AgentID,
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
				ExpiresAt:   claims.ExpiresAt.Unix(),
				Method:      "jwt",
			}, nil
		}
		// If JWT validation fails, don't fall through to API key
		return nil, err
	}

	// Try API key
	if a.apiKey != nil {
		claims, err := a.apiKey.Authenticate(ctx, credential)
		if err != nil {
			return nil, err
		}
		return &AuthClaims{
			AgentID:     claims.AgentID,
			Roles:       claims.Roles,
			Permissions: claims.Permissions,
			ExpiresAt:   claims.ExpiresAt,
			Method:      "api_key",
		}, nil
	}

	return nil, ErrNoCredentials
}
