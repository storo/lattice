// Package builtin provides essential tools for AI agents.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TimeTool provides time-related utilities.
type TimeTool struct {
	location *time.Location
}

// TimeOption configures the time tool.
type TimeOption func(*TimeTool)

// WithTimezone sets the timezone for time operations.
func WithTimezone(loc *time.Location) TimeOption {
	return func(t *TimeTool) {
		t.location = loc
	}
}

// NewTimeTool creates a new time tool.
func NewTimeTool(opts ...TimeOption) *TimeTool {
	t := &TimeTool{
		location: time.Local,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *TimeTool) Name() string {
	return "time"
}

// Description returns the tool description.
func (t *TimeTool) Description() string {
	return "Time utilities: get current time, parse dates, calculate durations. Actions: current_time, parse_time, add_duration, unix_timestamp"
}

// Schema returns the JSON Schema for the tool parameters.
func (t *TimeTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["current_time", "parse_time", "add_duration", "unix_timestamp"],
				"description": "The action to perform"
			},
			"value": {
				"type": "string",
				"description": "Time value for parse_time or duration for add_duration (e.g., '2h30m')"
			},
			"format": {
				"type": "string",
				"description": "Time format (e.g., 'RFC3339', '2006-01-02', 'kitchen')"
			}
		},
		"required": ["action"]
	}`)
}

// timeParams are the parameters for the time tool.
type timeParams struct {
	Action string `json:"action"`
	Value  string `json:"value,omitempty"`
	Format string `json:"format,omitempty"`
}

// Execute runs the time tool.
func (t *TimeTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p timeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	now := time.Now().In(t.location)

	switch p.Action {
	case "current_time":
		format := t.parseFormat(p.Format)
		return now.Format(format), nil

	case "parse_time":
		if p.Value == "" {
			return "", fmt.Errorf("value is required for parse_time")
		}
		parsed, err := t.parseTime(p.Value)
		if err != nil {
			return "", err
		}
		format := t.parseFormat(p.Format)
		return parsed.Format(format), nil

	case "add_duration":
		if p.Value == "" {
			return "", fmt.Errorf("value is required for add_duration")
		}
		duration, err := time.ParseDuration(p.Value)
		if err != nil {
			return "", fmt.Errorf("invalid duration: %w", err)
		}
		result := now.Add(duration)
		format := t.parseFormat(p.Format)
		return result.Format(format), nil

	case "unix_timestamp":
		if p.Value != "" {
			// Convert value to unix timestamp
			parsed, err := t.parseTime(p.Value)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%d", parsed.Unix()), nil
		}
		return fmt.Sprintf("%d", now.Unix()), nil

	default:
		return "", fmt.Errorf("unknown action: %s", p.Action)
	}
}

// parseFormat converts a format name to a Go time format string.
func (t *TimeTool) parseFormat(format string) string {
	switch format {
	case "RFC3339", "":
		return time.RFC3339
	case "RFC822":
		return time.RFC822
	case "RFC850":
		return time.RFC850
	case "RFC1123":
		return time.RFC1123
	case "kitchen":
		return time.Kitchen
	case "date":
		return "2006-01-02"
	case "datetime":
		return "2006-01-02 15:04:05"
	case "time":
		return "15:04:05"
	default:
		// Use as-is if not a known format
		return format
	}
}

// parseTime tries to parse a time string using common formats.
func (t *TimeTool) parseTime(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC822,
		time.RFC1123,
		"01/02/2006",
		"02-Jan-2006",
	}

	for _, format := range formats {
		if parsed, err := time.ParseInLocation(format, value, t.location); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse time: %s", value)
}
