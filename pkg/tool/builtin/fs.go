package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FSTool provides file system operations.
type FSTool struct {
	allowedPaths []string
	maxFileSize  int64
	readOnly     bool
}

// FSOption configures the FS tool.
type FSOption func(*FSTool)

// WithAllowedPaths restricts operations to specific directories.
func WithAllowedPaths(paths ...string) FSOption {
	return func(t *FSTool) {
		t.allowedPaths = paths
	}
}

// WithMaxFileSize limits the maximum file size for read/write.
func WithMaxFileSize(size int64) FSOption {
	return func(t *FSTool) {
		t.maxFileSize = size
	}
}

// WithReadOnly makes the tool read-only.
func WithReadOnly() FSOption {
	return func(t *FSTool) {
		t.readOnly = true
	}
}

// NewFSTool creates a new file system tool.
func NewFSTool(opts ...FSOption) *FSTool {
	t := &FSTool{
		maxFileSize: 10 * 1024 * 1024, // 10MB
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *FSTool) Name() string {
	return "fs"
}

// Description returns the tool description.
func (t *FSTool) Description() string {
	desc := "File system operations: read_file, list_dir, file_info"
	if !t.readOnly {
		desc += ", write_file, delete_file, mkdir"
	}
	return desc
}

// Schema returns the JSON Schema for the tool parameters.
func (t *FSTool) Schema() json.RawMessage {
	actions := []string{"read_file", "list_dir", "file_info"}
	if !t.readOnly {
		actions = append(actions, "write_file", "delete_file", "mkdir")
	}

	actionsJSON, _ := json.Marshal(actions)

	return json.RawMessage(fmt.Sprintf(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": %s,
				"description": "The file operation to perform"
			},
			"path": {
				"type": "string",
				"description": "File or directory path"
			},
			"content": {
				"type": "string",
				"description": "Content for write_file"
			}
		},
		"required": ["action", "path"]
	}`, string(actionsJSON)))
}

// fsParams are the parameters for the FS tool.
type fsParams struct {
	Action  string `json:"action"`
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
}

// Execute runs the FS tool.
func (t *FSTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p fsParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate path
	absPath, err := filepath.Abs(p.Path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	if !t.isPathAllowed(absPath) {
		return "", fmt.Errorf("path not allowed: %s", p.Path)
	}

	switch p.Action {
	case "read_file":
		return t.readFile(absPath)

	case "write_file":
		if t.readOnly {
			return "", fmt.Errorf("write operations not allowed")
		}
		return t.writeFile(absPath, p.Content)

	case "list_dir":
		return t.listDir(absPath)

	case "file_info":
		return t.fileInfo(absPath)

	case "delete_file":
		if t.readOnly {
			return "", fmt.Errorf("delete operations not allowed")
		}
		return t.deleteFile(absPath)

	case "mkdir":
		if t.readOnly {
			return "", fmt.Errorf("mkdir operations not allowed")
		}
		return t.mkdir(absPath)

	default:
		return "", fmt.Errorf("unknown action: %s", p.Action)
	}
}

// isPathAllowed checks if a path is within allowed directories.
func (t *FSTool) isPathAllowed(absPath string) bool {
	if len(t.allowedPaths) == 0 {
		return true
	}

	for _, allowed := range t.allowedPaths {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, allowedAbs) {
			return true
		}
	}
	return false
}

// readFile reads and returns the contents of a file.
func (t *FSTool) readFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	if info.Size() > t.maxFileSize {
		return "", fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), t.maxFileSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}

// writeFile writes content to a file.
func (t *FSTool) writeFile(path, content string) (string, error) {
	if int64(len(content)) > t.maxFileSize {
		return "", fmt.Errorf("content too large: %d bytes (max %d)", len(content), t.maxFileSize)
	}

	// Create parent directories if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

// listDir lists the contents of a directory.
func (t *FSTool) listDir(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var items []map[string]any
	for _, entry := range entries {
		info, _ := entry.Info()
		item := map[string]any{
			"name":  entry.Name(),
			"isDir": entry.IsDir(),
		}
		if info != nil {
			item["size"] = info.Size()
			item["mode"] = info.Mode().String()
			item["modTime"] = info.ModTime().Format("2006-01-02 15:04:05")
		}
		items = append(items, item)
	}

	output, _ := json.MarshalIndent(items, "", "  ")
	return string(output), nil
}

// fileInfo returns information about a file or directory.
func (t *FSTool) fileInfo(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	result := map[string]any{
		"name":    info.Name(),
		"size":    info.Size(),
		"mode":    info.Mode().String(),
		"modTime": info.ModTime().Format("2006-01-02 15:04:05"),
		"isDir":   info.IsDir(),
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}

// deleteFile deletes a file.
func (t *FSTool) deleteFile(path string) (string, error) {
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}
	return fmt.Sprintf("Successfully deleted %s", path), nil
}

// mkdir creates a directory.
func (t *FSTool) mkdir(path string) (string, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	return fmt.Sprintf("Successfully created directory %s", path), nil
}
