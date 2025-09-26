package parser

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	"github.com/charignon/umcp/internal/config"
)

// ParseOutput parses command output according to the output configuration
func ParseOutput(output string, outputCfg *config.Output) (string, error) {
	switch outputCfg.Type {
	case "json":
		return parseJSON(output, outputCfg.JQ)
	case "lines":
		return parseLines(output)
	case "regex":
		return parseRegex(output, outputCfg.Pattern, outputCfg.Groups)
	case "csv":
		return parseCSV(output)
	case "xml":
		return parseXML(output)
	case "raw":
		fallthrough
	default:
		return output, nil
	}
}

// parseJSON parses JSON output and optionally applies JQ filter
func parseJSON(output string, jqFilter string) (string, error) {
	// First validate that it's valid JSON
	var data interface{}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// TODO: Implement JQ filtering if needed
	// For now, just pretty-print the JSON
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return output, nil
	}

	return string(pretty), nil
}

// parseLines splits output into lines and returns as JSON array
func parseLines(output string) (string, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Filter out empty lines
	filtered := []string{}
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return output, err
	}

	return string(data), nil
}

// parseRegex applies regex pattern and extracts groups
func parseRegex(output string, pattern string, groups []config.Group) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("regex pattern is required")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	var results []map[string]interface{}

	// Find all matches
	matches := re.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		result := make(map[string]interface{})

		// If we have named groups, use them
		if len(groups) > 0 {
			for i, group := range groups {
				if i+1 < len(match) {
					result[group.Name] = convertType(match[i+1], group.Type)
				}
			}
		} else {
			// Otherwise, use numbered groups
			for i, submatch := range match {
				if i > 0 { // Skip the full match at index 0
					result[fmt.Sprintf("group%d", i)] = submatch
				}
			}
		}

		results = append(results, result)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return output, err
	}

	return string(data), nil
}

// parseCSV parses CSV output
func parseCSV(output string) (string, error) {
	reader := csv.NewReader(strings.NewReader(output))
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Convert to JSON array of objects
	if len(records) == 0 {
		return "[]", nil
	}

	// Use first row as headers
	headers := records[0]
	var results []map[string]string

	for i := 1; i < len(records); i++ {
		row := records[i]
		obj := make(map[string]string)

		for j, header := range headers {
			if j < len(row) {
				obj[header] = row[j]
			}
		}

		results = append(results, obj)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return output, err
	}

	return string(data), nil
}

// parseXML parses XML output
func parseXML(output string) (string, error) {
	// Simple XML to JSON conversion
	var result interface{}
	if err := xml.Unmarshal([]byte(output), &result); err != nil {
		return "", fmt.Errorf("failed to parse XML: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return output, err
	}

	return string(data), nil
}

// convertType converts a string value to the specified type
func convertType(value string, typeName string) interface{} {
	switch typeName {
	case "integer":
		// Try to convert to int
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
		return value
	case "float", "number":
		// Try to convert to float
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err == nil {
			return f
		}
		return value
	case "boolean":
		// Try to convert to bool
		lower := strings.ToLower(value)
		if lower == "true" || lower == "yes" || lower == "1" {
			return true
		}
		if lower == "false" || lower == "no" || lower == "0" {
			return false
		}
		return value
	default:
		return value
	}
}