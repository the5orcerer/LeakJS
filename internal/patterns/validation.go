package patterns

import (
	"fmt"
	"regexp"
)

// ValidationError represents an error in pattern validation
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidatePattern checks if a pattern is valid
func ValidatePattern(pattern Pattern) error {
	// Check required fields
	if pattern.Name == "" {
		return &ValidationError{Field: "name", Message: "pattern name cannot be empty"}
	}
	if pattern.Regex == "" {
		return &ValidationError{Field: "regex", Message: "regex pattern cannot be empty"}
	}

	// Validate confidence level
	validConfidence := map[string]bool{
		"High":   true,
		"Medium": true,
		"Low":    true,
	}
	if !validConfidence[pattern.Confidence] {
		return &ValidationError{
			Field:   "confidence",
			Message: "confidence must be one of: High, Medium, Low",
		}
	}

	// Validate regex syntax
	if _, err := regexp.Compile(pattern.Regex); err != nil {
		return &ValidationError{
			Field:   "regex",
			Message: fmt.Sprintf("invalid regex pattern: %v", err),
		}
	}

	return nil
}

// ValidatePatternFile validates the structure and content of a pattern file
func ValidatePatternFile(config UserPatternConfig) error {
	return ValidatePattern(config.Pattern)
}
