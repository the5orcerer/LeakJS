package patterns

import (
	"regexp"
	"sync"
)

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

// LoadPatterns is just an alias for LoadUserPatterns for backward compatibility
func LoadPatterns(verbose bool) []Pattern {
	patterns, _ := LoadUserPatterns(verbose)
	return patterns
}
