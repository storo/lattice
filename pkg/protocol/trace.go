package protocol

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Status represents the status of a span.
type Status string

const (
	StatusUnset Status = "unset"
	StatusOK    Status = "ok"
	StatusError Status = "error"
)

// context keys
type contextKey string

const (
	spanContextKey    contextKey = "span"
	traceIDContextKey contextKey = "trace_id"
)

// Span represents a unit of work in a trace.
type Span struct {
	TraceID       string
	SpanID        string
	ParentID      string
	Name          string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Attributes    map[string]any
	Events        []Event
	Status        Status
	StatusMessage string

	mu     sync.Mutex
	tracer *Tracer
}

// Event represents a point-in-time event within a span.
type Event struct {
	Name       string
	Timestamp  time.Time
	Attributes map[string]any
}

// SetAttribute sets an attribute on the span.
func (s *Span) SetAttribute(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Attributes == nil {
		s.Attributes = make(map[string]any)
	}
	s.Attributes[key] = value
}

// AddEvent adds an event to the span.
func (s *Span) AddEvent(name string, attrs map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Events = append(s.Events, Event{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
}

// SetStatus sets the status of the span.
func (s *Span) SetStatus(status Status, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = status
	s.StatusMessage = message
}

// End marks the span as complete.
func (s *Span) End() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)

	if s.tracer != nil {
		s.tracer.recordSpan(s)
	}
}

// Tracer creates and manages spans.
type Tracer struct {
	mu    sync.Mutex
	spans []*Span
}

// NewTracer creates a new tracer.
func NewTracer() *Tracer {
	return &Tracer{
		spans: make([]*Span, 0),
	}
}

// StartSpan creates a new span and adds it to the context.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	// Get parent span if exists
	var parentID string
	var traceID string

	if parent := SpanFromContext(ctx); parent != nil {
		parentID = parent.SpanID
		traceID = parent.TraceID
	}

	// Check for trace ID in context
	if traceID == "" {
		if tid := TraceIDFromContext(ctx); tid != "" {
			traceID = tid
		} else {
			traceID = uuid.New().String()
		}
	}

	span := &Span{
		TraceID:    traceID,
		SpanID:     uuid.New().String(),
		ParentID:   parentID,
		Name:       name,
		StartTime:  time.Now(),
		Attributes: make(map[string]any),
		Events:     make([]Event, 0),
		Status:     StatusUnset,
		tracer:     t,
	}

	ctx = context.WithValue(ctx, spanContextKey, span)
	return ctx, span
}

// recordSpan stores a completed span.
func (t *Tracer) recordSpan(span *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = append(t.spans, span)
}

// Spans returns all recorded spans.
func (t *Tracer) Spans() []*Span {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]*Span(nil), t.spans...)
}

// Clear removes all recorded spans.
func (t *Tracer) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = t.spans[:0]
}

// SpanFromContext retrieves the current span from context.
func SpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanContextKey).(*Span); ok {
		return span
	}
	return nil
}

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDContextKey, traceID)
}

// TraceIDFromContext retrieves the trace ID from context.
func TraceIDFromContext(ctx context.Context) string {
	if tid, ok := ctx.Value(traceIDContextKey).(string); ok {
		return tid
	}
	return ""
}

// SpanExporter exports spans to a backend.
type SpanExporter interface {
	Export(spans []*Span) error
}

// ConsoleExporter exports spans to stdout (for debugging).
type ConsoleExporter struct{}

// Export prints spans to console.
func (e *ConsoleExporter) Export(spans []*Span) error {
	for _, span := range spans {
		println("Span:", span.Name,
			"TraceID:", span.TraceID,
			"SpanID:", span.SpanID,
			"Duration:", span.Duration.String())
	}
	return nil
}
