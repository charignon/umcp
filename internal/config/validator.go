package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateFile checks if a file exists
func ValidateFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}
	return nil
}

// ValidateDirectory checks if a directory exists
func ValidateDirectory(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}
	return nil
}

// IsPathAllowed checks if a path is within allowed paths
func IsPathAllowed(path string, allowedPaths []string) bool {
	if len(allowedPaths) == 0 {
		return true // No restrictions
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, allowed := range allowedPaths {
		absAllowed, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, absAllowed) {
			return true
		}
	}
	return false
}

// IsCommandBlocked checks if a command is in the blocked list
func IsCommandBlocked(command string, blockedCommands []string) bool {
	for _, blocked := range blockedCommands {
		if command == blocked {
			return true
		}
	}
	return false
}