package extractor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractEndpoints(t *testing.T) {
	jsCode := `
		const API_HOST = "https://api.example.com/v1/auth/login";
		const USERS_ENDPOINT = "/api/v2/users/list?page=1";
		const INTERNAL_ROUTE = "/internal/metrics/health";
		const GRAPHQL = "/graphql";
		const ASSET = "/static/images/logo.png"; // should be ignored as false positive
		const DOC = "text/html"; // should be ignored

		function fetchData() {
			fetch("/portal/admin/settings")
				.then(r => r.json());
		}
	`

	endpoints := ExtractEndpoints(jsCode)
	found := make(map[string]bool)
	for _, e := range endpoints {
		found[e] = true
	}

	expectedFound := []string{
		"https://api.example.com/v1/auth/login",
		"/api/v2/users/list?page=1",
		"/internal/metrics/health",
		"/portal/admin/settings",
	}

	for _, exp := range expectedFound {
		if !found[exp] {
			t.Errorf("expected endpoint '%s' to be found, got: %v", exp, endpoints)
		}
	}

	// Verify blacklisted false positives are excluded
	for _, e := range endpoints {
		if strings.Contains(e, ".png") || strings.Contains(e, "text/html") {
			t.Errorf("unexpected false positive endpoint returned: %s", e)
		}
	}
}

func TestCheckSourceMap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/app.js.map" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"version":3,"sources":["webpack:///src/app.js"],"mappings":"AAAA"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	jsWithComment := fmt.Sprintf(`console.log("hello");\n//# sourceMappingURL=%s/app.js.map`, server.URL)
	mapURL, leaked := CheckSourceMap(server.URL+"/app.js", jsWithComment, server.Client(), "", nil)
	if !leaked {
		t.Error("expected source map to be detected as leaked")
	}
	if mapURL != server.URL+"/app.js.map" {
		t.Errorf("expected map URL '%s/app.js.map', got '%s'", server.URL, mapURL)
	}
}

