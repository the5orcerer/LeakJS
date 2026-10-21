package patterns

import (
	"testing"
)

func TestLoadDefaultPatterns(t *testing.T) {
	pats, err := LoadDefaultPatterns(false)
	if err != nil {
		t.Fatalf("unexpected error loading default patterns: %v", err)
	}
	if len(pats) == 0 {
		t.Fatal("expected default patterns to be loaded, got 0")
	}

	// Verify at least some core patterns exist
	foundAWS := false
	for _, p := range pats {
		if p.Name == "AWS Access Key ID" {
			foundAWS = true
		}
		if p.Compiled == nil {
			t.Errorf("pattern %s compiled regex is nil", p.Name)
		}
	}

	if !foundAWS {
		t.Error("expected 'AWS Access Key ID' pattern in default patterns")
	}
}

func TestGetCompiledPatternCaching(t *testing.T) {
	pat := "test-[0-9]{4}"
	re1, err1 := GetCompiledPattern(pat)
	if err1 != nil {
		t.Fatalf("failed to compile: %v", err1)
	}
	re2, err2 := GetCompiledPattern(pat)
	if err2 != nil {
		t.Fatalf("failed to compile second time: %v", err2)
	}
	if re1 != re2 {
		t.Error("expected cached compiled regex to have identical pointer")
	}
}

func TestValidatePattern(t *testing.T) {
	valid := Pattern{
		Name:       "Test Valid",
		Regex:      `[a-z0-9]{10}`,
		Confidence: "High",
	}
	if err := ValidatePattern(valid); err != nil {
		t.Errorf("expected valid pattern to pass, got: %v", err)
	}

	invalidName := Pattern{
		Name:       "",
		Regex:      `[a-z0-9]{10}`,
		Confidence: "High",
	}
	if err := ValidatePattern(invalidName); err == nil {
		t.Error("expected error for empty name")
	}

	invalidConf := Pattern{
		Name:       "Test Bad Conf",
		Regex:      `[a-z0-9]{10}`,
		Confidence: "SuperHigh",
	}
	if err := ValidatePattern(invalidConf); err == nil {
		t.Error("expected error for invalid confidence level")
	}
}

