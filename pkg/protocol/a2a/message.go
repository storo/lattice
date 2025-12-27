package a2a

import (
	"time"

	"github.com/google/uuid"
)

// MessageType defines the type of message.
type MessageType string

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeError    MessageType = "error"
	MessageTypeEvent    MessageType = "event"
)

// Error codes
type ErrorCode string

const (
	ErrCodeInvalidRequest  ErrorCode = "invalid_request"
	ErrCodeNotFound        ErrorCode = "not_found"
	ErrCodeUnauthorized    ErrorCode = "unauthorized"
	ErrCodeTimeout         ErrorCode = "timeout"
	ErrCodeInternalError   ErrorCode = "internal_error"
	ErrCodeCapabilityError ErrorCode = "capability_error"
)

// BroadcastAddress is the special address for broadcast messages.
const BroadcastAddress = "*"

// Message represents a communication between agents.
type Message struct {
	// ID is the unique identifier for this message.
	ID string `json:"id"`

	// Type is the message type.
	Type MessageType `json:"type"`

	// From is the sender agent ID.
	From string `json:"from"`

	// To is the recipient agent ID or broadcast address.
	To string `json:"to"`

	// ReplyTo is the ID of the message this is replying to.
	ReplyTo string `json:"reply_to,omitempty"`

	// Content is the message payload.
	Content string `json:"content"`

	// Metadata contains additional message properties.
	Metadata map[string]any `json:"metadata,omitempty"`

	// Timestamp is when the message was created.
	Timestamp time.Time `json:"timestamp"`

	// ExpiresAt is when the message expires (0 = never).
	ExpiresAt time.Time `json:"expires_at,omitempty"`

	// ErrorCode is set for error messages.
	ErrorCode ErrorCode `json:"error_code,omitempty"`

	// ErrorMessage is the error description.
	ErrorMessage string `json:"error_message,omitempty"`
}

// NewMessage creates a new request message.
func NewMessage(from, to, content string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      MessageTypeRequest,
		From:      from,
		To:        to,
		Content:   content,
		Metadata:  make(map[string]any),
		Timestamp: time.Now(),
	}
}

// NewBroadcast creates a broadcast message to all agents.
func NewBroadcast(from, content string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      MessageTypeEvent,
		From:      from,
		To:        BroadcastAddress,
		Content:   content,
		Metadata:  make(map[string]any),
		Timestamp: time.Now(),
	}
}

// Reply creates a response message.
func (m *Message) Reply(content string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      MessageTypeResponse,
		From:      m.To,
		To:        m.From,
		ReplyTo:   m.ID,
		Content:   content,
		Metadata:  make(map[string]any),
		Timestamp: time.Now(),
	}
}

// Error creates an error message.
func (m *Message) Error(code ErrorCode, message string) *Message {
	return &Message{
		ID:           uuid.New().String(),
		Type:         MessageTypeError,
		From:         m.To,
		To:           m.From,
		ReplyTo:      m.ID,
		ErrorCode:    code,
		ErrorMessage: message,
		Metadata:     make(map[string]any),
		Timestamp:    time.Now(),
	}
}

// SetMeta sets a metadata value.
func (m *Message) SetMeta(key string, value any) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[key] = value
}

// GetMeta retrieves a metadata value.
func (m *Message) GetMeta(key string) (any, bool) {
	if m.Metadata == nil {
		return nil, false
	}
	v, ok := m.Metadata[key]
	return v, ok
}

// IsExpired checks if the message has expired.
func (m *Message) IsExpired() bool {
	if m.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(m.ExpiresAt)
}

// SetTTL sets the time-to-live for the message.
func (m *Message) SetTTL(ttl time.Duration) {
	m.ExpiresAt = time.Now().Add(ttl)
}

// Envelope wraps a message with routing information.
type Envelope struct {
	// Message is the wrapped message.
	Message *Message `json:"message"`

	// Hops is the number of routing hops.
	Hops int `json:"hops"`

	// Route is the list of nodes the message has traversed.
	Route []string `json:"route,omitempty"`

	// TraceID links this message to a trace.
	TraceID string `json:"trace_id,omitempty"`
}

// NewEnvelope creates a new envelope for a message.
func NewEnvelope(msg *Message) *Envelope {
	return &Envelope{
		Message: msg,
		Hops:    0,
		Route:   make([]string, 0),
	}
}

// AddHop records a routing hop.
func (e *Envelope) AddHop(nodeID string) {
	e.Hops++
	e.Route = append(e.Route, nodeID)
}
