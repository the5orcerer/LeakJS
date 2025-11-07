package scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"gopkg.in/yaml.v3"

	"github.com/the5orcerer/LeakJS/internal/patterns"
)

var (
	red    = color.New(color.FgRed)
	green  = color.New(color.FgGreen)
	blue   = color.New(color.FgBlue)
	yellow = color.New(color.FgYellow)
)

type ScanStats struct {
	FilesScanned int
	URLsScanned  int
	TotalMatches int
	PatternsUsed int
	ScanDuration time.Duration
}

type Config struct {
	Concurrency  int    `yaml:"concurrency"`
	Output       string `yaml:"output"`
	Verbose      bool   `yaml:"verbose"`
	Silent       bool   `yaml:"silent"`
	JSON         bool   `yaml:"json"`
	Stats        bool   `yaml:"stats"`
	PatternsDir  string `yaml:"patterns_dir"`
	PatternsFile string `yaml:"patterns_file"`
	Exclude      string `yaml:"exclude"`
	Include      string `yaml:"include"` // New field for pattern shortcuts
}

// FilterPatterns applies both include and exclude filters to the pattern list
func FilterPatterns(pats []patterns.Pattern, includeList, excludeList string) []patterns.Pattern {
	// If neither include nor exclude is specified, return all patterns
	if includeList == "" && excludeList == "" {
		return pats
	}

	// Create include and exclude maps
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
		// Skip if explicitly excluded
		if excluded[pattern.Name] {
			continue
		}

		// Include if either:
		// 1. No include list specified (include all)
		// 2. Pattern name is in the include list
		if includeList == "" || included[pattern.Name] {
			filtered = append(filtered, pattern)
		}
	}

	return filtered
}

func LoadConfig(configFile string) (*Config, error) {
	var configPath string

	if configFile != "" {
		// Use explicitly provided config file
		configPath = configFile
	} else {
		// Check default locations in order of preference
		homeDir, err := os.UserHomeDir()
		if err == nil {
			// Check ~/.config/leakjs/config.yaml
			candidatePath := filepath.Join(homeDir, ".config", "leakjs", "config.yaml")
			if _, err := os.Stat(candidatePath); err == nil {
				configPath = candidatePath
			} else {
				// Check ~/.leakjs/config.yaml (fallback)
				candidatePath = filepath.Join(homeDir, ".leakjs", "config.yaml")
				if _, err := os.Stat(candidatePath); err == nil {
					configPath = candidatePath
				}
			}
		}
	}

	// If no config file found, return default config
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
	fmt.Printf("Patterns used: %d\n", s.PatternsUsed)
	fmt.Printf("Scan duration: %v\n", s.ScanDuration)
}

func ReadURLs(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && strings.HasSuffix(line, ".js") {
			urls = append(urls, line)
		}
	}
	return urls, scanner.Err()
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

	// Read patterns from filesystem
	if _, err := os.Stat(dirPath); err != nil {
		return nil, fmt.Errorf("pattern directory not found: %s", dirPath)
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process YAML files
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".yaml") {
			pats, err := ReadPatterns(path, verbose)
			if err != nil {
				if verbose {
					log.Printf("Error reading patterns from %s: %v", path, err)
				}
				return nil // Continue with other files
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
		// Skip non-High confidence patterns if highOnly is enabled
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
				// Truncate long matches for display
				match := matches[0]
				if len(match) > 50 {
					match = match[:47] + "..."
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

func FetchURLContent(url string, client *http.Client) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

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

func ProcessURL(url string, pats []patterns.Pattern, client *http.Client, outputFile string, verbose, silent, json bool, wg *sync.WaitGroup, mu *sync.Mutex, stats *ScanStats, highOnly bool) {
	defer wg.Done()
	content, err := FetchURLContent(url, client)
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
}

func ProcessFile(filePath string, pats []patterns.Pattern, outputFile string, verbose, silent, json bool, wg *sync.WaitGroup, mu *sync.Mutex, stats *ScanStats, highOnly bool) {
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
}

func RunLeakJS(urlsFile, singleURL, patternsFile, directPatterns, filePath, allDir, configFile, excludePatterns, outputFile string, concurrency int, verbose, silent, json, showStats bool, benchmarkIterations int, highOnly bool, stats *ScanStats) error {
	if benchmarkIterations > 0 {
		return RunBenchmark(urlsFile, singleURL, patternsFile, directPatterns, filePath, allDir, configFile, excludePatterns, outputFile, concurrency, verbose, silent, json, benchmarkIterations, highOnly)
	}

	start := time.Now()
	defer func() {
		if showStats {
			stats.ScanDuration = time.Since(start)
			stats.Print()
		}
	}()

	// Load configuration file if provided
	config, err := LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("error loading config file: %v", err)
	}

	// Override config with command-line arguments if provided
	if concurrency == 20 && config.Concurrency > 0 {
		concurrency = config.Concurrency
	}
	if outputFile == "" && config.Output != "" {
		outputFile = config.Output
	}
	// Note: verbose is not overridden by config as it's a user display preference
	if !silent && config.Silent {
		silent = config.Silent
	}
	if !json && config.JSON {
		json = config.JSON
	}
	if !showStats && config.Stats {
		showStats = config.Stats
	}
	if allDir == "" && config.PatternsDir != "" {
		allDir = config.PatternsDir
	}
	if patternsFile == "" && config.PatternsFile != "" {
		patternsFile = config.PatternsFile
	}
	if excludePatterns == "" && config.Exclude != "" {
		excludePatterns = config.Exclude
	}

	var pats []patterns.Pattern

	// Handle pattern loading
	if allDir != "" {
		// Load all patterns from directory
		dirPats, err := ReadAllPatternsFromDir(allDir, verbose)
		if err != nil {
			return fmt.Errorf("error loading patterns from directory %s: %v", allDir, err)
		}
		pats = dirPats
	} else if patternsFile != "" {
		// Use only patterns from the specified file
		filePats, err := ReadPatterns(patternsFile, verbose)
		if err != nil {
			return err
		}
		pats = filePats
	} else if directPatterns != "" {
		// Use direct patterns if provided
		directPats, err := ParseDirectPatterns(directPatterns, verbose)
		if err != nil {
			return err
		}
		pats = directPats
	} else {
		// No patterns specified
		return fmt.Errorf("no patterns specified. Use -p to specify a pattern file, --all-dir for a directory of patterns, or provide direct patterns")
	}

	if directPatterns != "" {
		additionalPats, err := ParseDirectPatterns(directPatterns, verbose)
		if err != nil {
			return err
		}
		pats = append(pats, additionalPats...)
	}

	// Filter out excluded patterns
	if excludePatterns != "" {
		pats = FilterPatterns(pats, config.Include, excludePatterns)
	}

	stats.PatternsUsed = len(pats)
	blue.Printf("[INF] Templates loaded: %d\n\n", len(pats))

	var sources []string
	if urlsFile != "" {
		urls, err := ReadURLs(urlsFile)
		if err != nil {
			return err
		}
		sources = urls
		stats.URLsScanned = len(urls)
	} else if singleURL != "" {
		sources = []string{singleURL}
		stats.URLsScanned = 1
	} else if filePath != "" {
		sources = []string{filePath}
		stats.FilesScanned = 1
	} else {
		// Read from stdin
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && strings.HasSuffix(line, ".js") {
				sources = append(sources, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		// Assume all from stdin are URLs for simplicity
		stats.URLsScanned = len(sources)
	}

	if len(sources) == 0 {
		red.Println("[ERR] No URLs or files provided.")
		return nil
	}

	if silent && outputFile == "" {
		red.Println("[ERR] Output file must be specified in silent mode (-s)")
		return nil
	}

	client := &http.Client{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, concurrency)

	if silent {
		bar := progressbar.Default(int64(len(sources)))
		for _, source := range sources {
			wg.Add(1)
			go func(src string) {
				sem <- struct{}{}
				defer func() { <-sem }()
				if strings.HasPrefix(src, "http") {
					ProcessURL(src, pats, client, outputFile, verbose, silent, json, &wg, &mu, stats, highOnly)
				} else {
					ProcessFile(src, pats, outputFile, verbose, silent, json, &wg, &mu, stats, highOnly)
				}
				bar.Add(1)
			}(source)
		}
	} else {
		for _, source := range sources {
			wg.Add(1)
			go func(src string) {
				sem <- struct{}{}
				defer func() { <-sem }()
				if strings.HasPrefix(src, "http") {
					ProcessURL(src, pats, client, outputFile, verbose, silent, json, &wg, &mu, stats, highOnly)
				} else {
					ProcessFile(src, pats, outputFile, verbose, silent, json, &wg, &mu, stats, highOnly)
				}
			}(source)
		}
	}

	wg.Wait()
	return nil
}

func RunBenchmark(urlsFile, singleURL, patternsFile, directPatterns, filePath, allDir, configFile, excludePatterns, outputFile string, concurrency int, verbose, silent, json bool, benchmarkIterations int, highOnly bool) error {
	blue.Printf("[INF] Running benchmark with %d iterations...\n", benchmarkIterations)

	var totalDuration time.Duration
	var totalMatches int

	for i := 0; i < benchmarkIterations; i++ {
		yellow.Printf("[BENCH] Iteration %d/%d\n", i+1, benchmarkIterations)

		stats := &ScanStats{}
		start := time.Now()

		err := RunLeakJS(urlsFile, singleURL, patternsFile, directPatterns, filePath, allDir, configFile, excludePatterns, outputFile, concurrency, verbose, true, json, false, 0, highOnly, stats)
		if err != nil {
			return fmt.Errorf("benchmark iteration %d failed: %v", i+1, err)
		}

		duration := time.Since(start)
		totalDuration += duration
		totalMatches += stats.TotalMatches

		fmt.Printf("Iteration %d: %v, %d matches\n", i+1, duration, stats.TotalMatches)
	}

	avgDuration := totalDuration / time.Duration(benchmarkIterations)
	avgMatches := totalMatches / benchmarkIterations

	blue.Printf("\n[BENCH] Benchmark completed!\n")
	fmt.Printf("Average scan time: %v\n", avgDuration)
	fmt.Printf("Average matches per iteration: %d\n", avgMatches)
	fmt.Printf("Total benchmark time: %v\n", totalDuration)

	return nil
}
