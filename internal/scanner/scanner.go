package scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"gopkg.in/yaml.v3"

	"github.com/the5orcerer/LeakJS/internal/crawler"
	"github.com/the5orcerer/LeakJS/internal/extractor"
	"github.com/the5orcerer/LeakJS/internal/patterns"
)

var (
	red    = color.New(color.FgRed)
	green  = color.New(color.FgGreen)
	blue   = color.New(color.FgBlue)
	yellow = color.New(color.FgYellow)
)

type ScanStats struct {
	FilesScanned    int
	URLsScanned     int
	TotalMatches    int
	PatternsUsed    int
	EndpointsFound  int
	SourceMapsFound int
	ScanDuration    time.Duration
}

type Config struct {
	Concurrency  int      `yaml:"concurrency"`
	Output       string   `yaml:"output"`
	Verbose      bool     `yaml:"verbose"`
	Silent       bool     `yaml:"silent"`
	JSON         bool     `yaml:"json"`
	Stats        bool     `yaml:"stats"`
	PatternsDir  string   `yaml:"patterns_dir"`
	PatternsFile string   `yaml:"patterns_file"`
	Exclude      string   `yaml:"exclude"`
	Include      string   `yaml:"include"`
	Cookies      string   `yaml:"cookies"`
	Headers      []string `yaml:"headers"`
	UserAgent    string   `yaml:"user_agent"`
	Proxy        string   `yaml:"proxy"`
	Insecure     bool     `yaml:"insecure"`
	Timeout      int      `yaml:"timeout"`
	Delay        int      `yaml:"delay"`
	Depth        int      `yaml:"depth"`
	MaxPages     int      `yaml:"max_pages"`
	Subdomains   bool     `yaml:"subdomains"`
	Endpoints    bool     `yaml:"endpoints"`
	SourceMaps   bool     `yaml:"sourcemaps"`
}

type ScanOptions struct {
	URLsFile             string
	SingleURL            string
	PatternsFile         string
	DirectPatterns       string
	FilePath             string
	Dir                  string
	AllDir               string
	ConfigFile           string
	ExcludePatterns      string
	OutputFile           string
	Concurrency          int
	Verbose              bool
	Silent               bool
	JSON                 bool
	Stats                bool
	BenchmarkIterations  int
	HighOnly             bool
	Cookies              string
	Headers              []string
	Proxy                string
	Insecure             bool
	Timeout              int
	Delay                int
	UserAgent            string
	Crawl                bool
	CrawlOnly            bool
	Depth                int
	MaxPages             int
	Subdomains           bool
	Endpoints            bool
	SourceMaps           bool
	ScanStats            *ScanStats
}

// FilterPatterns applies both include and exclude filters to the pattern list
func FilterPatterns(pats []patterns.Pattern, includeList, excludeList string) []patterns.Pattern {
	if includeList == "" && excludeList == "" {
		return pats
	}

	included := make(map[string]bool)
	if includeList != "" {
		for _, name := range strings.Split(includeList, ",") {
			included[strings.TrimSpace(name)] = true
		}
	}

	excluded := make(map[string]bool)
	if excludeList != "" {
		for _, name := range strings.Split(excludeList, ",") {
			excluded[strings.TrimSpace(name)] = true
		}
	}

	var filtered []patterns.Pattern
	for _, pattern := range pats {
		if excluded[pattern.Name] {
			continue
		}
		if includeList == "" || included[pattern.Name] {
			filtered = append(filtered, pattern)
		}
	}

	return filtered
}

func LoadConfig(configFile string) (*Config, error) {
	var configPath string

	if configFile != "" {
		configPath = configFile
	} else {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			candidatePath := filepath.Join(homeDir, ".config", "leakjs", "config.yaml")
			if _, err := os.Stat(candidatePath); err == nil {
				configPath = candidatePath
			} else {
				candidatePath = filepath.Join(homeDir, ".leakjs", "config.yaml")
				if _, err := os.Stat(candidatePath); err == nil {
					configPath = candidatePath
				}
			}
		}
	}

	if configPath == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (s *ScanStats) Print() {
	blue.Printf("\n[INF] Scan completed!\n")
	fmt.Printf("Files scanned: %d\n", s.FilesScanned)
	fmt.Printf("URLs scanned: %d\n", s.URLsScanned)
	fmt.Printf("Total matches found: %d\n", s.TotalMatches)
	if s.EndpointsFound > 0 {
		fmt.Printf("Endpoints found: %d\n", s.EndpointsFound)
	}
	if s.SourceMapsFound > 0 {
		fmt.Printf("Source maps found: %d\n", s.SourceMapsFound)
	}
	fmt.Printf("Patterns used: %d\n", s.PatternsUsed)
	fmt.Printf("Scan duration: %v\n", s.ScanDuration)
}

func ReadURLs(filePath string, allowAllURLs bool) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if allowAllURLs {
			urls = append(urls, line)
		} else {
			// Check if URL or path indicates a JS file (even with query params)
			u, err := url.Parse(line)
			if err == nil && (strings.HasSuffix(strings.ToLower(u.Path), ".js") || strings.Contains(strings.ToLower(line), ".js")) {
				urls = append(urls, line)
			} else if strings.HasSuffix(strings.ToLower(line), ".js") {
				urls = append(urls, line)
			}
		}
	}
	return urls, scanner.Err()
}

func FindJSFilesInDir(dirPath string) ([]string, error) {
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".js") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func ReadPatterns(yamlFile string, verbose bool) ([]patterns.Pattern, error) {
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, err
	}

	var config patterns.Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	var pats []patterns.Pattern
	for _, pc := range config.Patterns {
		re, err := patterns.GetCompiledPattern(pc.Pattern.Regex)
		if err != nil {
			if verbose {
				log.Printf("Invalid regex in pattern %s: %v", pc.Pattern.Name, err)
			}
			continue
		}
		pattern := pc.Pattern
		pattern.Compiled = re
		pats = append(pats, pattern)
	}
	return pats, nil
}

func ReadAllPatternsFromDir(dirPath string, verbose bool) ([]patterns.Pattern, error) {
	var allPats []patterns.Pattern

	if _, err := os.Stat(dirPath); err != nil {
		return nil, fmt.Errorf("pattern directory not found: %s", dirPath)
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml")) {
			pats, err := ReadPatterns(path, verbose)
			if err != nil {
				if verbose {
					log.Printf("Error reading patterns from %s: %v", path, err)
				}
				return nil
			}
			allPats = append(allPats, pats...)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error reading patterns from directory: %v", err)
	}

	if len(allPats) == 0 {
		return nil, fmt.Errorf("no pattern files found in directory: %s", dirPath)
	}

	return allPats, nil
}

func ParseDirectPatterns(patternsStr string, verbose bool) ([]patterns.Pattern, error) {
	var pats []patterns.Pattern
	for i, pattern := range strings.Split(patternsStr, ";") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		re, err := patterns.GetCompiledPattern(pattern)
		if err != nil {
			if verbose {
				log.Printf("Invalid regex pattern: %s - Skipped", pattern)
			}
			continue
		}
		pats = append(pats, patterns.Pattern{
			Name:       fmt.Sprintf("DirectPattern%d", i+1),
			Regex:      pattern,
			Confidence: "Unknown",
			Compiled:   re,
		})
	}
	return pats, nil
}

func ReadFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func SearchPatterns(content string, pats []patterns.Pattern, highOnly bool) map[string]map[string]interface{} {
	results := make(map[string]map[string]interface{})
	for _, pattern := range pats {
		if highOnly && pattern.Confidence != "High" {
			continue
		}
		matches := pattern.Compiled.FindAllString(content, -1)
		if len(matches) > 0 {
			results[pattern.Name] = map[string]interface{}{
				"matches":    matches,
				"confidence": pattern.Confidence,
			}
		}
	}
	return results
}

func DisplayResults(source string, results map[string]map[string]interface{}, sourceType string, verbose, json bool) {
	if json {
		DisplayResultsJSON(results)
		return
	}
	if len(results) > 0 {
		for patternName, data := range results {
			matches := data["matches"].([]string)
			confidence := data["confidence"].(string)
			if len(matches) > 0 {
				match := matches[0]
				if len(match) > 60 {
					match = match[:57] + "..."
				}
				green.Printf("%s [%s] [%s] [%s]\n", source, match, confidence, patternName)
			}
		}
	}
}

func DisplayResultsJSON(results map[string]map[string]interface{}) {
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}
	fmt.Println(string(jsonData))
}

func SaveResultsToFile(outputFile, source string, results map[string]map[string]interface{}, sourceType string) error {
	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for patternName, data := range results {
		matches := data["matches"].([]string)
		confidence := data["confidence"].(string)
		if len(matches) > 0 {
			fmt.Fprintf(file, "%s [%s] [%s] [%s]\n", source, matches[0], confidence, patternName)
		}
	}
	return nil
}

func DisplayEndpoints(source string, endpoints []string, isJSON bool) {
	if isJSON {
		data := map[string]interface{}{
			"source":    source,
			"endpoints": endpoints,
		}
		if b, err := json.MarshalIndent(data, "", "  "); err == nil {
			fmt.Println(string(b))
		}
		return
	}
	for _, ep := range endpoints {
		blue.Printf("[ENDPOINT] %s -> %s\n", source, ep)
	}
}

func SaveEndpointsToFile(outputFile, source string, endpoints []string) error {
	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, ep := range endpoints {
		fmt.Fprintf(file, "[ENDPOINT] %s -> %s\n", source, ep)
	}
	return nil
}

func DisplaySourceMap(source, mapURL string, isJSON bool) {
	if isJSON {
		data := map[string]interface{}{
			"source":    source,
			"sourcemap": mapURL,
		}
		if b, err := json.MarshalIndent(data, "", "  "); err == nil {
			fmt.Println(string(b))
		}
		return
	}
	yellow.Printf("[MAP] %s -> Exposed source map: %s\n", source, mapURL)
}

func SaveSourceMapToFile(outputFile, source, mapURL string) error {
	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "[MAP] %s -> Exposed source map: %s\n", source, mapURL)
	return nil
}

func FetchURLContent(url string, client *http.Client, cookies string, headers []string, userAgent string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	crawler.ApplyHeaders(req, cookies, headers, userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func ProcessURL(url string, pats []patterns.Pattern, client *http.Client, cookies string, headers []string, userAgent string, outputFile string, verbose, silent, json bool, wg *sync.WaitGroup, mu *sync.Mutex, stats *ScanStats, highOnly, findEndpoints, findSourceMaps bool) {
	defer wg.Done()
	content, err := FetchURLContent(url, client, cookies, headers, userAgent)
	if err != nil {
		if verbose {
			log.Printf("Error fetching URL %s: %v", url, err)
		}
		return
	}

	results := SearchPatterns(content, pats, highOnly)
	if len(results) > 0 {
		mu.Lock()
		stats.TotalMatches += len(results)
		if outputFile != "" {
			SaveResultsToFile(outputFile, url, results, "URL")
		}
		if !silent {
			DisplayResults(url, results, "URL", verbose, json)
		}
		mu.Unlock()
	}

	if findEndpoints {
		endpoints := extractor.ExtractEndpoints(content)
		if len(endpoints) > 0 {
			mu.Lock()
			stats.EndpointsFound += len(endpoints)
			if outputFile != "" {
				SaveEndpointsToFile(outputFile, url, endpoints)
			}
			if !silent {
				DisplayEndpoints(url, endpoints, json)
			}
			mu.Unlock()
		}
	}

	if findSourceMaps {
		if mapURL, leaked := extractor.CheckSourceMap(url, content, client, cookies, headers); leaked {
			mu.Lock()
			stats.SourceMapsFound++
			if outputFile != "" {
				SaveSourceMapToFile(outputFile, url, mapURL)
			}
			if !silent {
				DisplaySourceMap(url, mapURL, json)
			}
			mu.Unlock()
		}
	}
}

func ProcessFile(filePath string, pats []patterns.Pattern, outputFile string, verbose, silent, json bool, wg *sync.WaitGroup, mu *sync.Mutex, stats *ScanStats, highOnly, findEndpoints bool) {
	defer wg.Done()
	content, err := ReadFileContent(filePath)
	if err != nil {
		if verbose {
			log.Printf("Error reading file %s: %v", filePath, err)
		}
		return
	}
	results := SearchPatterns(content, pats, highOnly)
	if len(results) > 0 {
		mu.Lock()
		stats.TotalMatches += len(results)
		if outputFile != "" {
			SaveResultsToFile(outputFile, filePath, results, "File")
		}
		if !silent {
			DisplayResults(filePath, results, "File", verbose, json)
		}
		mu.Unlock()
	}

	if findEndpoints {
		endpoints := extractor.ExtractEndpoints(content)
		if len(endpoints) > 0 {
			mu.Lock()
			stats.EndpointsFound += len(endpoints)
			if outputFile != "" {
				SaveEndpointsToFile(outputFile, filePath, endpoints)
			}
			if !silent {
				DisplayEndpoints(filePath, endpoints, json)
			}
			mu.Unlock()
		}
	}
}

// Run executes a scan with the given ScanOptions
func Run(opts ScanOptions) error {
	if opts.BenchmarkIterations > 0 {
		return RunBenchmark(opts)
	}

	start := time.Now()
	defer func() {
		if opts.Stats && opts.ScanStats != nil {
			opts.ScanStats.ScanDuration = time.Since(start)
			opts.ScanStats.Print()
		}
	}()

	// Load configuration file if provided or discovered
	config, err := LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("error loading config file: %v", err)
	}

	// Apply config overrides if command-line flags were left default
	if opts.Concurrency <= 0 {
		if config.Concurrency > 0 {
			opts.Concurrency = config.Concurrency
		} else {
			opts.Concurrency = 20
		}
	}
	if opts.OutputFile == "" && config.Output != "" {
		opts.OutputFile = config.Output
	}
	if !opts.Silent && config.Silent {
		opts.Silent = config.Silent
	}
	if !opts.JSON && config.JSON {
		opts.JSON = config.JSON
	}
	if !opts.Stats && config.Stats {
		opts.Stats = config.Stats
	}
	if opts.AllDir == "" && config.PatternsDir != "" {
		opts.AllDir = config.PatternsDir
	}
	if opts.PatternsFile == "" && config.PatternsFile != "" {
		opts.PatternsFile = config.PatternsFile
	}
	if opts.ExcludePatterns == "" && config.Exclude != "" {
		opts.ExcludePatterns = config.Exclude
	}
	if opts.Cookies == "" && config.Cookies != "" {
		opts.Cookies = config.Cookies
	}
	if len(opts.Headers) == 0 && len(config.Headers) > 0 {
		opts.Headers = config.Headers
	}
	if opts.UserAgent == "" && config.UserAgent != "" {
		opts.UserAgent = config.UserAgent
	}
	if opts.Proxy == "" && config.Proxy != "" {
		opts.Proxy = config.Proxy
	}
	if !opts.Insecure && config.Insecure {
		opts.Insecure = config.Insecure
	}
	if opts.Timeout == 0 && config.Timeout > 0 {
		opts.Timeout = config.Timeout
	}
	if opts.Delay == 0 && config.Delay > 0 {
		opts.Delay = config.Delay
	}
	if opts.Depth == 0 {
		if config.Depth > 0 {
			opts.Depth = config.Depth
		} else {
			opts.Depth = 2
		}
	}
	if opts.MaxPages == 0 {
		if config.MaxPages > 0 {
			opts.MaxPages = config.MaxPages
		} else {
			opts.MaxPages = 50
		}
	}
	if !opts.Subdomains && config.Subdomains {
		opts.Subdomains = config.Subdomains
	}
	if !opts.Endpoints && config.Endpoints {
		opts.Endpoints = config.Endpoints
	}
	if !opts.SourceMaps && config.SourceMaps {
		opts.SourceMaps = config.SourceMaps
	}

	timeout := time.Duration(opts.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client, err := crawler.BuildHTTPClient(timeout, opts.Proxy, opts.Insecure)
	if err != nil {
		return fmt.Errorf("error setting up HTTP client: %w", err)
	}

	// Handle Crawl Mode
	var crawledJS []string
	if opts.Crawl {
		var seedURLs []string
		if opts.SingleURL != "" {
			seedURLs = append(seedURLs, opts.SingleURL)
		}
		if opts.URLsFile != "" {
			targets, err := ReadURLs(opts.URLsFile, true)
			if err != nil {
				return err
			}
			seedURLs = append(seedURLs, targets...)
		}

		if len(seedURLs) == 0 {
			return fmt.Errorf("crawl mode requires a URL (-u) or URL list (-l)")
		}

		crawlOpts := crawler.Options{
			Depth:       opts.Depth,
			MaxPages:    opts.MaxPages,
			Concurrency: opts.Concurrency,
			Delay:       time.Duration(opts.Delay) * time.Millisecond,
			Subdomains:  opts.Subdomains,
			Client:      client,
			Headers:     opts.Headers,
			Cookies:     opts.Cookies,
			UserAgent:   opts.UserAgent,
			Verbose:     opts.Verbose,
			Silent:      opts.Silent,
		}

		discoveredMap := make(map[string]bool)
		for _, seed := range seedURLs {
			if !opts.Silent {
				blue.Printf("[INF] Crawling %s (depth=%d, max_pages=%d)...\n", seed, opts.Depth, opts.MaxPages)
			}
			jsList, err := crawler.Crawl(seed, crawlOpts)
			if err != nil {
				if opts.Verbose {
					log.Printf("Error crawling %s: %v", seed, err)
				}
				continue
			}
			for _, js := range jsList {
				if !discoveredMap[js] {
					discoveredMap[js] = true
					crawledJS = append(crawledJS, js)
				}
			}
		}

		if !opts.Silent {
			green.Printf("[INF] Crawl completed: gathered %d unique JavaScript URLs.\n\n", len(crawledJS))
		}

		if opts.CrawlOnly {
			if opts.OutputFile != "" {
				file, err := os.Create(opts.OutputFile)
				if err != nil {
					return fmt.Errorf("error creating output file: %w", err)
				}
				defer file.Close()
				for _, jsURL := range crawledJS {
					fmt.Fprintln(file, jsURL)
				}
				if !opts.Silent {
					blue.Printf("[INF] Gathered %d JS URLs saved to %s\n", len(crawledJS), opts.OutputFile)
				}
			}
			if !opts.Silent || opts.OutputFile == "" {
				for _, jsURL := range crawledJS {
					fmt.Println(jsURL)
				}
			}
			return nil
		}
	}

	// Handle Pattern Loading
	var pats []patterns.Pattern
	if opts.AllDir != "" {
		dirPats, err := ReadAllPatternsFromDir(opts.AllDir, opts.Verbose)
		if err != nil {
			return fmt.Errorf("error loading patterns from directory %s: %v", opts.AllDir, err)
		}
		pats = dirPats
	} else if opts.PatternsFile != "" {
		filePats, err := ReadPatterns(opts.PatternsFile, opts.Verbose)
		if err != nil {
			return err
		}
		pats = filePats
	} else if opts.DirectPatterns != "" {
		directPats, err := ParseDirectPatterns(opts.DirectPatterns, opts.Verbose)
		if err != nil {
			return err
		}
		pats = directPats
	} else {
		// Auto-discovery fallback:
		// 1. Check user patterns directory ~/.config/leakjs/patterns
		userPats, _ := patterns.LoadUserPatterns(opts.Verbose)
		if len(userPats) > 0 {
			pats = userPats
		} else {
			// 2. Fall back to embedded comprehensive patterns
			embeddedPats, err := patterns.LoadDefaultPatterns(opts.Verbose)
			if err != nil {
				return fmt.Errorf("no patterns specified and failed to load embedded patterns: %v", err)
			}
			pats = embeddedPats
		}
	}

	// Filter excluded/included patterns
	if opts.ExcludePatterns != "" || config.Include != "" {
		pats = FilterPatterns(pats, config.Include, opts.ExcludePatterns)
	}

	if opts.ScanStats != nil {
		opts.ScanStats.PatternsUsed = len(pats)
	}
	if !opts.Silent {
		blue.Printf("[INF] Templates loaded: %d\n\n", len(pats))
	}

	// Gather sources to scan
	var sources []string
	if opts.Crawl {
		sources = append(sources, crawledJS...)
		if opts.ScanStats != nil {
			opts.ScanStats.URLsScanned += len(crawledJS)
		}
	} else if opts.URLsFile != "" {
		urls, err := ReadURLs(opts.URLsFile, false)
		if err != nil {
			return err
		}
		sources = urls
		if opts.ScanStats != nil {
			opts.ScanStats.URLsScanned = len(urls)
		}
	} else if opts.SingleURL != "" {
		sources = []string{opts.SingleURL}
		if opts.ScanStats != nil {
			opts.ScanStats.URLsScanned = 1
		}
	} else if opts.FilePath != "" {
		sources = []string{opts.FilePath}
		if opts.ScanStats != nil {
			opts.ScanStats.FilesScanned = 1
		}
	} else if opts.Dir != "" {
		files, err := FindJSFilesInDir(opts.Dir)
		if err != nil {
			return fmt.Errorf("error reading directory %s: %w", opts.Dir, err)
		}
		sources = files
		if opts.ScanStats != nil {
			opts.ScanStats.FilesScanned = len(files)
		}
	} else {
		// Read from stdin
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				sources = append(sources, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		if opts.ScanStats != nil {
			opts.ScanStats.URLsScanned = len(sources)
		}
	}

	if len(sources) == 0 {
		red.Println("[ERR] No URLs or files provided.")
		return nil
	}

	if opts.Silent && opts.OutputFile == "" {
		red.Println("[ERR] Output file must be specified in silent mode (-s)")
		return nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, opts.Concurrency)

	delay := time.Duration(opts.Delay) * time.Millisecond

	scanWorker := func(src string) {
		sem <- struct{}{}
		defer func() { <-sem }()
		if delay > 0 {
			time.Sleep(delay)
		}
		if strings.HasPrefix(src, "http") {
			ProcessURL(src, pats, client, opts.Cookies, opts.Headers, opts.UserAgent, opts.OutputFile, opts.Verbose, opts.Silent, opts.JSON, &wg, &mu, opts.ScanStats, opts.HighOnly, opts.Endpoints, opts.SourceMaps)
		} else {
			ProcessFile(src, pats, opts.OutputFile, opts.Verbose, opts.Silent, opts.JSON, &wg, &mu, opts.ScanStats, opts.HighOnly, opts.Endpoints)
		}
	}

	if opts.Silent {
		bar := progressbar.Default(int64(len(sources)))
		for _, source := range sources {
			wg.Add(1)
			go func(src string) {
				scanWorker(src)
				bar.Add(1)
			}(source)
		}
	} else {
		for _, source := range sources {
			wg.Add(1)
			go func(src string) {
				scanWorker(src)
			}(source)
		}
	}

	wg.Wait()
	return nil
}

// RunLeakJS provides backwards compatibility for existing callers
func RunLeakJS(urlsFile, singleURL, patternsFile, directPatterns, filePath, allDir, configFile, excludePatterns, outputFile string, concurrency int, verbose, silent, json, showStats bool, benchmarkIterations int, highOnly bool, stats *ScanStats) error {
	opts := ScanOptions{
		URLsFile:            urlsFile,
		SingleURL:           singleURL,
		PatternsFile:        patternsFile,
		DirectPatterns:      directPatterns,
		FilePath:            filePath,
		AllDir:              allDir,
		ConfigFile:          configFile,
		ExcludePatterns:     excludePatterns,
		OutputFile:          outputFile,
		Concurrency:         concurrency,
		Verbose:             verbose,
		Silent:              silent,
		JSON:                json,
		Stats:               showStats,
		BenchmarkIterations: benchmarkIterations,
		HighOnly:            highOnly,
		ScanStats:           stats,
	}
	return Run(opts)
}

func RunBenchmark(opts ScanOptions) error {
	blue.Printf("[INF] Running benchmark with %d iterations...\n", opts.BenchmarkIterations)

	var totalDuration time.Duration
	var totalMatches int

	iters := opts.BenchmarkIterations
	opts.BenchmarkIterations = 0 // prevent recursion
	opts.Silent = true

	for i := 0; i < iters; i++ {
		yellow.Printf("[BENCH] Iteration %d/%d\n", i+1, iters)

		iterStats := &ScanStats{}
		opts.ScanStats = iterStats
		start := time.Now()

		err := Run(opts)
		if err != nil {
			return fmt.Errorf("benchmark iteration %d failed: %v", i+1, err)
		}

		duration := time.Since(start)
		totalDuration += duration
		totalMatches += iterStats.TotalMatches

		fmt.Printf("Iteration %d: %v, %d matches\n", i+1, duration, iterStats.TotalMatches)
	}

	avgDuration := totalDuration / time.Duration(iters)
	avgMatches := totalMatches / iters

	blue.Printf("\n[BENCH] Benchmark completed!\n")
	fmt.Printf("Average scan time: %v\n", avgDuration)
	fmt.Printf("Average matches per iteration: %d\n", avgMatches)
	fmt.Printf("Total benchmark time: %v\n", totalDuration)

	return nil
}
