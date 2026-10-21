package patterns

import (
	_ "embed"
	"fmt"
	"log"
	"regexp"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed default_patterns.yaml
var DefaultPatternsYAML []byte

var (
	patternCache = make(map[string]*regexp.Regexp)
	cacheMutex   sync.RWMutex
)

func GetCompiledPattern(regex string) (*regexp.Regexp, error) {
	cacheMutex.RLock()
	if re, exists := patternCache[regex]; exists {
		cacheMutex.RUnlock()
		return re, nil
	}
	cacheMutex.RUnlock()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Double-check in case another goroutine compiled it
	if re, exists := patternCache[regex]; exists {
		return re, nil
	}

	re, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}

	patternCache[regex] = re
	return re, nil
}

type Pattern struct {
	Name       string         `yaml:"name"`
	Regex      string         `yaml:"regex"`
	Confidence string         `yaml:"confidence"`
	Compiled   *regexp.Regexp `yaml:"-"`
}

type PatternConfig struct {
	Pattern Pattern `yaml:"pattern"`
}

type Config struct {
	Patterns []PatternConfig `yaml:"patterns"`
}

// LoadDefaultPatterns loads patterns embedded into the binary at compile time.
func LoadDefaultPatterns(verbose bool) ([]Pattern, error) {
	var config Config
	if err := yaml.Unmarshal(DefaultPatternsYAML, &config); err != nil {
		return nil, fmt.Errorf("failed to parse embedded default patterns: %w", err)
	}

	var pats []Pattern
	for _, pc := range config.Patterns {
		re, err := GetCompiledPattern(pc.Pattern.Regex)
		if err != nil {
			if verbose {
				log.Printf("Invalid regex in embedded pattern %s: %v", pc.Pattern.Name, err)
			}
			continue
		}
		p := pc.Pattern
		p.Compiled = re
		pats = append(pats, p)
	}
	return pats, nil
}

// LoadPatterns is an alias for LoadUserPatterns for backward compatibility.
// If no user patterns exist, it falls back to embedded default patterns.
func LoadPatterns(verbose bool) []Pattern {
	patterns, err := LoadUserPatterns(verbose)
	if err == nil && len(patterns) > 0 {
		return patterns
	}
	embedded, _ := LoadDefaultPatterns(verbose)
	return embedded
}
