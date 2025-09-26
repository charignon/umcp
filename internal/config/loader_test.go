package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
version: "1.0"
metadata:
  name: test
  description: Test tool
  version: 1.0.0

settings:
  command: echo
  working_dir: /tmp
  timeout: 60s
  environment:
    - TEST_VAR=value

security:
  blocked_commands:
    - rm
    - dd
  max_output_size: 1048576

tools:
  - name: hello
    description: Say hello
    command: hello
    arguments:
      - name: name
        description: Name to greet
        type: string
        required: true
        flag: "--name"
      - name: verbose
        description: Be verbose
        type: boolean
        flag: "-v"
    output:
      type: raw
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify metadata
	assert.Equal(t, "test", cfg.Metadata.Name)
	assert.Equal(t, "Test tool", cfg.Metadata.Description)
	assert.Equal(t, "1.0.0", cfg.Metadata.Version)

	// Verify settings
	assert.Equal(t, "echo", cfg.Settings.Command)
	assert.Equal(t, "/tmp", cfg.Settings.WorkingDir)
	assert.Equal(t, 60*time.Second, cfg.Settings.Timeout)
	assert.Contains(t, cfg.Settings.Environment, "TEST_VAR=value")

	// Verify security
	assert.Contains(t, cfg.Security.BlockedCommands, "rm")
	assert.Contains(t, cfg.Security.BlockedCommands, "dd")
	assert.Equal(t, int64(1048576), cfg.Security.MaxOutputSize)

	// Verify tools
	require.Len(t, cfg.Tools, 1)
	tool := cfg.Tools[0]
	assert.Equal(t, "hello", tool.Name)
	assert.Equal(t, "Say hello", tool.Description)
	assert.Equal(t, "hello", tool.Command)

	// Verify arguments
	require.Len(t, tool.Arguments, 2)
	arg1 := tool.Arguments[0]
	assert.Equal(t, "name", arg1.Name)
	assert.Equal(t, "string", arg1.Type)
	assert.True(t, arg1.Required)
	assert.Equal(t, "--name", arg1.Flag)

	arg2 := tool.Arguments[1]
	assert.Equal(t, "verbose", arg2.Name)
	assert.Equal(t, "boolean", arg2.Type)
	assert.False(t, arg2.Required)
	assert.Equal(t, "-v", arg2.Flag)
}

func TestLoadConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError string
	}{
		{
			name: "missing metadata name",
			config: `
version: "1.0"
metadata:
  description: Test
settings:
  command: test
tools:
  - name: test
    description: Test tool
`,
			expectError: "metadata.name is required",
		},
		{
			name: "missing command",
			config: `
version: "1.0"
metadata:
  name: test
tools:
  - name: test
    description: Test tool
`,
			expectError: "settings.command is required",
		},
		{
			name: "no tools",
			config: `
version: "1.0"
metadata:
  name: test
settings:
  command: test
`,
			expectError: "at least one tool must be defined",
		},
		{
			name: "invalid output type",
			config: `
version: "1.0"
metadata:
  name: test
settings:
  command: test
tools:
  - name: test
    description: Test tool
    output:
      type: invalid
`,
			expectError: "invalid output type",
		},
		{
			name: "regex without pattern",
			config: `
version: "1.0"
metadata:
  name: test
settings:
  command: test
tools:
  - name: test
    description: Test tool
    output:
      type: regex
`,
			expectError: "pattern is required for regex output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.yaml")

			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			_, err = LoadConfig(configPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{
		Metadata: Metadata{
			Name: "test",
		},
		Settings: Settings{
			Command: "test",
		},
		Tools: []Tool{
			{
				Name:        "test",
				Description: "Test",
				Arguments: []Argument{
					{Name: "arg1"},
				},
			},
		},
	}

	err := cfg.applyDefaults()
	require.NoError(t, err)

	// Check defaults were applied
	assert.Equal(t, "1.0", cfg.Version)
	assert.Equal(t, ".", cfg.Settings.WorkingDir)
	assert.Equal(t, 30*time.Second, cfg.Settings.Timeout)
	assert.Equal(t, int64(10*1024*1024), cfg.Security.MaxOutputSize)
	assert.Equal(t, "raw", cfg.Tools[0].Output.Type)
	assert.Equal(t, "string", cfg.Tools[0].Arguments[0].Type)
}