package extractor

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/the5orcerer/LeakJS/internal/crawler"
)

var (
	// Full URLs in JS
	fullURLRegex = regexp.MustCompile(`https?://[a-zA-Z0-9\.\-_]+(?::[0-9]+)?/[a-zA-Z0-9\.\-_/?#&=;%~+]+`)

	// API paths and routes in JS (e.g., "/api/v1/users", "/graphql", "/admin/dashboard", "/internal/settings")
	apiRouteRegex = regexp.MustCompile(`["']((?:/api|/v[0-9]+|/rest|/graphql|/admin|/internal|/user|/auth|/oauth|/webhook|/service|/portal)/[a-zA-Z0-9_\-\./?=&%#]+)["']`)

	// Common clean paths in quotes (at least 2 segments, e.g. "/users/profile")
	cleanPathRegex = regexp.MustCompile(`["'](/([a-zA-Z0-9_\-]+/){1,}[a-zA-Z0-9_\-]+(?:\?[a-zA-Z0-9_\-=&%]*)?)["']`)

	// Source map comments
	sourceMapRegex = regexp.MustCompile(`(?m)//[#@]\s*sourceMappingURL=([^\s]+)|/\*[#@]\s*sourceMappingURL=([^\s]+)\s*\*\/`)

	// Blacklisted false positive path extensions and substrings
	fpBlacklist = []string{
		"text/javascript", "text/html", "text/css", "application/json", "application/x-www-form-urlencoded",
		"multipart/form-data", "image/png", "image/jpeg", "image/svg+xml",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".css", ".woff", ".woff2", ".ttf",
		"w3.org", "schema.org", "react", "webpack", "node_modules",
	}
)

func isFalsePositive(path string) bool {
	lower := strings.ToLower(path)
	for _, fp := range fpBlacklist {
		if strings.Contains(lower, fp) {
			return true
		}
	}
	return false
}

// ExtractEndpoints extracts API endpoints and routes from JavaScript code.
func ExtractEndpoints(content string) []string {
	found := make(map[string]bool)

	// 1. Full URLs
	for _, match := range fullURLRegex.FindAllString(content, -1) {
		match = strings.TrimSpace(match)
		if !isFalsePositive(match) {
			found[match] = true
		}
	}

	// 2. High-confidence API routes
	for _, match := range apiRouteRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			route := strings.TrimSpace(match[1])
			if !isFalsePositive(route) && len(route) > 3 {
				found[route] = true
			}
		}
	}

	// 3. Clean paths
	for _, match := range cleanPathRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			route := strings.TrimSpace(match[1])
			if !isFalsePositive(route) && len(route) > 4 {
				found[route] = true
			}
		}
	}

	var results []string
	for endpoint := range found {
		results = append(results, endpoint)
	}
	return results
}

// CheckSourceMap checks if a JavaScript file exposes a valid, accessible source map (.map).
func CheckSourceMap(jsURL string, content string, client *http.Client, cookies string, headers []string) (string, bool) {
	if client == nil {
		c, err := crawler.BuildHTTPClient(0, "", false)
		if err != nil {
			return "", false
		}
		client = c
	}

	baseURL, err := url.Parse(jsURL)
	if err != nil {
		return "", false
	}

	var candidateMapURLs []string

	// Look for sourceMappingURL comment
	for _, m := range sourceMapRegex.FindAllStringSubmatch(content, -1) {
		ref := ""
		if m[1] != "" {
			ref = m[1]
		} else if len(m) > 2 && m[2] != "" {
			ref = m[2]
		}
		if ref != "" && !strings.HasPrefix(ref, "data:") {
			if resolved := crawler.ResolveURL(baseURL, ref); resolved != "" {
				candidateMapURLs = append(candidateMapURLs, resolved)
			}
		}
	}

	// Also fallback to jsURL + ".map" if none found
	if len(candidateMapURLs) == 0 && strings.HasSuffix(baseURL.Path, ".js") {
		candidateMapURLs = append(candidateMapURLs, jsURL+".map")
	}

	for _, mapURL := range candidateMapURLs {
		req, err := http.NewRequest("GET", mapURL, nil)
		if err != nil {
			continue
		}
		crawler.ApplyHeaders(req, cookies, headers, "")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			// Read first 2KB to verify JSON with sources or mappings
			buf := make([]byte, 2048)
			n, _ := io.ReadFull(resp.Body, buf)
			prefix := string(buf[:n])
			if strings.Contains(prefix, `"sources"`) || strings.Contains(prefix, `"mappings"`) {
				var sm struct {
					Version int `json:"version"`
				}
				if json.Unmarshal([]byte(prefix[:strings.LastIndex(prefix, "}")+1]), &sm) == nil || sm.Version >= 3 {
					return mapURL, true
				}
				if strings.Contains(prefix, `"version"`) && strings.Contains(prefix, `"sources"`) {
					return mapURL, true
				}
			}
		}
	}

	return "", false
}

