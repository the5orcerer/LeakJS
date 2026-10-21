package crawler

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// Regex patterns for script sources
	scriptSrcRegex       = regexp.MustCompile(`(?i)<script[^>]+src\s*=\s*(?:"([^"]+)"|'([^']+)'|([^"'>\s]+))`)
	linkJSRegex          = regexp.MustCompile(`(?i)<link[^>]+href\s*=\s*(?:"([^"]+\.js(?:\?[^"]*)?)"|'([^']+\.js(?:\?[^']*)?)'|([^"'>\s]+\.js(?:\?[^"'>\s]*)?))`)
	linkPreloadScript    = regexp.MustCompile(`(?i)<link[^>]+(?:as=["']script["'][^>]+href\s*=\s*(?:"([^"]+)"|'([^']+)'|([^"'>\s]+))|href\s*=\s*(?:"([^"]+)"|'([^']+)'|([^"'>\s]+))[^>]+as=["']script["'])`)
	inlineScriptTagRegex = regexp.MustCompile(`(?is)<script[^>]*>(.*?)</script>`)
	dynamicImportRegex   = regexp.MustCompile(`(?i)import\s*\(\s*['"]([^'"]+\.js(?:\?[^'"]*)?)['"]\s*\)`)
	jsPathRegex          = regexp.MustCompile(`['"]((?:/[a-zA-Z0-9_\-\.]+)+/[a-zA-Z0-9_\-\.]+\.js(?:\?[a-zA-Z0-9_\-=&%]*)?)['"]`)
	jsFullURLRegex       = regexp.MustCompile(`['"]((?:https?:)?//[a-zA-Z0-9_\-\.]+(?::[0-9]+)?/[a-zA-Z0-9_\-\./]+\.js(?:\?[a-zA-Z0-9_\-=&%]*)?)['"]`)

	// Regex for crawling hyperlinks
	aHrefRegex = regexp.MustCompile(`(?i)<a[^>]+href\s*=\s*(?:"([^"#]+)"|'([^'#]+)'|([^"'>\s#]+))`)

	// Static media extensions to skip when crawling HTML pages
	skipExtensions = map[string]bool{
		".png":   true,
		".jpg":   true,
		".jpeg":  true,
		".gif":   true,
		".svg":   true,
		".ico":   true,
		".webp":  true,
		".css":   true,
		".woff":  true,
		".woff2": true,
		".ttf":   true,
		".eot":   true,
		".otf":   true,
		".pdf":   true,
		".zip":   true,
		".tar":   true,
		".gz":    true,
		".mp3":   true,
		".mp4":   true,
		".avi":   true,
		".mov":   true,
		".wmv":   true,
		".xml":   true,
		".json":  true,
	}
)

type Options struct {
	Depth       int
	MaxPages    int
	Concurrency int
	Delay       time.Duration
	ScopeHost   string
	Subdomains  bool
	Client      *http.Client
	Headers     []string
	Cookies     string
	UserAgent   string
	Verbose     bool
	Silent      bool
}

type queueItem struct {
	URL   string
	Depth int
}

// BuildHTTPClient creates an http.Client configured with timeouts, proxy, TLS settings, cookies, and headers.
func BuildHTTPClient(timeout time.Duration, proxyStr string, insecure bool) (*http.Client, error) {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	if proxyStr != "" {
		pURL, err := url.Parse(proxyStr)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL '%s': %w", proxyStr, err)
		}
		transport.Proxy = http.ProxyURL(pURL)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			// Forward Cookies and Authorization headers to redirects on the same root domain
			if len(via) > 0 {
				lastReq := via[len(via)-1]
				lastDomain := strings.TrimPrefix(strings.ToLower(lastReq.URL.Hostname()), "www.")
				newDomain := strings.TrimPrefix(strings.ToLower(req.URL.Hostname()), "www.")
				if lastDomain == newDomain || strings.HasSuffix(newDomain, "."+lastDomain) {
					if cookie := lastReq.Header.Get("Cookie"); cookie != "" && req.Header.Get("Cookie") == "" {
						req.Header.Set("Cookie", cookie)
					}
					if auth := lastReq.Header.Get("Authorization"); auth != "" && req.Header.Get("Authorization") == "" {
						req.Header.Set("Authorization", auth)
					}
				}
			}
			return nil
		},
	}

	return client, nil
}

// ApplyHeaders attaches standard browser headers, cookies, and custom headers to an HTTP request.
func ApplyHeaders(req *http.Request, cookies string, headers []string, userAgent string) {
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	}

	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	}
	if req.Header.Get("Sec-Ch-Ua") == "" {
		req.Header.Set("Sec-Ch-Ua", `"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`)
	}
	if req.Header.Get("Sec-Ch-Ua-Mobile") == "" {
		req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	}
	if req.Header.Get("Sec-Ch-Ua-Platform") == "" {
		req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	}
	if req.Header.Get("Sec-Fetch-Dest") == "" {
		req.Header.Set("Sec-Fetch-Dest", "document")
	}
	if req.Header.Get("Sec-Fetch-Mode") == "" {
		req.Header.Set("Sec-Fetch-Mode", "navigate")
	}
	if req.Header.Get("Sec-Fetch-Site") == "" {
		req.Header.Set("Sec-Fetch-Site", "none")
	}
	if req.Header.Get("Sec-Fetch-User") == "" {
		req.Header.Set("Sec-Fetch-User", "?1")
	}
	if req.Header.Get("Upgrade-Insecure-Requests") == "" {
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}

	// Attach Cookie
	if cookies != "" {
		existing := req.Header.Get("Cookie")
		if existing != "" {
			req.Header.Set("Cookie", existing+"; "+cookies)
		} else {
			req.Header.Set("Cookie", cookies)
		}
	}

	// Attach custom headers
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			if strings.EqualFold(k, "cookie") {
				existing := req.Header.Get("Cookie")
				if existing != "" {
					req.Header.Set("Cookie", existing+"; "+v)
				} else {
					req.Header.Set("Cookie", v)
				}
			} else {
				req.Header.Set(k, v)
			}
		}
	}
}

// NormalizeTargetURL ensures the target has an http:// or https:// scheme and cleans trailing slashes.
func NormalizeTargetURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err == nil {
		// Strip trailing slash on path if it has subpaths (e.g. /brand-portal/ -> /brand-portal)
		if len(u.Path) > 1 && strings.HasSuffix(u.Path, "/") {
			u.Path = strings.TrimRight(u.Path, "/")
			raw = u.String()
		}
	}
	return raw
}

// ResolveURL resolves a relative URL reference against a base page URL.
func ResolveURL(baseURL *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	// Unescape JSON escaped slashes if any
	ref = strings.ReplaceAll(ref, `\/`, `/`)

	if ref == "" || strings.HasPrefix(ref, "javascript:") || strings.HasPrefix(ref, "data:") || strings.HasPrefix(ref, "mailto:") {
		return ""
	}

	// Handle protocol-relative URL
	if strings.HasPrefix(ref, "//") {
		ref = baseURL.Scheme + ":" + ref
	}

	parsedRef, err := url.Parse(ref)
	if err != nil {
		return ""
	}

	resolved := baseURL.ResolveReference(parsedRef)
	// Strip fragment
	resolved.Fragment = ""

	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}

	return resolved.String()
}

// ExtractJS finds all JavaScript references in an HTML string.
func ExtractJS(htmlContent string, baseURL *url.URL) []string {
	found := make(map[string]bool)

	// Unescape JSON escaped slashes in the HTML content for Next.js / React chunk extraction
	cleanHTML := strings.ReplaceAll(htmlContent, `\/`, `/`)

	// 1. <script src="...">
	for _, m := range scriptSrcRegex.FindAllStringSubmatch(cleanHTML, -1) {
		for i := 1; i < len(m); i++ {
			if m[i] != "" {
				if u := ResolveURL(baseURL, m[i]); u != "" {
					found[u] = true
				}
			}
		}
	}

	// 2. <link href="...js">
	for _, m := range linkJSRegex.FindAllStringSubmatch(cleanHTML, -1) {
		for i := 1; i < len(m); i++ {
			if m[i] != "" {
				if u := ResolveURL(baseURL, m[i]); u != "" {
					found[u] = true
				}
			}
		}
	}

	// 3. <link rel="preload" as="script" href="...">
	for _, m := range linkPreloadScript.FindAllStringSubmatch(cleanHTML, -1) {
		for i := 1; i < len(m); i++ {
			if m[i] != "" {
				if u := ResolveURL(baseURL, m[i]); u != "" {
					found[u] = true
				}
			}
		}
	}

	// 4. Inline scripts: inspect dynamic imports and referenced JS chunk paths
	for _, sm := range inlineScriptTagRegex.FindAllStringSubmatch(cleanHTML, -1) {
		if len(sm) > 1 {
			inlineCode := sm[1]
			for _, m := range dynamicImportRegex.FindAllStringSubmatch(inlineCode, -1) {
				if len(m) > 1 && m[1] != "" {
					if u := ResolveURL(baseURL, m[1]); u != "" {
						found[u] = true
					}
				}
			}
			for _, m := range jsPathRegex.FindAllStringSubmatch(inlineCode, -1) {
				if len(m) > 1 && m[1] != "" {
					if u := ResolveURL(baseURL, m[1]); u != "" {
						found[u] = true
					}
				}
			}
			for _, m := range jsFullURLRegex.FindAllStringSubmatch(inlineCode, -1) {
				if len(m) > 1 && m[1] != "" {
					if u := ResolveURL(baseURL, m[1]); u != "" {
						found[u] = true
					}
				}
			}
		}
	}

	var results []string
	for u := range found {
		results = append(results, u)
	}
	return results
}

// ExtractLinks extracts navigable HTML links within the crawl scope.
func ExtractLinks(htmlContent string, baseURL *url.URL, scopeHost string, allowSubdomains bool) []string {
	found := make(map[string]bool)
	rootDomain := strings.TrimPrefix(strings.ToLower(scopeHost), "www.")

	for _, m := range aHrefRegex.FindAllStringSubmatch(htmlContent, -1) {
		raw := ""
		for i := 1; i < len(m); i++ {
			if m[i] != "" {
				raw = m[i]
				break
			}
		}
		if raw == "" {
			continue
		}

		resolved := ResolveURL(baseURL, raw)
		if resolved == "" {
			continue
		}

		parsed, err := url.Parse(resolved)
		if err != nil {
			continue
		}

		// Scope check
		host := strings.ToLower(parsed.Hostname())
		hostRoot := strings.TrimPrefix(host, "www.")

		if allowSubdomains {
			if hostRoot != rootDomain && !strings.HasSuffix(host, "."+rootDomain) {
				continue
			}
		} else {
			if hostRoot != rootDomain {
				continue
			}
		}

		// Check file extension
		pathLower := strings.ToLower(parsed.Path)
		skip := false
		for ext := range skipExtensions {
			if strings.HasSuffix(pathLower, ext) {
				skip = true
				break
			}
		}
		if skip || strings.HasSuffix(pathLower, ".js") {
			continue
		}

		found[resolved] = true
	}

	var results []string
	for u := range found {
		results = append(results, u)
	}
	return results
}

// Crawl visits the target website up to opts.Depth and gathers all JavaScript URLs.
func Crawl(targetURL string, opts Options) ([]string, error) {
	targetURL = NormalizeTargetURL(targetURL)
	seedParsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	if opts.ScopeHost == "" {
		opts.ScopeHost = seedParsed.Hostname()
	}
	if opts.Depth <= 0 {
		opts.Depth = 2
	}
	if opts.MaxPages <= 0 {
		opts.MaxPages = 50
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 10
	}
	if opts.Client == nil {
		c, err := BuildHTTPClient(15*time.Second, "", false)
		if err != nil {
			return nil, err
		}
		opts.Client = c
	}

	var (
		visitedPages sync.Map
		discoveredJS sync.Map
		queue        = make(chan queueItem, opts.MaxPages*10+200)
		wg           sync.WaitGroup
		pageCount    int32
	)

	// Clean root domain for scope checking
	rootDomain := strings.TrimPrefix(strings.ToLower(opts.ScopeHost), "www.")

	// Launch worker pool
	for i := 0; i < opts.Concurrency; i++ {
		go func() {
			for item := range queue {
				func() {
					defer wg.Done()

					if opts.Delay > 0 {
						time.Sleep(opts.Delay)
					}

					if opts.Verbose {
						log.Printf("[CRAWL] [Depth %d] Fetching: %s", item.Depth, item.URL)
					}

					req, err := http.NewRequest("GET", item.URL, nil)
					if err != nil {
						if opts.Verbose {
							log.Printf("[CRAWL] Error creating request for %s: %v", item.URL, err)
						}
						return
					}
					ApplyHeaders(req, opts.Cookies, opts.Headers, opts.UserAgent)

					resp, err := opts.Client.Do(req)
					if err != nil {
						if opts.Verbose {
							log.Printf("[CRAWL] Error fetching %s: %v", item.URL, err)
						}
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode == 403 {
						if opts.Verbose {
							log.Printf("[CRAWL] Warning: HTTP 403 Forbidden for %s (bot protection or expired cookie)", item.URL)
						}
					} else if resp.StatusCode >= 400 && opts.Verbose {
						log.Printf("[CRAWL] Warning: HTTP %d for %s", resp.StatusCode, item.URL)
					}

					cType := strings.ToLower(resp.Header.Get("Content-Type"))
					if !strings.Contains(cType, "html") && !strings.Contains(cType, "javascript") && !strings.Contains(cType, "text") && cType != "" {
						return
					}

					bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
					if err != nil {
						if opts.Verbose {
							log.Printf("[CRAWL] Error reading body of %s: %v", item.URL, err)
						}
						return
					}

					bodyStr := string(bodyBytes)
					pageURL := resp.Request.URL
					if pageURL == nil {
						pageURL = seedParsed
					}

					// Extract JavaScript references
					jsList := ExtractJS(bodyStr, pageURL)
					for _, jsURL := range jsList {
						discoveredJS.Store(jsURL, true)
					}

					// Extract child links if depth allows
					if item.Depth < opts.Depth {
						links := ExtractLinks(bodyStr, pageURL, rootDomain, opts.Subdomains)
						for _, link := range links {
							if atomic.LoadInt32(&pageCount) >= int32(opts.MaxPages) {
								break
							}

							if _, loaded := visitedPages.LoadOrStore(link, true); !loaded {
								if atomic.AddInt32(&pageCount, 1) <= int32(opts.MaxPages) {
									wg.Add(1)
									select {
									case queue <- queueItem{URL: link, Depth: item.Depth + 1}:
									default:
										wg.Done()
									}
								}
							}
						}
					}
				}()
			}
		}()
	}

	// Enqueue seed URL
	visitedPages.Store(targetURL, true)
	atomic.StoreInt32(&pageCount, 1)
	wg.Add(1)
	queue <- queueItem{URL: targetURL, Depth: 1}

	// Wait for all crawl tasks to complete
	wg.Wait()
	close(queue)

	var jsURLs []string
	discoveredJS.Range(func(key, _ interface{}) bool {
		jsURLs = append(jsURLs, key.(string))
		return true
	})

	return jsURLs, nil
}
