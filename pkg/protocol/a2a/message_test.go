package a2a

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessage_Create(t *testing.T) {
	msg := NewMessage("agent-1", "agent-2", "Hello, agent!")

	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}

	if msg.From != "agent-1" {
		t.Errorf("expected from 'agent-1', got '%s'", msg.From)
	}

	if msg.To != "agent-2" {
		t.Errorf("expected to 'agent-2', got '%s'", msg.To)
	}

	if msg.Content != "Hello, agent!" {
		t.Errorf("expected content 'Hello, agent!', got '%s'", msg.Content)
	}

	if msg.Type != MessageTypeRequest {
		t.Errorf("expected type '%s', got '%s'", MessageTypeRequest, msg.Type)
	}
}

func TestMessage_Response(t *testing.T) {
	request := NewMessage("agent-1", "agent-2", "What is 2+2?")

	response := request.Reply("4")

	if response.ReplyTo != request.ID {
		t.Errorf("expected reply_to '%s', got '%s'", request.ID, response.ReplyTo)
	}

	if response.From != request.To {
		t.Error("response should be from the original recipient")
	}

	if response.To != request.From {
		t.Error("response should be to the original sender")
	}

	if response.Type != MessageTypeResponse {
		t.Errorf("expected type '%s', got '%s'", MessageTypeResponse, response.Type)
	}
}

func TestMessage_Error(t *testing.T) {
	request := NewMessage("agent-1", "agent-2", "Invalid request")

	errMsg := request.Error(ErrCodeInvalidRequest, "Bad format")

	if errMsg.Type != MessageTypeError {
		t.Errorf("expected type '%s', got '%s'", MessageTypeError, errMsg.Type)
	}

	if errMsg.ErrorCode != ErrCodeInvalidRequest {
		t.Errorf("expected error code '%s', got '%s'", ErrCodeInvalidRequest, errMsg.ErrorCode)
	}

	if errMsg.ErrorMessage != "Bad format" {
		t.Errorf("expected error message 'Bad format', got '%s'", errMsg.ErrorMessage)
	}
}

func TestMessage_Metadata(t *testing.T) {
	msg := NewMessage("a", "b", "test")

	msg.SetMeta("priority", "high")
	msg.SetMeta("retry_count", 3)

	if msg.Metadata["priority"] != "high" {
		t.Error("expected priority metadata")
	}

	if msg.Metadata["retry_count"] != 3 {
		t.Error("expected retry_count metadata")
	}
}

func TestMessage_JSON(t *testing.T) {
	original := NewMessage("agent-1", "agent-2", "Test content")
	original.SetMeta("key", "value")

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Error("ID mismatch")
	}

	if decoded.From != original.From {
		t.Error("From mismatch")
	}

	if decoded.Content != original.Content {
		t.Error("Content mismatch")
	}
}

func TestMessage_TTL(t *testing.T) {
	msg := NewMessage("a", "b", "test")

	// Set TTL to expired
	msg.ExpiresAt = time.Now().Add(-time.Hour)

	if !msg.IsExpired() {
		t.Error("message should be expired")
	}

	// Set TTL to future
	msg.ExpiresAt = time.Now().Add(time.Hour)

	if msg.IsExpired() {
		t.Error("message should not be expired")
	}
}

func TestMessage_Broadcast(t *testing.T) {
	msg := NewBroadcast("agent-1", "Attention all agents!")

	if msg.To != BroadcastAddress {
		t.Errorf("expected broadcast address '%s', got '%s'", BroadcastAddress, msg.To)
	}
}

func TestEnvelope(t *testing.T) {
	msg := NewMessage("a", "b", "test")

	envelope := NewEnvelope(msg)

	if envelope.Message.ID != msg.ID {
		t.Error("envelope should contain the message")
	}

	if envelope.Hops != 0 {
		t.Error("initial hops should be 0")
	}

	envelope.AddHop("router-1")
	envelope.AddHop("router-2")

	if envelope.Hops != 2 {
		t.Errorf("expected 2 hops, got %d", envelope.Hops)
	}

	if len(envelope.Route) != 2 {
		t.Errorf("expected route length 2, got %d", len(envelope.Route))
	}
}
