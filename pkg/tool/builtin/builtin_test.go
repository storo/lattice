package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTimeTool_CurrentTime(t *testing.T) {
	tool := NewTimeTool()

	params, _ := json.Marshal(map[string]string{
		"action": "current_time",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be parseable as RFC3339
	_, err = time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("expected RFC3339 format, got: %s", result)
	}
}

func TestTimeTool_CurrentTimeWithFormat(t *testing.T) {
	tool := NewTimeTool()

	params, _ := json.Marshal(map[string]string{
		"action": "current_time",
		"format": "date",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be in date format
	_, err = time.Parse("2006-01-02", result)
	if err != nil {
		t.Errorf("expected date format, got: %s", result)
	}
}

func TestTimeTool_ParseTime(t *testing.T) {
	tool := NewTimeTool()

	params, _ := json.Marshal(map[string]string{
		"action": "parse_time",
		"value":  "2024-01-15",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(result, "2024-01-15") {
		t.Errorf("expected parsed time to start with 2024-01-15, got: %s", result)
	}
}

func TestTimeTool_AddDuration(t *testing.T) {
	tool := NewTimeTool()

	before := time.Now()
	params, _ := json.Marshal(map[string]string{
		"action": "add_duration",
		"value":  "1h",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	expected := before.Add(1 * time.Hour)
	diff := parsed.Sub(expected)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expected ~1 hour from now, got: %s", result)
	}
}

func TestTimeTool_UnixTimestamp(t *testing.T) {
	tool := NewTimeTool()

	before := time.Now().Unix()
	params, _ := json.Marshal(map[string]string{
		"action": "unix_timestamp",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now().Unix()

	var ts int64
	json.Unmarshal([]byte(result), &ts)

	if ts < before || ts > after {
		t.Errorf("expected timestamp between %d and %d, got: %s", before, after, result)
	}
}

func TestTimeTool_Name(t *testing.T) {
	tool := NewTimeTool()
	if tool.Name() != "time" {
		t.Errorf("expected 'time', got '%s'", tool.Name())
	}
}

func TestFSTool_ListDir(t *testing.T) {
	tool := NewFSTool()

	params, _ := json.Marshal(map[string]string{
		"action": "list_dir",
		"path":   ".",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be valid JSON array
	var items []map[string]any
	if err := json.Unmarshal([]byte(result), &items); err != nil {
		t.Errorf("expected JSON array, got: %s", result)
	}
}

func TestFSTool_ReadWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"

	tool := NewFSTool(WithAllowedPaths(tmpDir))

	// Write file
	writeParams, _ := json.Marshal(map[string]string{
		"action":  "write_file",
		"path":    testFile,
		"content": content,
	})

	_, err := tool.Execute(context.Background(), writeParams)
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Read file
	readParams, _ := json.Marshal(map[string]string{
		"action": "read_file",
		"path":   testFile,
	})

	result, err := tool.Execute(context.Background(), readParams)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	if result != content {
		t.Errorf("expected '%s', got '%s'", content, result)
	}
}

func TestFSTool_FileInfo(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "info.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	tool := NewFSTool(WithAllowedPaths(tmpDir))

	params, _ := json.Marshal(map[string]string{
		"action": "file_info",
		"path":   testFile,
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var info map[string]any
	if err := json.Unmarshal([]byte(result), &info); err != nil {
		t.Errorf("expected JSON object, got: %s", result)
	}

	if info["name"] != "info.txt" {
		t.Errorf("expected name 'info.txt', got: %v", info["name"])
	}
}

func TestFSTool_PathRestriction(t *testing.T) {
	tool := NewFSTool(WithAllowedPaths("/tmp/allowed"))

	params, _ := json.Marshal(map[string]string{
		"action": "read_file",
		"path":   "/etc/passwd",
	})

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Error("expected error for restricted path")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}
}

func TestFSTool_ReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewFSTool(WithAllowedPaths(tmpDir), WithReadOnly())

	params, _ := json.Marshal(map[string]string{
		"action":  "write_file",
		"path":    filepath.Join(tmpDir, "test.txt"),
		"content": "test",
	})

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Error("expected error for write in read-only mode")
	}
}

func TestShellTool_AllowedCommand(t *testing.T) {
	tool := NewShellTool(WithAllowedCommands("echo", "date"))

	params, _ := json.Marshal(map[string]string{
		"command": "echo hello",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output struct {
		Stdout string `json:"stdout"`
	}
	json.Unmarshal([]byte(result), &output)

	if !strings.Contains(output.Stdout, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", output.Stdout)
	}
}

func TestShellTool_DisallowedCommand(t *testing.T) {
	tool := NewShellTool(WithAllowedCommands("echo"))

	params, _ := json.Marshal(map[string]string{
		"command": "rm -rf /",
	})

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Error("expected error for disallowed command")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %v", err)
	}
}

func TestShellTool_Timeout(t *testing.T) {
	tool := NewShellTool(WithShellTimeout(100 * time.Millisecond))

	params, _ := json.Marshal(map[string]string{
		"command": "sleep 10",
	})

	start := time.Now()
	result, _ := tool.Execute(context.Background(), params)
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Error("command should have timed out")
	}

	var output struct {
		Error string `json:"error"`
	}
	json.Unmarshal([]byte(result), &output)
	// The error should indicate timeout or signal
}

func TestHTTPTool_Name(t *testing.T) {
	tool := NewHTTPTool()
	if tool.Name() != "http" {
		t.Errorf("expected 'http', got '%s'", tool.Name())
	}
}

func TestHTTPTool_DomainRestriction(t *testing.T) {
	tool := NewHTTPTool(WithAllowedDomains("example.com"))

	params, _ := json.Marshal(map[string]string{
		"method": "GET",
		"url":    "https://evil.com/api",
	})

	_, err := tool.Execute(context.Background(), params)
	if err == nil {
		t.Error("expected error for disallowed domain")
	}
}

func TestDefaultTools(t *testing.T) {
	tools := DefaultTools()

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}

	if !names["time"] {
		t.Error("expected time tool")
	}
	if !names["http"] {
		t.Error("expected http tool")
	}
}

func TestDeveloperTools(t *testing.T) {
	tools := DeveloperTools()

	if len(tools) != 4 {
		t.Errorf("expected 4 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}

	for _, expected := range []string{"time", "http", "fs", "shell"} {
		if !names[expected] {
			t.Errorf("expected %s tool", expected)
		}
	}
}
