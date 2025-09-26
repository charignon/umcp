package config

import "time"

// Config represents the complete YAML configuration for a CLI tool
type Config struct {
	Version  string    `yaml:"version"`
	Metadata Metadata  `yaml:"metadata"`
	Settings Settings  `yaml:"settings"`
	Security Security  `yaml:"security"`
	Tools    []Tool    `yaml:"tools"`
}

// Metadata contains information about the tool
type Metadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// Settings contains global settings for the CLI tool
type Settings struct {
	Command     string            `yaml:"command"`
	WorkingDir  string            `yaml:"working_dir"`
	Timeout     time.Duration     `yaml:"timeout"`
	Environment []string          `yaml:"environment"`
	Shell       string            `yaml:"shell"`
}

// Security contains security settings
type Security struct {
	AllowedPaths     []string `yaml:"allowed_paths"`
	BlockedCommands  []string `yaml:"blocked_commands"`
	MaxOutputSize    int64    `yaml:"max_output_size"`
	RateLimit        string   `yaml:"rate_limit"`
	DisableInjectionCheck bool `yaml:"disable_injection_check"` // Allow disabling injection detection for trusted tools
}

// Tool represents a single MCP tool that wraps a CLI command
type Tool struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Command     string     `yaml:"command"`
	Arguments   []Argument `yaml:"arguments"`
	Output      Output     `yaml:"output"`
	Chain       []Chain    `yaml:"chain"`
}

// Argument represents a command-line argument
type Argument struct {
	Name         string      `yaml:"name"`
	Description  string      `yaml:"description"`
	Type         string      `yaml:"type"`
	Required     bool        `yaml:"required"`
	Flag         string      `yaml:"flag"`
	Default      interface{} `yaml:"default"`
	Min          *int        `yaml:"min"`
	Max          *int        `yaml:"max"`
	Validation   string      `yaml:"validation"`
	When         string      `yaml:"when"`
	Positional   bool        `yaml:"positional"`
	Position     int         `yaml:"position"`
}

// Output defines how to parse command output
type Output struct {
	Type    string       `yaml:"type"`
	Pattern string       `yaml:"pattern"`
	Groups  []Group      `yaml:"groups"`
	JQ      string       `yaml:"jq"`
}

// Group represents a regex capture group
type Group struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// Chain represents a command in a command chain
type Chain struct {
	Command   string   `yaml:"command"`
	Arguments []string `yaml:"arguments"`
}