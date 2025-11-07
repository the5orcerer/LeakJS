package patterns

import (
	"log"
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

// LoadPatterns loads and merges both built-in and user patterns
func LoadPatterns(verbose bool) []Pattern {
	builtIn := loadBuiltInPatterns(verbose)
	user, err := LoadUserPatterns(verbose)
	if err != nil && verbose {
		log.Printf("Warning: Error loading user patterns: %v", err)
	}
	return MergePatterns(builtIn, user)
}

func loadBuiltInPatterns(verbose bool) []Pattern {
	builtInPatterns := []struct {
		name       string
		regex      string
		confidence string
	}{
		{"Stripe Secret Key", `sk_live_[0-9a-zA-Z]{24}`, "High"},
		{"Stripe Publishable Key", `pk_live_[0-9a-zA-Z]{24}`, "High"},
		{"Stripe Test Secret Key", `sk_test_[0-9a-zA-Z]{24}`, "Medium"},
		{"Stripe Test Publishable Key", `pk_test_[0-9a-zA-Z]{24}`, "Medium"},
		{"AWS Access Key ID", `AKIA[0-9A-Z]{16}`, "High"},
		{"AWS Secret Access Key", `[0-9a-zA-Z/+]{40}`, "Medium"},
		{"Google API Key", `AIza[0-9A-Za-z-_]{35}`, "High"},
		{"Google OAuth Client ID", `[0-9]+-[0-9A-Za-z_]{32}\.apps\.googleusercontent\.com`, "High"},
		{"Google OAuth Client Secret", `GOCSPX-[0-9A-Za-z_-]{32}`, "High"},
		{"GitHub Personal Access Token", `ghp_[0-9a-zA-Z]{36}`, "High"},
		{"GitHub OAuth Token", `github_pat_[0-9a-zA-Z_]{82}`, "High"},
		{"Slack Token", `xox[baprs]-[0-9a-zA-Z-]{10,48}`, "High"},
		{"Discord Bot Token", `[MN][A-Za-z\d]{23}\.[\w-]{6}\.[\w-]{27}`, "High"},
		{"Twitter Bearer Token", `AAAAAAAAAAAAAAAAAAAAA[0-9a-zA-Z%]{39}`, "High"},
		{"Facebook Access Token", `EAACEdEose0cBA[0-9A-Za-z]+`, "High"},
		{"PayPal Client ID", `A[0-9a-zA-Z-_]{71}`, "High"},
		{"PayPal Client Secret", `E[0-9a-zA-Z-_]{71}`, "High"},
		{"Twilio Account SID", `AC[0-9a-f]{32}`, "High"},
		{"Twilio Auth Token", `[0-9a-f]{32}`, "High"},
		{"SendGrid API Key", `SG\.[0-9A-Za-z_-]{22}\.[0-9A-Za-z_-]{43}`, "High"},
		{"Mailgun API Key", `key-[0-9a-zA-Z]{32}`, "High"},
		{"Firebase API Key", `AAAA[0-9A-Za-z_-]{7}:APA[0-9A-Za-z_-]{178}`, "High"},
		{"JWT Token", `eyJ[A-Za-z0-9-_=]+\.eyJ[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*`, "Medium"},
		{"Generic API Key", `(?i)(api[_-]?key|apikey)\s*[=:]\s*["']?([a-zA-Z0-9_-]{32,})["']?`, "Low"},
		{"Generic Secret", `(?i)(secret|password|token)\s*[=:]\s*["']?([a-zA-Z0-9_-]{32,})["']?`, "Low"},
		{"Private Key", `-----BEGIN [A-Z ]*PRIVATE KEY-----`, "High"},
		{"SSH Private Key", `-----BEGIN OPENSSH PRIVATE KEY-----`, "High"},
		{"Database Connection String", `(mongodb|mysql|postgresql)://[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+@[a-zA-Z0-9.-]+:[0-9]+/[a-zA-Z0-9_-]+`, "High"},
		{"Credit Card Number", `\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`, "High"},
		{"Social Security Number", `\b\d{3}-\d{2}-\d{4}\b`, "High"},
		{"Base64 Encoded", `\b[A-Za-z0-9+/]{40,}={0,2}\b`, "Low"},
	}

	var patterns []Pattern
	for _, p := range builtInPatterns {
		re, err := GetCompiledPattern(p.regex)
		if err != nil {
			if verbose {
				log.Printf("Error compiling built-in pattern %s: %v", p.name, err)
			}
			continue
		}
		patterns = append(patterns, Pattern{
			Name:       p.name,
			Regex:      p.regex,
			Confidence: p.confidence,
			Compiled:   re,
		})
	}
	return patterns
}
