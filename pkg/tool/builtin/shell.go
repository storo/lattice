package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ShellTool executes shell commands.
type ShellTool struct {
	allowedCommands []string
	timeout         time.Duration
	workDir         string
	shell           string
}

// ShellOption configures the shell tool.
type ShellOption func(*ShellTool)

// WithAllowedCommands restricts to specific commands.
// If empty, all commands are allowed (dangerous!).
func WithAllowedCommands(commands ...string) ShellOption {
	return func(t *ShellTool) {
		t.allowedCommands = commands
	}
}

// WithShellTimeout sets the command timeout.
func WithShellTimeout(d time.Duration) ShellOption {
	return func(t *ShellTool) {
		t.timeout = d
	}
}

// WithWorkDir sets the working directory.
func WithWorkDir(dir string) ShellOption {
	return func(t *ShellTool) {
		t.workDir = dir
	}
}

// WithShell sets the shell to use (e.g., "/bin/bash", "/bin/sh").
func WithShell(shell string) ShellOption {
	return func(t *ShellTool) {
		t.shell = shell
	}
}

// NewShellTool creates a new shell tool.
// WARNING: Without allowedCommands, this tool can execute any command!
func NewShellTool(opts ...ShellOption) *ShellTool {
	t := &ShellTool{
		timeout: 30 * time.Second,
		shell:   "/bin/sh",
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *ShellTool) Name() string {
	return "shell"
}

// Description returns the tool description.
func (t *ShellTool) Description() string {
	if len(t.allowedCommands) > 0 {
		return fmt.Sprintf("Execute shell commands. Allowed: %s", strings.Join(t.allowedCommands, ", "))
	}
	return "Execute shell commands"
}

// Schema returns the JSON Schema for the tool parameters.
func (t *ShellTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The shell command to execute"
			},
			"args": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Command arguments"
			},
			"stdin": {
				"type": "string",
				"description": "Input to pass to stdin"
			}
		},
		"required": ["command"]
	}`)
}

// shellParams are the parameters for the shell tool.
type shellParams struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Stdin   string   `json:"stdin,omitempty"`
}

// Execute runs the shell tool.
func (t *ShellTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p shellParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate command
	if !t.isCommandAllowed(p.Command) {
		return "", fmt.Errorf("command not allowed: %s", p.Command)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Build command
	var cmd *exec.Cmd
	if len(p.Args) > 0 {
		cmd = exec.CommandContext(ctx, p.Command, p.Args...)
	} else {
		// Execute through shell for complex commands
		cmd = exec.CommandContext(ctx, t.shell, "-c", p.Command)
	}

	if t.workDir != "" {
		cmd.Dir = t.workDir
	}

	// Set stdin if provided
	if p.Stdin != "" {
		cmd.Stdin = strings.NewReader(p.Stdin)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Build result
	result := struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exitCode"`
		Error    string `json:"error,omitempty"`
	}{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = err.Error()
			result.ExitCode = -1
		}
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return string(output), nil
}

// isCommandAllowed checks if a command is in the allowed list.
func (t *ShellTool) isCommandAllowed(cmd string) bool {
	if len(t.allowedCommands) == 0 {
		return true // No restrictions
	}

	// Extract base command (first word)
	baseCmd := strings.Fields(cmd)[0]

	for _, allowed := range t.allowedCommands {
		if baseCmd == allowed || cmd == allowed {
			return true
		}
	}
	return false
}
