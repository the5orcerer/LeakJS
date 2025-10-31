package scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
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

func ReadPatterns(yamlFile string) ([]patterns.Pattern, error) {
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
		re, err := regexp.Compile(pc.Pattern.Regex)
		if err != nil {
			log.Printf("Invalid regex in pattern %s: %v", pc.Pattern.Name, err)
			continue
		}
		pattern := pc.Pattern
		pattern.Compiled = re
		pats = append(pats, pattern)
	}
	return pats, nil
}

func ParseDirectPatterns(patternsStr string) ([]patterns.Pattern, error) {
	var pats []patterns.Pattern
	for i, pattern := range strings.Split(patternsStr, ";") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("Invalid regex pattern: %s - Skipped", pattern)
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

func SearchPatterns(content string, pats []patterns.Pattern) map[string]map[string]interface{} {
	results := make(map[string]map[string]interface{})
	for _, pattern := range pats {
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
		for _, data := range results {
			matches := data["matches"].([]string)
			confidence := data["confidence"].(string)
			if len(matches) > 0 {
				// Truncate long matches for display
				match := matches[0]
				if len(match) > 50 {
					match = match[:47] + "..."
				}
				green.Printf("%s [%s] [%s]\n", source, match, confidence)
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

	for _, data := range results {
		matches := data["matches"].([]string)
		confidence := data["confidence"].(string)
		if len(matches) > 0 {
			fmt.Fprintf(file, "%s [%s] [%s]\n", source, matches[0], confidence)
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

func ProcessURL(url string, pats []patterns.Pattern, client *http.Client, outputFile string, verbose, silent, json bool, wg *sync.WaitGroup, mu *sync.Mutex, stats *ScanStats) {
	defer wg.Done()
	content, err := FetchURLContent(url, client)
	if err != nil {
		if verbose {
			log.Printf("Error fetching URL %s: %v", url, err)
		}
		return
	}
	results := SearchPatterns(content, pats)
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

func ProcessFile(filePath string, pats []patterns.Pattern, outputFile string, verbose, silent, json bool, wg *sync.WaitGroup, mu *sync.Mutex, stats *ScanStats) {
	defer wg.Done()
	content, err := ReadFileContent(filePath)
	if err != nil {
		if verbose {
			log.Printf("Error reading file %s: %v", filePath, err)
		}
		return
	}
	results := SearchPatterns(content, pats)
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

func RunLeakJS(urlsFile, singleURL, patternsFile, directPatterns, filePath, outputFile string, concurrency int, verbose, silent, json, showStats bool, stats *ScanStats) error {
	start := time.Now()
	defer func() {
		if showStats {
			stats.ScanDuration = time.Since(start)
			stats.Print()
		}
	}()

	pats := patterns.GetBuiltInPatterns()

	if patternsFile != "" {
		additionalPats, err := ReadPatterns(patternsFile)
		if err != nil {
			return err
		}
		pats = append(pats, additionalPats...)
	}

	if directPatterns != "" {
		additionalPats, err := ParseDirectPatterns(directPatterns)
		if err != nil {
			return err
		}
		pats = append(pats, additionalPats...)
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
					ProcessURL(src, pats, client, outputFile, verbose, silent, json, &wg, &mu, stats)
				} else {
					ProcessFile(src, pats, outputFile, verbose, silent, json, &wg, &mu, stats)
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
					ProcessURL(src, pats, client, outputFile, verbose, silent, json, &wg, &mu, stats)
				} else {
					ProcessFile(src, pats, outputFile, verbose, silent, json, &wg, &mu, stats)
				}
			}(source)
		}
	}

	wg.Wait()
	return nil
}
