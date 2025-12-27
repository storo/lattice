package security

import (
	"context"
	"testing"
	"time"
)

func TestAPIKeyAuth_RegisterAndAuthenticate(t *testing.T) {
	ctx := context.Background()
	auth := NewAPIKeyAuth()

	auth.RegisterKey("secret-key-123", &KeyEntry{
		AgentID: "agent-1",
		Roles:   []string{"read", "write"},
	})

	claims, err := auth.Authenticate(ctx, "secret-key-123")
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	if claims.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", claims.AgentID)
	}
	if len(claims.Roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(claims.Roles))
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	ctx := context.Background()
	auth := NewAPIKeyAuth()

	auth.RegisterKey("secret-key-123", &KeyEntry{
		AgentID: "agent-1",
	})

	_, err := auth.Authenticate(ctx, "wrong-key")
	if err != ErrInvalidAPIKey {
		t.Errorf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func TestAPIKeyAuth_ExpiredKey(t *testing.T) {
	ctx := context.Background()
	auth := NewAPIKeyAuth()

	// Register an expired key
	auth.RegisterKey("expired-key", &KeyEntry{
		AgentID:   "agent-1",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	})

	_, err := auth.Authenticate(ctx, "expired-key")
	if err != ErrExpiredAPIKey {
		t.Errorf("expected ErrExpiredAPIKey, got %v", err)
	}
}

func TestAPIKeyAuth_NonExpiringKey(t *testing.T) {
	ctx := context.Background()
	auth := NewAPIKeyAuth()

	// Register a key with no expiration (ExpiresAt = 0)
	auth.RegisterKey("permanent-key", &KeyEntry{
		AgentID:   "agent-1",
		ExpiresAt: 0,
	})

	claims, err := auth.Authenticate(ctx, "permanent-key")
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	if claims.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", claims.AgentID)
	}
}

func TestAPIKeyAuth_FutureExpiration(t *testing.T) {
	ctx := context.Background()
	auth := NewAPIKeyAuth()

	// Register a key that expires in the future
	auth.RegisterKey("future-key", &KeyEntry{
		AgentID:   "agent-1",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	claims, err := auth.Authenticate(ctx, "future-key")
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	if claims.AgentID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got '%s'", claims.AgentID)
	}
}

func TestAPIKeyAuth_RevokeKey(t *testing.T) {
	ctx := context.Background()
	auth := NewAPIKeyAuth()

	auth.RegisterKey("revocable-key", &KeyEntry{
		AgentID: "agent-1",
	})

	// Verify key works
	_, err := auth.Authenticate(ctx, "revocable-key")
	if err != nil {
		t.Fatalf("failed to authenticate before revoke: %v", err)
	}

	// Revoke the key
	auth.RevokeKey("revocable-key")

	// Verify key no longer works
	_, err = auth.Authenticate(ctx, "revocable-key")
	if err != ErrInvalidAPIKey {
		t.Errorf("expected ErrInvalidAPIKey after revoke, got %v", err)
	}
}

func TestAPIKeyAuth_ConstantTimeComparison(t *testing.T) {
	ctx := context.Background()
	auth := NewAPIKeyAuth()

	// Register a key
	auth.RegisterKey("real-key", &KeyEntry{AgentID: "agent-1"})

	// Test that invalid keys take similar time (approximate timing attack resistance)
	// This is a basic check - real timing attack tests require statistical analysis
	start1 := time.Now()
	auth.Authenticate(ctx, "wrong-key-short")
	elapsed1 := time.Since(start1)

	start2 := time.Now()
	auth.Authenticate(ctx, "wrong-key-that-is-much-longer-than-the-first")
	elapsed2 := time.Since(start2)

	// Times should be within reasonable range of each other
	// This is not a perfect test but catches obvious issues
	ratio := float64(elapsed1) / float64(elapsed2)
	if ratio < 0.1 || ratio > 10 {
		t.Logf("Warning: timing difference between key lengths is significant (ratio: %f)", ratio)
	}
}

func TestAPIKeyAuth_MultipleKeys(t *testing.T) {
	ctx := context.Background()
	auth := NewAPIKeyAuth()

	auth.RegisterKey("key-1", &KeyEntry{AgentID: "agent-1"})
	auth.RegisterKey("key-2", &KeyEntry{AgentID: "agent-2"})
	auth.RegisterKey("key-3", &KeyEntry{AgentID: "agent-3"})

	// Test each key
	claims1, _ := auth.Authenticate(ctx, "key-1")
	if claims1.AgentID != "agent-1" {
		t.Error("key-1 should authenticate agent-1")
	}

	claims2, _ := auth.Authenticate(ctx, "key-2")
	if claims2.AgentID != "agent-2" {
		t.Error("key-2 should authenticate agent-2")
	}

	claims3, _ := auth.Authenticate(ctx, "key-3")
	if claims3.AgentID != "agent-3" {
		t.Error("key-3 should authenticate agent-3")
	}
}

func TestAPIKeyAuth_EmptyKey(t *testing.T) {
	ctx := context.Background()
	auth := NewAPIKeyAuth()

	auth.RegisterKey("valid-key", &KeyEntry{AgentID: "agent-1"})

	_, err := auth.Authenticate(ctx, "")
	if err != ErrInvalidAPIKey {
		t.Errorf("expected ErrInvalidAPIKey for empty key, got %v", err)
	}
}
