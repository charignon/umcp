package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charignon/umcp/internal/config"
	"github.com/charignon/umcp/internal/parser"
	"github.com/rs/zerolog/log"
)

// Tracer interface for command tracing
type Tracer interface {
	TraceCommand(command string, args []string, workingDir string, env []string)
	TraceCommandOutput(output string, exitCode int, err error)
}

// CommandExecutor executes CLI commands with sandboxing
type CommandExecutor struct {
	builder *CommandBuilder
	sandbox *Sandbox
	tracer  Tracer
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{
		builder: NewCommandBuilder(),
		sandbox: NewSandbox(),
		tracer:  nil, // Will be set by SetTracer
	}
}

// SetTracer sets the tracer for command execution
func (e *CommandExecutor) SetTracer(tracer Tracer) {
	e.tracer = tracer
}

// Execute runs a command and returns the output
func (e *CommandExecutor) Execute(cfg *config.Config, tool *config.Tool, args map[string]interface{}) (string, error) {
	// Build the command
	cmdParts, err := e.builder.BuildCommand(cfg, tool, args)
	if err != nil {
		return "", fmt.Errorf("failed to build command: %w", err)
	}

	// Validate command against security policy
	if err := e.sandbox.ValidateCommand(cmdParts, &cfg.Security); err != nil {
		return "", fmt.Errorf("command blocked by security policy: %w", err)
	}

	// Determine working directory
	workingDir := cfg.Settings.WorkingDir
	if workingDir == "." || workingDir == "" {
		workingDir, _ = os.Getwd()
	}

	// Create command
	timeout := cfg.Settings.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	cmd.Dir = workingDir

	// Set environment variables
	cmd.Env = os.Environ()
	for _, envVar := range cfg.Settings.Environment {
		cmd.Env = append(cmd.Env, envVar)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Debug().
		Strs("command", cmdParts).
		Str("workingDir", workingDir).
		Msg("Executing command")

	// Trace command execution
	if e.tracer != nil {
		e.tracer.TraceCommand(cmdParts[0], cmdParts[1:], workingDir, cmd.Env)
	}

	// Run the command
	err = cmd.Run()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %v", timeout)
	}

	// Get output
	output := stdout.String()
	if stderr.String() != "" {
		output += "\n" + stderr.String()
	}

	// Check output size limit
	if cfg.Security.MaxOutputSize > 0 && int64(len(output)) > cfg.Security.MaxOutputSize {
		output = output[:cfg.Security.MaxOutputSize]
		output += "\n... (output truncated)"
	}

	// Trace command output
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	if e.tracer != nil {
		e.tracer.TraceCommandOutput(output, exitCode, err)
	}

	// If command failed, include error info
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return output, fmt.Errorf("command failed with exit code %d", exitErr.ExitCode())
		}
		return output, fmt.Errorf("command failed: %w", err)
	}

	// Parse output according to configuration
	parsedOutput, err := parser.ParseOutput(output, &tool.Output)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse output, returning raw")
		return output, nil
	}

	return parsedOutput, nil
}

// ExecuteChain executes a chain of commands
func (e *CommandExecutor) ExecuteChain(cfg *config.Config, chain []config.Chain, args map[string]interface{}) (string, error) {
	var outputs []string

	for i, chainCmd := range chain {
		// Build command with substitutions
		cmdParts := []string{cfg.Settings.Command}
		if chainCmd.Command != "" {
			cmdParts = append(cmdParts, chainCmd.Command)
		}

		// Process arguments with variable substitution
		for _, arg := range chainCmd.Arguments {
			processed := e.substituteVariables(arg, args)
			cmdParts = append(cmdParts, processed)
		}

		// Execute
		timeout := cfg.Settings.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
		cmd.Dir = cfg.Settings.WorkingDir
		cmd.Env = os.Environ()
		for _, envVar := range cfg.Settings.Environment {
			cmd.Env = append(cmd.Env, envVar)
		}

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		log.Debug().
			Int("step", i+1).
			Strs("command", cmdParts).
			Msg("Executing chain command")

		if err := cmd.Run(); err != nil {
			return strings.Join(outputs, "\n"), fmt.Errorf("chain step %d failed: %w", i+1, err)
		}

		output := stdout.String()
		if stderr.String() != "" {
			output += "\n" + stderr.String()
		}
		outputs = append(outputs, output)
	}

	return strings.Join(outputs, "\n"), nil
}

// substituteVariables replaces ${var} with values from args
func (e *CommandExecutor) substituteVariables(input string, args map[string]interface{}) string {
	result := input
	for key, value := range args {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// Sandbox provides security sandboxing for commands
type Sandbox struct{}

// NewSandbox creates a new sandbox
func NewSandbox() *Sandbox {
	return &Sandbox{}
}

// ValidateCommand validates a command against security policy
func (s *Sandbox) ValidateCommand(cmdParts []string, security *config.Security) error {
	if len(cmdParts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Check if command is blocked
	cmd := filepath.Base(cmdParts[0])
	if config.IsCommandBlocked(cmd, security.BlockedCommands) {
		return fmt.Errorf("command '%s' is blocked", cmd)
	}

	// Check for common injection patterns (unless explicitly disabled)
	if !security.DisableInjectionCheck {
		for _, part := range cmdParts {
			if pattern := s.findInjectionPattern(part); pattern != "" {
				return fmt.Errorf("potential command injection detected\n\nPattern found: %s\nIn content: %s\n\nThis security check prevents shell injection attacks.\nIf this is a false positive (e.g., writing documentation with code examples),\nyou can disable this check by adding to your UMCP config YAML:\n\nsecurity:\n  disable_injection_check: true\n\nOnly do this for trusted tools that handle user text content.",
					pattern, part)
			}
		}
	}

	// Validate file paths (skip if injection check is disabled - for text content tools)
	if !security.DisableInjectionCheck {
		for _, part := range cmdParts[1:] {
			if s.looksLikeFilePath(part) {
				if !config.IsPathAllowed(part, security.AllowedPaths) {
					return fmt.Errorf("path '%s' is not in allowed paths", part)
				}
			}
		}
	}

	return nil
}

// findInjectionPattern checks for common injection patterns and returns the pattern found
func (s *Sandbox) findInjectionPattern(input string) string {
	// Original aggressive patterns - keep for backward compatibility
	dangerousPatterns := []string{
		"$(", "`", "&&", "||", ";", "|", ">", "<", ">>", "<<",
		"\n", "\r", "$IFS", "${IFS}",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(input, pattern) {
			return pattern
		}
	}
	return ""
}

// hasInjectionPattern checks for common injection patterns (legacy, kept for compatibility)
func (s *Sandbox) hasInjectionPattern(input string) bool {
	return s.findInjectionPattern(input) != ""
}

// looksLikeFilePath checks if a string looks like a file path
func (s *Sandbox) looksLikeFilePath(input string) bool {
	// Simple heuristic - starts with / or ./ or contains /
	return strings.HasPrefix(input, "/") ||
		strings.HasPrefix(input, "./") ||
		strings.HasPrefix(input, "../") ||
		strings.Contains(input, "/")
}