package executor

import (
	"testing"

	"github.com/charignon/umcp/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCommand(t *testing.T) {
	builder := NewCommandBuilder()

	tests := []struct {
		name     string
		cfg      *config.Config
		tool     *config.Tool
		args     map[string]interface{}
		expected []string
	}{
		{
			name: "simple command with flags",
			cfg: &config.Config{
				Settings: config.Settings{
					Command: "git",
				},
			},
			tool: &config.Tool{
				Command: "status",
				Arguments: []config.Argument{
					{
						Name: "short",
						Type: "boolean",
						Flag: "--short",
					},
					{
						Name: "branch",
						Type: "boolean",
						Flag: "--branch",
					},
				},
			},
			args: map[string]interface{}{
				"short":  true,
				"branch": true,
			},
			expected: []string{"git", "status", "--short", "--branch"},
		},
		{
			name: "command with positional arguments",
			cfg: &config.Config{
				Settings: config.Settings{
					Command: "docker",
				},
			},
			tool: &config.Tool{
				Command: "run",
				Arguments: []config.Argument{
					{
						Name:       "image",
						Type:       "string",
						Positional: true,
						Position:   0,
					},
					{
						Name:       "command",
						Type:       "string",
						Positional: true,
						Position:   1,
					},
					{
						Name: "detach",
						Type: "boolean",
						Flag: "-d",
					},
				},
			},
			args: map[string]interface{}{
				"image":   "ubuntu",
				"command": "bash",
				"detach":  true,
			},
			expected: []string{"docker", "run", "ubuntu", "bash", "-d"},
		},
		{
			name: "command with array arguments",
			cfg: &config.Config{
				Settings: config.Settings{
					Command: "docker",
				},
			},
			tool: &config.Tool{
				Command: "run",
				Arguments: []config.Argument{
					{
						Name: "env",
						Type: "array",
						Flag: "--env",
					},
				},
			},
			args: map[string]interface{}{
				"env": []interface{}{"FOO=bar", "BAZ=qux"},
			},
			expected: []string{"docker", "run", "--env", "FOO=bar", "--env", "BAZ=qux"},
		},
		{
			name: "command with default values",
			cfg: &config.Config{
				Settings: config.Settings{
					Command: "git",
				},
			},
			tool: &config.Tool{
				Command: "log",
				Arguments: []config.Argument{
					{
						Name:    "limit",
						Type:    "integer",
						Flag:    "-n",
						Default: 10,
					},
				},
			},
			args:     map[string]interface{}{},
			expected: []string{"git", "log", "-n", "10"},
		},
		{
			name: "command with conditional arguments",
			cfg: &config.Config{
				Settings: config.Settings{
					Command: "test",
				},
			},
			tool: &config.Tool{
				Arguments: []config.Argument{
					{
						Name: "debug",
						Type: "boolean",
						Flag: "--debug",
					},
					{
						Name: "verbose",
						Type: "boolean",
						Flag: "-v",
						When: "${debug} == true",
					},
				},
			},
			args: map[string]interface{}{
				"debug":   true,
				"verbose": true,
			},
			expected: []string{"test", "--debug", "-v"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.BuildCommand(tt.cfg, tt.tool, tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCommandErrors(t *testing.T) {
	builder := NewCommandBuilder()

	tests := []struct {
		name        string
		cfg         *config.Config
		tool        *config.Tool
		args        map[string]interface{}
		expectError string
	}{
		{
			name: "missing required argument",
			cfg: &config.Config{
				Settings: config.Settings{
					Command: "git",
				},
			},
			tool: &config.Tool{
				Command: "commit",
				Arguments: []config.Argument{
					{
						Name:     "message",
						Type:     "string",
						Required: true,
						Flag:     "-m",
					},
				},
			},
			args:        map[string]interface{}{},
			expectError: "required argument message not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := builder.BuildCommand(tt.cfg, tt.tool, tt.args)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestFormatValue(t *testing.T) {
	builder := NewCommandBuilder()

	tests := []struct {
		name     string
		argType  string
		value    interface{}
		expected string
	}{
		{
			name:     "string value",
			argType:  "string",
			value:    "hello",
			expected: "hello",
		},
		{
			name:     "integer from float64",
			argType:  "integer",
			value:    float64(42),
			expected: "42",
		},
		{
			name:     "integer from string",
			argType:  "integer",
			value:    "42",
			expected: "42",
		},
		{
			name:     "float value",
			argType:  "float",
			value:    3.14,
			expected: "3.140000",
		},
		{
			name:     "object as JSON",
			argType:  "object",
			value:    map[string]interface{}{"key": "value"},
			expected: `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.formatValue(tt.argType, tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateCondition(t *testing.T) {
	builder := NewCommandBuilder()

	tests := []struct {
		name      string
		condition string
		args      map[string]interface{}
		expected  bool
	}{
		{
			name:      "equals true",
			condition: "${debug} == true",
			args:      map[string]interface{}{"debug": true},
			expected:  true,
		},
		{
			name:      "equals false",
			condition: "${debug} == false",
			args:      map[string]interface{}{"debug": true},
			expected:  false,
		},
		{
			name:      "not equals",
			condition: "${mode} != production",
			args:      map[string]interface{}{"mode": "development"},
			expected:  true,
		},
		{
			name:      "variable not exists",
			condition: "${missing} == true",
			args:      map[string]interface{}{},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.evaluateCondition(tt.condition, tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}