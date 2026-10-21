package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/the5orcerer/LeakJS/internal/scanner"
)

func banner() {
	fmt.Println(`
                        __               __       _______
                       / /   ___  ____ _/ /__    / / ___/
                      / /   / _ \/ __ \` + "`" + ` //_/_  / /\__ \
                     / /___/  __/ /_/ / ,< / /_/ /___/ /
                    /_____/\___/\__,_/_/|_|\\____//____/
                                               @rootplinix
	`)
}

func main() {
	banner()

	var (
		url         string
		list        string
		patterns    string
		regex       string
		file        string
		dir         string
		all         string
		config      string
		exclude     string
		benchmark   int
		concurrency int
		output      string
		verbose     bool
		silent      bool
		json        bool
		stats       bool
		help        bool
		highOnly    bool
		cookies     string
		headers     []string
		proxy       string
		insecure    bool
		timeout     int
		delay       int
		userAgent   string
		crawl       bool
		crawlOnly   bool
		depth       int
		maxPages    int
		subdomains  bool
		endpoints   bool
		sourceMaps  bool
	)

	rootCmd := &cobra.Command{
		Use:   "leakjs",
		Short: "A fast, concurrent JavaScript secret leak detector and crawler",
		Run: func(cmd *cobra.Command, args []string) {
			if help {
				cmd.Help()
				return
			}

			scanStats := &scanner.ScanStats{}
			opts := scanner.ScanOptions{
				URLsFile:            list,
				SingleURL:           url,
				PatternsFile:        patterns,
				DirectPatterns:      regex,
				FilePath:            file,
				Dir:                 dir,
				AllDir:              all,
				ConfigFile:          config,
				ExcludePatterns:     exclude,
				OutputFile:          output,
				Concurrency:         concurrency,
				Verbose:             verbose,
				Silent:              silent,
				JSON:                json,
				Stats:               stats,
				BenchmarkIterations: benchmark,
				HighOnly:            highOnly,
				Cookies:             cookies,
				Headers:             headers,
				Proxy:               proxy,
				Insecure:            insecure,
				Timeout:             timeout,
				Delay:               delay,
				UserAgent:           userAgent,
				Crawl:               crawl,
				CrawlOnly:           crawlOnly,
				Depth:               depth,
				MaxPages:            maxPages,
				Subdomains:          subdomains,
				Endpoints:           endpoints,
				SourceMaps:          sourceMaps,
				ScanStats:           scanStats,
			}

			err := scanner.Run(opts)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	// Target options
	rootCmd.Flags().StringVarP(&url, "url", "u", "", "Single URL to scan or seed URL to crawl")
	rootCmd.Flags().StringVarP(&list, "list", "l", "", "Path to file containing URLs or crawl targets")
	rootCmd.Flags().StringVarP(&file, "file", "f", "", "Path to a local JavaScript file to scan")
	rootCmd.Flags().StringVarP(&dir, "dir", "d", "", "Recursively scan a local directory for JavaScript files")

	// Crawl options
	rootCmd.Flags().BoolVar(&crawl, "crawl", false, "Crawl website starting from seed URL(s) to discover all JS files")
	rootCmd.Flags().BoolVar(&crawlOnly, "crawl-only", false, "Only crawl and output discovered JS URLs without scanning for leaks")
	rootCmd.Flags().IntVar(&depth, "depth", 2, "Maximum crawling depth (default 2)")
	rootCmd.Flags().IntVar(&maxPages, "max-pages", 50, "Maximum number of pages to crawl per target (default 50)")
	rootCmd.Flags().BoolVar(&subdomains, "subdomains", false, "Include subdomains during crawling")

	// Authentication & HTTP options
	rootCmd.Flags().StringVar(&cookies, "cookie", "", "Cookie string attached to all requests (e.g. 'session=xyz; token=abc')")
	rootCmd.Flags().StringVar(&cookies, "cookies", "", "Alias for --cookie")
	rootCmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Custom HTTP header(s), can be used multiple times (e.g. -H 'Authorization: Bearer ...')")
	rootCmd.Flags().StringVar(&proxy, "proxy", "", "HTTP/HTTPS/SOCKS5 proxy (e.g. http://127.0.0.1:8080)")
	rootCmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "Skip TLS/SSL certificate verification")
	rootCmd.Flags().IntVar(&timeout, "timeout", 15, "HTTP request timeout in seconds (default 15)")
	rootCmd.Flags().IntVar(&delay, "delay", 0, "Delay between requests in milliseconds")
	rootCmd.Flags().StringVar(&userAgent, "user-agent", "", "Custom User-Agent header")

	// Pattern options
	rootCmd.Flags().StringVarP(&patterns, "patterns", "p", "", "Path to YAML pattern file (defaults to built-in patterns)")
	rootCmd.Flags().StringVarP(&regex, "regex", "r", "", "Direct regex patterns separated by ';'")
	rootCmd.Flags().StringVarP(&all, "all", "a", "", "Path to directory containing all regex YAML files to load")
	rootCmd.Flags().StringVarP(&config, "config", "c", "", "Path to configuration YAML file")
	rootCmd.Flags().StringVarP(&exclude, "exclude", "e", "", "Comma-separated list of pattern names to exclude")
	rootCmd.Flags().BoolVar(&highOnly, "high-only", false, "Show only high-confidence pattern matches")

	// Recon & extraction tools
	rootCmd.Flags().BoolVarP(&endpoints, "endpoints", "E", false, "Extract API endpoints, REST routes, and URLs from JS files")
	rootCmd.Flags().BoolVarP(&sourceMaps, "sourcemaps", "m", false, "Detect and verify exposed JavaScript source maps (.map)")

	// Execution & Output options
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "C", 20, "Number of concurrent requests")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Path to output file to save results")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Show progress bar without console output")
	rootCmd.Flags().BoolVarP(&json, "json", "j", false, "Output results in JSON format")
	rootCmd.Flags().BoolVarP(&stats, "stats", "t", false, "Show scan statistics at the end")
	rootCmd.Flags().IntVarP(&benchmark, "benchmark", "b", 0, "Run benchmark with specified number of iterations")
	rootCmd.Flags().BoolVar(&help, "help", false, "Show help message and exit")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
