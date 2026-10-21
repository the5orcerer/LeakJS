package scanner

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/the5orcerer/LeakJS/internal/patterns"
)

func TestFilterPatterns(t *testing.T) {
	pats := []patterns.Pattern{
		{Name: "AWS Key", Confidence: "High"},
		{Name: "Slack Token", Confidence: "High"},
		{Name: "Generic Secret", Confidence: "Low"},
	}

	// Exclude test
	filtered := FilterPatterns(pats, "", "Generic Secret")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 patterns after exclusion, got %d", len(filtered))
	}
	for _, p := range filtered {
		if p.Name == "Generic Secret" {
			t.Errorf("expected Generic Secret to be excluded")
		}
	}

	// Include test
	filtered = FilterPatterns(pats, "Slack Token", "")
	if len(filtered) != 1 || filtered[0].Name != "Slack Token" {
		t.Fatalf("expected only Slack Token to be included")
	}
}

func TestSearchPatterns(t *testing.T) {
	re, _ := patterns.GetCompiledPattern(`AKIA[0-9A-Z]{16}`)
	pats := []patterns.Pattern{
		{Name: "AWS Access Key", Confidence: "High", Compiled: re},
	}

	content := `const aws = "AKIAIOSFODNN7EXAMPLE";`
	matches := SearchPatterns(content, pats, false)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if _, ok := matches["AWS Access Key"]; !ok {
		t.Errorf("expected match for AWS Access Key")
	}
}

func TestRunEmbeddedPatternsFallback(t *testing.T) {
	// Create a temporary JS file with a known leak
	tmpDir := t.TempDir()
	jsFile := filepath.Join(tmpDir, "test.js")
	err := os.WriteFile(jsFile, []byte(`
		// Leaked credentials
		const token = "xoxb-123456789012-1234567890123-abcdefghijklmnopqrstuvwxyz0123456789";
	`), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "results.txt")
	stats := &ScanStats{}
	opts := ScanOptions{
		FilePath:   jsFile,
		OutputFile: outFile,
		ScanStats:  stats,
		Silent:     true,
		// No PatternsFile or DirectPatterns or AllDir specified!
	}

	err = Run(opts)
	if err != nil {
		t.Fatalf("Run failed with embedded patterns: %v", err)
	}

	if stats.PatternsUsed == 0 {
		t.Error("expected embedded patterns to be loaded and used")
	}
	if stats.FilesScanned != 1 {
		t.Errorf("expected 1 file scanned, got %d", stats.FilesScanned)
	}
}

func TestRunCrawlWithCookiesAndEndpoints(t *testing.T) {
	var gotCookie, gotHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotHeader = r.Header.Get("X-Custom-Auth")

		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
				<head>
					<script src="/static/app.js"></script>
				</head>
				<body>Welcome</body>
				</html>
			`)
		case "/static/app.js":
			w.Header().Set("Content-Type", "application/javascript")
			fmt.Fprint(w, `
				const API_URL = "/api/v1/user/profile";
				const SECRET_KEY = "AKIA1111111111111111";
			`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "crawl_results.txt")

	stats := &ScanStats{}
	opts := ScanOptions{
		SingleURL:   server.URL,
		Crawl:       true,
		Depth:       1,
		MaxPages:    5,
		Cookies:     "session=auth_secret_xyz",
		Headers:     []string{"X-Custom-Auth: active_session"},
		Endpoints:   true,
		OutputFile:  outFile,
		ScanStats:   stats,
		Silent:      true,
		Concurrency: 2,
	}

	err := Run(opts)
	if err != nil {
		t.Fatalf("Run crawl mode failed: %v", err)
	}

	if !strings.Contains(gotCookie, "session=auth_secret_xyz") {
		t.Errorf("expected crawler to send session cookie, got: '%s'", gotCookie)
	}
	if gotHeader != "active_session" {
		t.Errorf("expected crawler to send custom header, got: '%s'", gotHeader)
	}

	if stats.URLsScanned != 1 {
		t.Errorf("expected 1 JS URL scanned, got %d", stats.URLsScanned)
	}
	if stats.TotalMatches < 1 {
		t.Errorf("expected at least 1 match found in crawled JS, got %d", stats.TotalMatches)
	}
	if stats.EndpointsFound < 1 {
		t.Errorf("expected at least 1 endpoint found in crawled JS, got %d", stats.EndpointsFound)
	}

	outContent, _ := os.ReadFile(outFile)
	if !strings.Contains(string(outContent), "AKIA1111111111111111") {
		t.Errorf("expected output file to contain AWS key finding")
	}
	if !strings.Contains(string(outContent), "/api/v1/user/profile") {
		t.Errorf("expected output file to contain extracted endpoint")
	}
}

func TestCrawlOnlyMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
			<html>
			<head>
				<script src="/static/vendor.js"></script>
				<script src="/static/main.js"></script>
			</head>
			</html>
		`)
	}))
	defer server.Close()

	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "gathered_js.txt")

	opts := ScanOptions{
		SingleURL:  server.URL,
		Crawl:      true,
		CrawlOnly:  true,
		OutputFile: outFile,
		Silent:     true,
	}

	err := Run(opts)
	if err != nil {
		t.Fatalf("CrawlOnly run failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "/static/vendor.js") || !strings.Contains(content, "/static/main.js") {
		t.Errorf("output file missing crawled JS URLs: %s", content)
	}
}
