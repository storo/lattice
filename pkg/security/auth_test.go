package security

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestAuth_APIKeyFromHeader(t *testing.T) {
	auth := NewAuth(
		WithAPIKeyAuth(NewAPIKeyAuth()),
	)

	// Register an API key
	auth.apiKey.RegisterKey("test-api-key", &KeyEntry{
		AgentID: "agent-1",
		Roles:   []string{"read"},
	})

	// Create a request with the API key header
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-api-key")

	ctx := context.Background()
	claims, err := auth.AuthenticateRequest(ctx, req)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	if claims.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", claims.AgentID)
	}
}

func TestAuth_JWTFromHeader(t *testing.T) {
	jwtAuth := NewJWTAuth("test-secret-key-that-is-long-enough")
	auth := NewAuth(
		WithJWTAuth(jwtAuth),
	)

	// Generate a JWT token
	token, err := jwtAuth.Generate(&JWTClaims{
		AgentID: "agent-1",
		Roles:   []string{"admin"},
	}, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Create a request with the Bearer token
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	ctx := context.Background()
	claims, err := auth.AuthenticateRequest(ctx, req)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	if claims.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", claims.AgentID)
	}

	if len(claims.Roles) != 1 || claims.Roles[0] != "admin" {
		t.Errorf("expected roles [admin], got %v", claims.Roles)
	}
}

func TestAuth_PreferJWTOverAPIKey(t *testing.T) {
	apiKeyAuth := NewAPIKeyAuth()
	jwtAuth := NewJWTAuth("test-secret-key-that-is-long-enough")

	auth := NewAuth(
		WithAPIKeyAuth(apiKeyAuth),
		WithJWTAuth(jwtAuth),
	)

	// Register an API key
	apiKeyAuth.RegisterKey("test-api-key", &KeyEntry{
		AgentID: "api-agent",
	})

	// Generate a JWT token
	token, _ := jwtAuth.Generate(&JWTClaims{
		AgentID: "jwt-agent",
	}, time.Hour)

	// Create a request with BOTH auth methods
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	req.Header.Set("Authorization", "Bearer "+token)

	ctx := context.Background()
	claims, err := auth.AuthenticateRequest(ctx, req)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	// Should prefer JWT over API key
	if claims.AgentID != "jwt-agent" {
		t.Errorf("expected JWT auth to take precedence, got agent ID '%s'", claims.AgentID)
	}
}

func TestAuth_NoCredentials(t *testing.T) {
	auth := NewAuth(
		WithAPIKeyAuth(NewAPIKeyAuth()),
		WithJWTAuth(NewJWTAuth("secret")),
	)

	req, _ := http.NewRequest("GET", "/test", nil)

	ctx := context.Background()
	_, err := auth.AuthenticateRequest(ctx, req)
	if err != ErrNoCredentials {
		t.Errorf("expected ErrNoCredentials, got %v", err)
	}
}

func TestAuth_InvalidAPIKey(t *testing.T) {
	auth := NewAuth(
		WithAPIKeyAuth(NewAPIKeyAuth()),
	)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")

	ctx := context.Background()
	_, err := auth.AuthenticateRequest(ctx, req)
	if err != ErrInvalidAPIKey {
		t.Errorf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func TestAuth_InvalidJWT(t *testing.T) {
	auth := NewAuth(
		WithJWTAuth(NewJWTAuth("secret")),
	)

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	ctx := context.Background()
	_, err := auth.AuthenticateRequest(ctx, req)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuth_CustomAPIKeyHeader(t *testing.T) {
	auth := NewAuth(
		WithAPIKeyAuth(NewAPIKeyAuth()),
		WithAPIKeyHeader("X-Custom-Key"),
	)

	// Register an API key
	auth.apiKey.RegisterKey("test-api-key", &KeyEntry{
		AgentID: "agent-1",
	})

	// Create a request with the custom header
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Custom-Key", "test-api-key")

	ctx := context.Background()
	claims, err := auth.AuthenticateRequest(ctx, req)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	if claims.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", claims.AgentID)
	}
}

func TestAuth_QueryParam(t *testing.T) {
	auth := NewAuth(
		WithAPIKeyAuth(NewAPIKeyAuth()),
		WithQueryParamAuth("api_key"),
	)

	// Register an API key
	auth.apiKey.RegisterKey("test-api-key", &KeyEntry{
		AgentID: "agent-1",
	})

	// Create a request with the API key in query param
	req, _ := http.NewRequest("GET", "/test?api_key=test-api-key", nil)

	ctx := context.Background()
	claims, err := auth.AuthenticateRequest(ctx, req)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	if claims.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", claims.AgentID)
	}
}

func TestAuth_HasRole(t *testing.T) {
	claims := &AuthClaims{
		AgentID: "agent-1",
		Roles:   []string{"admin", "read"},
	}

	if !claims.HasRole("admin") {
		t.Error("expected HasRole('admin') to be true")
	}

	if !claims.HasRole("read") {
		t.Error("expected HasRole('read') to be true")
	}

	if claims.HasRole("write") {
		t.Error("expected HasRole('write') to be false")
	}
}

func TestAuth_HasPermission(t *testing.T) {
	claims := &AuthClaims{
		AgentID:     "agent-1",
		Permissions: []string{"api:read", "api:write"},
	}

	if !claims.HasPermission("api:read") {
		t.Error("expected HasPermission('api:read') to be true")
	}

	if claims.HasPermission("api:delete") {
		t.Error("expected HasPermission('api:delete') to be false")
	}
}

func TestAuth_HasAnyRole(t *testing.T) {
	claims := &AuthClaims{
		AgentID: "agent-1",
		Roles:   []string{"viewer"},
	}

	if !claims.HasAnyRole("admin", "viewer", "editor") {
		t.Error("expected HasAnyRole to return true for 'viewer'")
	}

	if claims.HasAnyRole("admin", "editor") {
		t.Error("expected HasAnyRole to return false when no match")
	}
}

func TestAuth_HasAllRoles(t *testing.T) {
	claims := &AuthClaims{
		AgentID: "agent-1",
		Roles:   []string{"admin", "viewer", "editor"},
	}

	if !claims.HasAllRoles("admin", "viewer") {
		t.Error("expected HasAllRoles to return true")
	}

	if claims.HasAllRoles("admin", "superuser") {
		t.Error("expected HasAllRoles to return false when missing role")
	}
}
