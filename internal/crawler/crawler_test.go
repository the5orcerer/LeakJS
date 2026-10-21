package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestExtractJS(t *testing.T) {
	baseURL, _ := url.Parse("https://example.com/app/index.html")

	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<script src="/static/js/main.123.js"></script>
		<script src="relative.js"></script>
		<script src="//cdn.example.com/lib.js?v=2"></script>
		<link rel="preload" as="script" href="/chunks/chunk.js">
		<link rel="modulepreload" href="https://example.com/module.js">
		<script>
			import('./dynamic-chunk.js');
			const api = "/api/v1";
			const extra = "/bundles/extra.bundle.js";
		</script>
	</head>
	<body>
	</body>
	</html>
	`

	extracted := ExtractJS(html, baseURL)
	sort.Strings(extracted)

	expected := []string{
		"https://cdn.example.com/lib.js?v=2",
		"https://example.com/app/dynamic-chunk.js",
		"https://example.com/app/relative.js",
		"https://example.com/bundles/extra.bundle.js",
		"https://example.com/chunks/chunk.js",
		"https://example.com/module.js",
		"https://example.com/static/js/main.123.js",
	}

	foundMap := make(map[string]bool)
	for _, u := range extracted {
		foundMap[u] = true
	}

	for _, exp := range expected {
		if !foundMap[exp] {
			t.Errorf("expected JS URL '%s' to be found in extracted list %v", exp, extracted)
		}
	}
}

func TestApplyHeadersAndCookies(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	ApplyHeaders(req, "session_id=secret123; user=admin", []string{"Authorization: Bearer tokenXYZ", "X-Custom: Value1"}, "CustomAgent/1.0")

	if req.Header.Get("User-Agent") != "CustomAgent/1.0" {
		t.Errorf("unexpected User-Agent: %s", req.Header.Get("User-Agent"))
	}
	if req.Header.Get("Authorization") != "Bearer tokenXYZ" {
		t.Errorf("unexpected Authorization header: %s", req.Header.Get("Authorization"))
	}
	if req.Header.Get("X-Custom") != "Value1" {
		t.Errorf("unexpected X-Custom header: %s", req.Header.Get("X-Custom"))
	}
	cookie := req.Header.Get("Cookie")
	if !strings.Contains(cookie, "session_id=secret123") || !strings.Contains(cookie, "user=admin") {
		t.Errorf("unexpected Cookie header: %s", cookie)
	}
}

func TestCrawlEndToEnd(t *testing.T) {
	receivedCookie := ""
	receivedHeader := ""

	// Set up mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCookie = r.Header.Get("Cookie")
		receivedHeader = r.Header.Get("X-Test-Auth")

		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
				<head>
					<script src="/static/app.js"></script>
				</head>
				<body>
					<a href="/about">About Us</a>
					<a href="/dashboard">Dashboard</a>
					<a href="https://external.com/page">External</a>
				</body>
				</html>
			`)
		case "/about":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
				<head>
					<script src="/static/about.js"></script>
				</head>
				<body>
					<a href="/team">Team</a>
				</body>
				</html>
			`)
		case "/dashboard":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
				<head>
					<script src="/static/dashboard.bundle.js"></script>
				</head>
				<body>
				</body>
				</html>
			`)
		case "/team":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
				<head>
					<script src="/static/team.js"></script>
				</head>
				</html>
			`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	opts := Options{
		Depth:       3,
		MaxPages:    10,
		Concurrency: 2,
		Delay:       10 * time.Millisecond,
		Cookies:     "auth_token=super_secret_cookie",
		Headers:     []string{"X-Test-Auth: authorized_user"},
		Client:      server.Client(),
	}

	jsURLs, err := Crawl(server.URL, opts)
	if err != nil {
		t.Fatalf("crawl returned unexpected error: %v", err)
	}

	if len(jsURLs) < 3 {
		t.Errorf("expected at least 3 JS URLs discovered, got %d: %v", len(jsURLs), jsURLs)
	}

	if !strings.Contains(receivedCookie, "auth_token=super_secret_cookie") {
		t.Errorf("expected mock server to receive auth cookie, got '%s'", receivedCookie)
	}
	if receivedHeader != "authorized_user" {
		t.Errorf("expected mock server to receive auth header, got '%s'", receivedHeader)
	}
}

