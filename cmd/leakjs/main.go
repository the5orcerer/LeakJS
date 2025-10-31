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
	)

	rootCmd := &cobra.Command{
		Use:   "leakjs",
		Short: "A JavaScript leak detection tool",
		Run: func(cmd *cobra.Command, args []string) {
			if help {
				cmd.Help()
				return
			}

			scanStats := &scanner.ScanStats{}
			err := scanner.RunLeakJS(list, url, patterns, regex, file, all, config, exclude, output, concurrency, verbose, silent, json, stats, benchmark, scanStats)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	rootCmd.Flags().StringVarP(&url, "url", "u", "", "Single URL to scan")
	rootCmd.Flags().StringVarP(&list, "list", "l", "", "Path to the file containing URLs")
	rootCmd.Flags().StringVarP(&patterns, "patterns", "p", "", "Path to the YAML file containing patterns")
	rootCmd.Flags().StringVarP(&regex, "regex", "r", "", "Directly input regex patterns separated by ';'")
	rootCmd.Flags().StringVarP(&file, "file", "f", "", "Path to a JavaScript file to scan")
	rootCmd.Flags().StringVarP(&all, "all", "a", "", "Path to directory containing all regex YAML files to load")
	rootCmd.Flags().StringVarP(&config, "config", "c", "", "Path to configuration YAML file")
	rootCmd.Flags().StringVarP(&exclude, "exclude", "e", "", "Comma-separated list of pattern names to exclude")
	rootCmd.Flags().IntVarP(&benchmark, "benchmark", "b", 0, "Run benchmark with specified number of iterations")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "C", 1, "Number of concurrent requests")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Path to the output file to save results")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Show progress bar without any output in the terminal")
	rootCmd.Flags().BoolVarP(&json, "json", "j", false, "Output results in JSON format")
	rootCmd.Flags().BoolVarP(&stats, "stats", "t", false, "Show scan statistics at the end")
	rootCmd.Flags().BoolVar(&help, "help", false, "Show help message and exit")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
