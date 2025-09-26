package parser

import (
	"encoding/json"
	"testing"

	"github.com/charignon/umcp/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJSON(t *testing.T) {
	input := `{"name": "test", "value": 42, "items": ["a", "b", "c"]}`
	result, err := parseJSON(input, "")
	require.NoError(t, err)

	// Verify it's valid formatted JSON
	var data map[string]interface{}
	err = json.Unmarshal([]byte(result), &data)
	require.NoError(t, err)
	assert.Equal(t, "test", data["name"])
	assert.Equal(t, float64(42), data["value"])
}

func TestParseLines(t *testing.T) {
	input := `line1
line2
line3

line4`

	result, err := parseLines(input)
	require.NoError(t, err)

	var lines []string
	err = json.Unmarshal([]byte(result), &lines)
	require.NoError(t, err)
	assert.Equal(t, []string{"line1", "line2", "line3", "line4"}, lines)
}

func TestParseRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pattern  string
		groups   []config.Group
		expected int // number of matches
	}{
		{
			name:    "simple pattern",
			input:   "error: file not found\nwarning: deprecated function\nerror: null pointer",
			pattern: `(error|warning): (.+)`,
			groups: []config.Group{
				{Name: "level", Type: "string"},
				{Name: "message", Type: "string"},
			},
			expected: 3,
		},
		{
			name:     "extract numbers",
			input:    "CPU: 45%\nMemory: 78%\nDisk: 92%",
			pattern:  `(\w+): (\d+)%`,
			groups:   []config.Group{{Name: "resource", Type: "string"}, {Name: "usage", Type: "integer"}},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRegex(tt.input, tt.pattern, tt.groups)
			require.NoError(t, err)

			var matches []map[string]interface{}
			err = json.Unmarshal([]byte(result), &matches)
			require.NoError(t, err)
			assert.Len(t, matches, tt.expected)

			if tt.expected > 0 && len(tt.groups) > 0 {
				// Check that the named groups exist
				for _, group := range tt.groups {
					_, exists := matches[0][group.Name]
					assert.True(t, exists, "Group %s should exist", group.Name)
				}
			}
		})
	}
}

func TestParseCSV(t *testing.T) {
	input := `Name,Age,City
Alice,30,New York
Bob,25,San Francisco
Charlie,35,Chicago`

	result, err := parseCSV(input)
	require.NoError(t, err)

	var data []map[string]string
	err = json.Unmarshal([]byte(result), &data)
	require.NoError(t, err)

	assert.Len(t, data, 3)
	assert.Equal(t, "Alice", data[0]["Name"])
	assert.Equal(t, "30", data[0]["Age"])
	assert.Equal(t, "New York", data[0]["City"])
}

func TestParseOutput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		outputCfg config.Output
		validate  func(t *testing.T, result string)
	}{
		{
			name:  "raw output",
			input: "This is raw output",
			outputCfg: config.Output{
				Type: "raw",
			},
			validate: func(t *testing.T, result string) {
				assert.Equal(t, "This is raw output", result)
			},
		},
		{
			name:  "json output",
			input: `{"test": true}`,
			outputCfg: config.Output{
				Type: "json",
			},
			validate: func(t *testing.T, result string) {
				var data map[string]interface{}
				err := json.Unmarshal([]byte(result), &data)
				require.NoError(t, err)
				assert.Equal(t, true, data["test"])
			},
		},
		{
			name:  "lines output",
			input: "line1\nline2\nline3",
			outputCfg: config.Output{
				Type: "lines",
			},
			validate: func(t *testing.T, result string) {
				var lines []string
				err := json.Unmarshal([]byte(result), &lines)
				require.NoError(t, err)
				assert.Len(t, lines, 3)
			},
		},
		{
			name:  "regex output",
			input: "test123",
			outputCfg: config.Output{
				Type:    "regex",
				Pattern: `([a-zA-Z]+)(\d+)`,
				Groups: []config.Group{
					{Name: "text", Type: "string"},
					{Name: "number", Type: "integer"},
				},
			},
			validate: func(t *testing.T, result string) {
				var matches []map[string]interface{}
				err := json.Unmarshal([]byte(result), &matches)
				require.NoError(t, err)
				assert.Len(t, matches, 1)
				assert.Equal(t, "test", matches[0]["text"])
				// JSON unmarshaling converts numbers to float64
				assert.Equal(t, float64(123), matches[0]["number"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseOutput(tt.input, &tt.outputCfg)
			require.NoError(t, err)
			tt.validate(t, result)
		})
	}
}

func TestConvertType(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		typeName string
		expected interface{}
	}{
		{
			name:     "string to integer",
			value:    "42",
			typeName: "integer",
			expected: 42,
		},
		{
			name:     "string to float",
			value:    "3.14",
			typeName: "float",
			expected: 3.14,
		},
		{
			name:     "string to boolean true",
			value:    "true",
			typeName: "boolean",
			expected: true,
		},
		{
			name:     "string to boolean false",
			value:    "false",
			typeName: "boolean",
			expected: false,
		},
		{
			name:     "invalid integer keeps string",
			value:    "not a number",
			typeName: "integer",
			expected: "not a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertType(tt.value, tt.typeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}