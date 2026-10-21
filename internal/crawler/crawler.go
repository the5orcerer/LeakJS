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
	"time"
)

var (
	// Regex patterns for script sources
	scriptSrcRegex       = regexp.MustCompile(`(?i)<script[^>]+src=["']?([^"'>\s]+)`)
	linkJSRegex          = regexp.MustCompile(`(?i)<link[^>]+href=["']?([^"'>\s]+\.js(?:\?[^"'>\s]*)?)["']`)
	linkPreloadScript    = regexp.MustCompile(`(?i)<link[^>]+(?:as=["']script["'][^>]+href=["']?([^"'>\s]+)|href=["']?([^"'>\s]+)["'][^>]+as=["']script["'])`)
	inlineScriptTagRegex = regexp.MustCompile(`(?is)<script[^>]*>(.*?)</script>`)
	dynamicImportRegex   = regexp.MustCompile(`(?i)import\s*\(\s*['"]([^'"]+\.js(?:\?[^'"]*)?)['"]\s*\)`)
	jsPathRegex          = regexp.MustCompile(`['"]((?:/[a-zA-Z0-9_\-\.]+)+/[a-zA-Z0-9_\-\.]+\.js(?:\?[a-zA-Z0-9_\-=&%]*)?)['"]`)
	jsFullURLRegex       = regexp.MustCompile(`['"]((?:https?:)?//[a-zA-Z0-9_\-\.]+(?::[0-9]+)?/[a-zA-Z0-9_\-\./]+\.js(?:\?[a-zA-Z0-9_\-=&%]*)?)['"]`)

	// Regex for crawling hyperlinks
	aHrefRegex = regexp.MustCompile(`(?i)<a[^>]+href=["']?([^"'>\s#]+)`)

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
	}

	return client, nil
}

// ApplyHeaders attaches standard headers, cookies, and custom headers to an HTTP request.
func ApplyHeaders(req *http.Request, cookies string, headers []string, userAgent string) {
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	}

	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
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

// NormalizeTargetURL ensures the target has an http:// or https:// scheme.
func NormalizeTargetURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "https://" + raw
	}
	return raw
}

// ResolveURL resolves a relative URL reference against a base page URL.
func ResolveURL(baseURL *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
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

	// 1. <script src="...">
	for _, m := range scriptSrcRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(m) > 1 {
			if u := ResolveURL(baseURL, m[1]); u != "" {
				found[u] = true
			}
		}
	}

	// 2. <link href="...js">
	for _, m := range linkJSRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(m) > 1 {
			if u := ResolveURL(baseURL, m[1]); u != "" {
				found[u] = true
			}
		}
	}

	// 3. <link rel="preload" as="script" href="...">
	for _, m := range linkPreloadScript.FindAllStringSubmatch(htmlContent, -1) {
		for i := 1; i < len(m); i++ {
			if m[i] != "" {
				if u := ResolveURL(baseURL, m[i]); u != "" {
					found[u] = true
				}
			}
		}
	}

	// 4. Inline scripts: inspect dynamic imports and referenced JS chunk paths
	for _, sm := range inlineScriptTagRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(sm) > 1 {
			inlineCode := sm[1]
			for _, m := range dynamicImportRegex.FindAllStringSubmatch(inlineCode, -1) {
				if len(m) > 1 {
					if u := ResolveURL(baseURL, m[1]); u != "" {
						found[u] = true
					}
				}
			}
			for _, m := range jsPathRegex.FindAllStringSubmatch(inlineCode, -1) {
				if len(m) > 1 {
					if u := ResolveURL(baseURL, m[1]); u != "" {
						found[u] = true
					}
				}
			}
			for _, m := range jsFullURLRegex.FindAllStringSubmatch(inlineCode, -1) {
				if len(m) > 1 {
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

	for _, m := range aHrefRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(m) <= 1 {
			continue
		}
		raw := m[1]
		resolved := ResolveURL(baseURL, raw)
		if resolved == "" {
			continue
		}

		parsed, err := url.Parse(resolved)
		if err != nil {
			continue
		}

		// Scope check
		host := parsed.Hostname()
		if allowSubdomains {
			if host != scopeHost && !strings.HasSuffix(host, "."+scopeHost) {
				continue
			}
		} else {
			if host != scopeHost {
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
		queue        = make(chan queueItem, opts.MaxPages*5)
		activeTasks  int32
		mu           sync.Mutex
		done         = make(chan struct{})
	)

	// Enqueue root page
	visitedPages.Store(targetURL, true)
	queue <- queueItem{URL: targetURL, Depth: 1}

	var wg sync.WaitGroup
	workerSem := make(chan struct{}, opts.Concurrency)

	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case item, ok := <-queue:
					if !ok {
						return
					}

					workerSem <- struct{}{}
					if opts.Delay > 0 {
						time.Sleep(opts.Delay)
					}

					if opts.Verbose {
						log.Printf("[CRAWL] [Depth %d] Fetching: %s", item.Depth, item.URL)
					}

					req, err := http.NewRequest("GET", item.URL, nil)
					if err == nil {
						ApplyHeaders(req, opts.Cookies, opts.Headers, opts.UserAgent)
						resp, err := opts.Client.Do(req)
						if err == nil {
							defer resp.Body.Close()

							cType := resp.Header.Get("Content-Type")
							// Read only if HTML or JavaScript or text
							if strings.Contains(cType, "html") || strings.Contains(cType, "javascript") || strings.Contains(cType, "text") || cType == "" {
								bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5MB limit
								if readErr == nil {
									bodyStr := string(bodyBytes)
									pageURL := resp.Request.URL
									if pageURL == nil {
										pageURL = seedParsed
									}

									// Extract JS files
									jsList := ExtractJS(bodyStr, pageURL)
									for _, jsURL := range jsList {
										discoveredJS.Store(jsURL, true)
									}

									// Crawl child pages if depth allows
									if item.Depth < opts.Depth {
										childLinks := ExtractLinks(bodyStr, pageURL, opts.ScopeHost, opts.Subdomains)
										for _, link := range childLinks {
											mu.Lock()
											count := 0
											visitedPages.Range(func(_, _ interface{}) bool {
												count++
												return true
											})
											if count >= opts.MaxPages {
												mu.Unlock()
												break
											}

											if _, loaded := visitedPages.LoadOrStore(link, true); !loaded {
												select {
												case queue <- queueItem{URL: link, Depth: item.Depth + 1}:
												default:
												}
											}
											mu.Unlock()
										}
									}
								}
							}
						} else if opts.Verbose {
							log.Printf("[CRAWL] Error fetching %s: %v", item.URL, err)
						}
					}
					<-workerSem

					mu.Lock()
					activeTasks--
					if len(queue) == 0 && activeTasks <= 0 {
						select {
						case <-done:
						default:
							close(done)
						}
						mu.Unlock()
						return
					}
					mu.Unlock()
				}
			}
		}()
	}

	// Update activeTasks tracking
	// We can close queue safely by managing lifecycle
	// Let's monitor queue completion
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				time.Sleep(100 * time.Millisecond)
				mu.Lock()
				count := 0
				visitedPages.Range(func(_, _ interface{}) bool {
					count++
					return true
				})
				if len(queue) == 0 || count >= opts.MaxPages {
					// Check if all workers are idle
					if len(workerSem) == 0 {
						select {
						case <-done:
						default:
							close(done)
						}
						mu.Unlock()
						return
					}
				}
				mu.Unlock()
			}
		}
	}()

	wg.Wait()

	var jsURLs []string
	discoveredJS.Range(func(key, _ interface{}) bool {
		jsURLs = append(jsURLs, key.(string))
		return true
	})

	return jsURLs, nil
}
