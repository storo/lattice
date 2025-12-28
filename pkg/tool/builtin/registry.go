package builtin

import (
	"github.com/storo/lattice/pkg/core"
)

// DefaultTools returns a safe set of built-in tools.
// These tools are suitable for use in untrusted environments.
// Includes: time, http (read-only operations)
func DefaultTools() []core.Tool {
	return []core.Tool{
		NewTimeTool(),
		NewHTTPTool(),
	}
}

// SafeTools returns tools with security restrictions.
// Includes file system access limited to specified paths.
func SafeTools(allowedPaths ...string) []core.Tool {
	return []core.Tool{
		NewTimeTool(),
		NewHTTPTool(),
		NewFSTool(WithAllowedPaths(allowedPaths...), WithReadOnly()),
	}
}

// AllTools returns all built-in tools.
// WARNING: This includes shell access - only use in trusted environments!
func AllTools(opts AllToolsOptions) []core.Tool {
	tools := []core.Tool{
		NewTimeTool(),
		NewHTTPTool(),
		NewFSTool(),
	}

	// Add shell tool only if commands are specified
	if len(opts.ShellCommands) > 0 {
		tools = append(tools, NewShellTool(WithAllowedCommands(opts.ShellCommands...)))
	} else if opts.AllowAllShellCommands {
		tools = append(tools, NewShellTool())
	}

	return tools
}

// AllToolsOptions configures which tools to include.
type AllToolsOptions struct {
	// ShellCommands restricts shell to these commands.
	ShellCommands []string

	// AllowAllShellCommands enables unrestricted shell access.
	// WARNING: This is dangerous in untrusted environments!
	AllowAllShellCommands bool

	// FSAllowedPaths restricts file system access to these paths.
	FSAllowedPaths []string

	// FSReadOnly makes file system read-only.
	FSReadOnly bool

	// HTTPAllowedDomains restricts HTTP to these domains.
	HTTPAllowedDomains []string
}

// CustomTools creates tools with custom configuration.
func CustomTools(opts AllToolsOptions) []core.Tool {
	var tools []core.Tool

	// Time tool (always safe)
	tools = append(tools, NewTimeTool())

	// HTTP tool
	var httpOpts []HTTPOption
	if len(opts.HTTPAllowedDomains) > 0 {
		httpOpts = append(httpOpts, WithAllowedDomains(opts.HTTPAllowedDomains...))
	}
	tools = append(tools, NewHTTPTool(httpOpts...))

	// FS tool
	var fsOpts []FSOption
	if len(opts.FSAllowedPaths) > 0 {
		fsOpts = append(fsOpts, WithAllowedPaths(opts.FSAllowedPaths...))
	}
	if opts.FSReadOnly {
		fsOpts = append(fsOpts, WithReadOnly())
	}
	tools = append(tools, NewFSTool(fsOpts...))

	// Shell tool (only if configured)
	if len(opts.ShellCommands) > 0 {
		tools = append(tools, NewShellTool(WithAllowedCommands(opts.ShellCommands...)))
	} else if opts.AllowAllShellCommands {
		tools = append(tools, NewShellTool())
	}

	return tools
}

// DeveloperTools returns tools commonly needed for development tasks.
func DeveloperTools() []core.Tool {
	return []core.Tool{
		NewTimeTool(),
		NewHTTPTool(),
		NewFSTool(),
		NewShellTool(WithAllowedCommands(
			"ls", "cat", "head", "tail", "grep", "find", "wc",
			"git", "go", "npm", "node", "python", "pip",
			"make", "docker", "kubectl",
		)),
	}
}

// MinimalTools returns the absolute minimum set of tools.
func MinimalTools() []core.Tool {
	return []core.Tool{
		NewTimeTool(),
	}
}
