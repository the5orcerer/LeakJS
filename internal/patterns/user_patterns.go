package patterns

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// UserPatternConfig represents a single pattern file structure
type UserPatternConfig struct {
	Pattern Pattern `yaml:"pattern"`
}

// EnsureUserPatternsDir creates the user patterns directory if it doesn't exist
func EnsureUserPatternsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	// Primary location: ~/.config/leakjs/patterns
	patternsDir := filepath.Join(homeDir, ".config", "leakjs", "patterns")
	if err := os.MkdirAll(patternsDir, 0755); err != nil {
		// Fallback to ~/.leakjs/patterns
		patternsDir = filepath.Join(homeDir, ".leakjs", "patterns")
		if err := os.MkdirAll(patternsDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create patterns directory: %v", err)
		}
	}

	return patternsDir, nil
}

// LoadUserPatterns loads all pattern files from the user's patterns directory
func LoadUserPatterns(verbose bool) ([]Pattern, error) {
	patternsDir, err := EnsureUserPatternsDir()
	if err != nil {
		if verbose {
			fmt.Printf("Warning: Could not access user patterns directory: %v\n", err)
		}
		return nil, nil
	}

	var patterns []Pattern
	err = filepath.Walk(patternsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-yaml files
		if info.IsDir() || (!strings.HasSuffix(info.Name(), ".yaml") && !strings.HasSuffix(info.Name(), ".yml")) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Could not read pattern file %s: %v\n", path, err)
			}
			return nil
		}

		var config UserPatternConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			if verbose {
				fmt.Printf("Warning: Invalid pattern file %s: %v\n", path, err)
			}
			return nil
		}

		// Validate pattern
		if err := ValidatePatternFile(config); err != nil {
			if verbose {
				fmt.Printf("Warning: Invalid pattern in file %s: %v\n", path, err)
			}
			return nil
		}

		// Compile the pattern
		re, err := GetCompiledPattern(config.Pattern.Regex)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Invalid regex in pattern file %s: %v\n", path, err)
			}
			return nil
		}

		pattern := config.Pattern
		pattern.Compiled = re
		patterns = append(patterns, pattern)

		return nil
	})

	if err != nil && verbose {
		fmt.Printf("Warning: Error walking patterns directory: %v\n", err)
	}

	return patterns, nil
}

// MergePatterns combines built-in and user patterns, with user patterns taking precedence
func MergePatterns(builtIn []Pattern, user []Pattern) []Pattern {
	// Create a map to track pattern names to avoid duplicates
	patternMap := make(map[string]Pattern)

	// Add built-in patterns first
	for _, p := range builtIn {
		patternMap[p.Name] = p
	}

	// Add user patterns, overwriting any built-in patterns with the same name
	for _, p := range user {
		patternMap[p.Name] = p
	}

	// Convert map back to slice
	var merged []Pattern
	for _, p := range patternMap {
		merged = append(merged, p)
	}

	return merged
}
