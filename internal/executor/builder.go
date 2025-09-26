package executor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charignon/umcp/internal/config"
)

// CommandBuilder builds CLI commands from MCP arguments
type CommandBuilder struct{}

// NewCommandBuilder creates a new command builder
func NewCommandBuilder() *CommandBuilder {
	return &CommandBuilder{}
}

// BuildCommand constructs a command line from tool configuration and arguments
func (b *CommandBuilder) BuildCommand(cfg *config.Config, tool *config.Tool, args map[string]interface{}) ([]string, error) {
	cmd := []string{}

	// Start with the base command
	if cfg.Settings.Command != "" {
		cmd = append(cmd, cfg.Settings.Command)
	}

	// Add subcommand if specified
	if tool.Command != "" {
		cmd = append(cmd, tool.Command)
	}

	// Process positional arguments first
	positionalArgs := b.extractPositionalArgs(tool.Arguments, args)
	for _, arg := range positionalArgs {
		value, exists := args[arg.Name]
		if !exists {
			if arg.Default != nil {
				value = arg.Default
			} else if arg.Required {
				return nil, fmt.Errorf("required argument %s not provided", arg.Name)
			} else {
				continue
			}
		}

		strVal, err := b.formatValue(arg.Type, value)
		if err != nil {
			return nil, fmt.Errorf("failed to format %s: %w", arg.Name, err)
		}
		cmd = append(cmd, strVal)
	}

	// Process flag arguments
	for _, arg := range tool.Arguments {
		if arg.Positional {
			continue
		}

		value, exists := args[arg.Name]
		if !exists {
			if arg.Default != nil {
				value = arg.Default
			} else if arg.Required {
				return nil, fmt.Errorf("required argument %s not provided", arg.Name)
			} else {
				continue
			}
		}

		// Handle conditional arguments
		if arg.When != "" && !b.evaluateCondition(arg.When, args) {
			continue
		}

		// Build the flag
		flagParts, err := b.buildFlag(arg, value)
		if err != nil {
			return nil, fmt.Errorf("failed to build flag for %s: %w", arg.Name, err)
		}
		cmd = append(cmd, flagParts...)
	}

	return cmd, nil
}

// extractPositionalArgs extracts and sorts positional arguments
func (b *CommandBuilder) extractPositionalArgs(arguments []config.Argument, args map[string]interface{}) []config.Argument {
	positional := []config.Argument{}
	for _, arg := range arguments {
		if arg.Positional {
			positional = append(positional, arg)
		}
	}

	// Sort by position
	sort.Slice(positional, func(i, j int) bool {
		return positional[i].Position < positional[j].Position
	})

	return positional
}

// buildFlag builds command-line flag(s) from an argument
func (b *CommandBuilder) buildFlag(arg config.Argument, value interface{}) ([]string, error) {
	switch arg.Type {
	case "boolean":
		boolVal, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("expected boolean, got %T", value)
		}
		if boolVal && arg.Flag != "" {
			return []string{arg.Flag}, nil
		}
		return []string{}, nil

	case "array":
		arr, ok := value.([]interface{})
		if !ok {
			// Try to convert single value to array
			arr = []interface{}{value}
		}

		result := []string{}
		for _, item := range arr {
			strVal, err := b.formatValue("string", item)
			if err != nil {
				return nil, err
			}
			if strings.Contains(arg.Flag, "=") {
				result = append(result, fmt.Sprintf("%s%s", arg.Flag, strVal))
			} else {
				result = append(result, arg.Flag, strVal)
			}
		}
		return result, nil

	default:
		strVal, err := b.formatValue(arg.Type, value)
		if err != nil {
			return nil, err
		}

		if arg.Flag == "" {
			return []string{strVal}, nil
		}

		if strings.Contains(arg.Flag, "=") {
			return []string{fmt.Sprintf("%s%s", arg.Flag, strVal)}, nil
		}
		return []string{arg.Flag, strVal}, nil
	}
}

// formatValue formats a value according to its type
func (b *CommandBuilder) formatValue(argType string, value interface{}) (string, error) {
	switch argType {
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Sprintf("%v", value), nil
		}
		return str, nil

	case "integer":
		switch v := value.(type) {
		case float64:
			return strconv.Itoa(int(v)), nil
		case int:
			return strconv.Itoa(v), nil
		case string:
			// Parse string to int
			i, err := strconv.Atoi(v)
			if err != nil {
				return "", fmt.Errorf("invalid integer: %s", v)
			}
			return strconv.Itoa(i), nil
		default:
			return "", fmt.Errorf("expected integer, got %T", value)
		}

	case "float":
		switch v := value.(type) {
		case float64:
			return fmt.Sprintf("%f", v), nil
		case int:
			return fmt.Sprintf("%f", float64(v)), nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return "", fmt.Errorf("invalid float: %s", v)
			}
			return fmt.Sprintf("%f", f), nil
		default:
			return "", fmt.Errorf("expected float, got %T", value)
		}

	case "object":
		data, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("failed to marshal object: %w", err)
		}
		return string(data), nil

	default:
		return fmt.Sprintf("%v", value), nil
	}
}

// evaluateCondition evaluates a simple condition expression
func (b *CommandBuilder) evaluateCondition(condition string, args map[string]interface{}) bool {
	// Simple implementation - can be extended
	// Format: "${varname} == value"
	parts := strings.Split(condition, " ")
	if len(parts) != 3 {
		return false
	}

	varName := strings.TrimPrefix(strings.TrimSuffix(parts[0], "}"), "${")
	operator := parts[1]
	expectedValue := strings.Trim(parts[2], "\"")

	actualValue, exists := args[varName]
	if !exists {
		return false
	}

	switch operator {
	case "==":
		return fmt.Sprintf("%v", actualValue) == expectedValue
	case "!=":
		return fmt.Sprintf("%v", actualValue) != expectedValue
	default:
		return false
	}
}