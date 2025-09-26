package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads and validates a YAML configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply defaults and validate
	if err := cfg.applyDefaults(); err != nil {
		return nil, fmt.Errorf("failed to apply defaults: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// applyDefaults sets default values for optional fields
func (c *Config) applyDefaults() error {
	if c.Version == "" {
		c.Version = "1.0"
	}

	if c.Settings.WorkingDir == "" {
		c.Settings.WorkingDir = "."
	}

	if c.Settings.Timeout == 0 {
		c.Settings.Timeout = 30 * time.Second
	}

	if c.Security.MaxOutputSize == 0 {
		c.Security.MaxOutputSize = 10 * 1024 * 1024 // 10MB default
	}

	// Apply defaults to tools
	for i := range c.Tools {
		tool := &c.Tools[i]
		if tool.Output.Type == "" {
			tool.Output.Type = "raw"
		}

		// Apply argument defaults
		for j := range tool.Arguments {
			arg := &tool.Arguments[j]
			if arg.Type == "" {
				arg.Type = "string"
			}
		}
	}

	return nil
}

// validate checks that the configuration is valid
func (c *Config) validate() error {
	if c.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	if c.Settings.Command == "" {
		return fmt.Errorf("settings.command is required")
	}

	if len(c.Tools) == 0 {
		return fmt.Errorf("at least one tool must be defined")
	}

	// Validate each tool
	for _, tool := range c.Tools {
		if tool.Name == "" {
			return fmt.Errorf("tool name is required")
		}

		if tool.Description == "" {
			return fmt.Errorf("tool %s: description is required", tool.Name)
		}

		// Validate output type
		validOutputTypes := map[string]bool{
			"raw": true, "json": true, "lines": true,
			"regex": true, "csv": true, "xml": true,
		}
		if !validOutputTypes[tool.Output.Type] {
			return fmt.Errorf("tool %s: invalid output type %s", tool.Name, tool.Output.Type)
		}

		// If regex output, pattern is required
		if tool.Output.Type == "regex" && tool.Output.Pattern == "" {
			return fmt.Errorf("tool %s: pattern is required for regex output", tool.Name)
		}

		// Validate arguments
		for _, arg := range tool.Arguments {
			if arg.Name == "" {
				return fmt.Errorf("tool %s: argument name is required", tool.Name)
			}

			// Validate argument type
			validArgTypes := map[string]bool{
				"string": true, "boolean": true, "integer": true,
				"array": true, "object": true, "float": true,
			}
			if !validArgTypes[arg.Type] {
				return fmt.Errorf("tool %s, argument %s: invalid type %s", tool.Name, arg.Name, arg.Type)
			}

			// Check that required args have no default
			if arg.Required && arg.Default != nil {
				return fmt.Errorf("tool %s, argument %s: required arguments cannot have defaults", tool.Name, arg.Name)
			}
		}
	}

	return nil
}