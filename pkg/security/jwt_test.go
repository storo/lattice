package security

import (
	"context"
	"testing"
	"time"
)

func TestJWTAuth_GenerateAndValidate(t *testing.T) {
	auth := NewJWTAuth("test-secret-key-that-is-long-enough")

	claims := &JWTClaims{
		AgentID:     "agent-1",
		Roles:       []string{"read", "write"},
		Permissions: []string{"api:read"},
	}

	token, err := auth.Generate(claims, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("expected non-empty token")
	}

	// Validate the token
	ctx := context.Background()
	validated, err := auth.Validate(ctx, token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if validated.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", validated.AgentID)
	}

	if len(validated.Roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(validated.Roles))
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	auth := NewJWTAuth("test-secret-key-that-is-long-enough")

	claims := &JWTClaims{
		AgentID: "agent-1",
	}

	// Generate a token that expires immediately
	token, err := auth.Generate(claims, -time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	ctx := context.Background()
	_, err = auth.Validate(ctx, token)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	auth := NewJWTAuth("test-secret-key-that-is-long-enough")

	ctx := context.Background()
	_, err := auth.Validate(ctx, "invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTAuth_TamperedToken(t *testing.T) {
	auth := NewJWTAuth("test-secret-key-that-is-long-enough")

	claims := &JWTClaims{
		AgentID: "agent-1",
	}

	token, err := auth.Generate(claims, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Tamper with the token
	tamperedToken := token[:len(token)-5] + "xxxxx"

	ctx := context.Background()
	_, err = auth.Validate(ctx, tamperedToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for tampered token, got %v", err)
	}
}

func TestJWTAuth_WrongSecret(t *testing.T) {
	auth1 := NewJWTAuth("secret-key-one-that-is-long-enough")
	auth2 := NewJWTAuth("secret-key-two-that-is-long-enough")

	claims := &JWTClaims{
		AgentID: "agent-1",
	}

	// Generate with auth1
	token, err := auth1.Generate(claims, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Try to validate with auth2
	ctx := context.Background()
	_, err = auth2.Validate(ctx, token)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for wrong secret, got %v", err)
	}
}

func TestJWTAuth_EmptyToken(t *testing.T) {
	auth := NewJWTAuth("test-secret-key-that-is-long-enough")

	ctx := context.Background()
	_, err := auth.Validate(ctx, "")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for empty token, got %v", err)
	}
}

func TestJWTAuth_Refresh(t *testing.T) {
	auth := NewJWTAuth("test-secret-key-that-is-long-enough")

	claims := &JWTClaims{
		AgentID:     "agent-1",
		Roles:       []string{"admin"},
		Permissions: []string{"all"},
	}

	// Generate original token
	originalToken, err := auth.Generate(claims, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Refresh the token
	ctx := context.Background()
	newToken, err := auth.Refresh(ctx, originalToken, time.Hour*2)
	if err != nil {
		t.Fatalf("failed to refresh token: %v", err)
	}

	if newToken == originalToken {
		t.Error("refreshed token should be different from original")
	}

	// Validate the new token
	validated, err := auth.Validate(ctx, newToken)
	if err != nil {
		t.Fatalf("failed to validate refreshed token: %v", err)
	}

	if validated.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", validated.AgentID)
	}

	if len(validated.Roles) != 1 || validated.Roles[0] != "admin" {
		t.Errorf("expected roles [admin], got %v", validated.Roles)
	}
}

func TestJWTAuth_Revoke(t *testing.T) {
	auth := NewJWTAuth("test-secret-key-that-is-long-enough")

	claims := &JWTClaims{
		AgentID: "agent-1",
	}

	token, err := auth.Generate(claims, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Token should be valid initially
	ctx := context.Background()
	_, err = auth.Validate(ctx, token)
	if err != nil {
		t.Fatalf("token should be valid: %v", err)
	}

	// Revoke the token
	auth.Revoke(token)

	// Token should now be invalid
	_, err = auth.Validate(ctx, token)
	if err != ErrRevokedToken {
		t.Errorf("expected ErrRevokedToken, got %v", err)
	}
}

func TestJWTAuth_CleanupRevoked(t *testing.T) {
	auth := NewJWTAuth("test-secret-key-that-is-long-enough")

	claims := &JWTClaims{
		AgentID: "agent-1",
	}

	// Generate a token that expires quickly
	token, err := auth.Generate(claims, time.Millisecond*100)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Revoke it
	auth.Revoke(token)

	// Wait for expiration
	time.Sleep(time.Millisecond * 150)

	// Cleanup expired tokens from revocation list
	auth.CleanupRevoked()

	// Verify internal state was cleaned
	auth.mu.RLock()
	_, exists := auth.revoked[token]
	auth.mu.RUnlock()

	if exists {
		t.Error("expired token should have been cleaned from revocation list")
	}
}
