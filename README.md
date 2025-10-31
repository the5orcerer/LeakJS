
# LeakJS

![Go](https://img.shields.io/badge/go-1.18%2B-blue) ![Issues](https://img.shields.io/github/issues/the5orcerer/LeakJS) ![Forks](https://img.shields.io/github/forks/the5orcerer/LeakJS) ![Stars](https://img.shields.io/github/stars/the5orcerer/LeakJS) ![License](https://img.shields.io/github/license/the5orcerer/LeakJS)

## Getting Started

**LeakJS** is a robust tool designed to scan JavaScript files and URLs for predefined patterns, aiding in the identification of potential leaks and vulnerabilities. Leveraging Go's goroutines, it ensures fast and efficient scanning. Users can define custom patterns in YAML files for tailored scans. Detailed logging and comprehensive configuration options make LeakJS a versatile solution for securing JavaScript code.

## 🚀 Key Features

- **Concurrent Scanning**: Fast and efficient scanning using Go goroutines.
- **Custom Patterns**: Define your own patterns in YAML files.
- **High Performance**: Pre-compiled regexes and optimized concurrency.
- **Multiple Output Formats**: Text and JSON output options.
- **Detailed Logging**: Comprehensive logging for debugging and analysis.
- **Configurable**: Easily configure headers, patterns, and more.
- **Portable & Self-Contained**: Embedded regex patterns and automatic config discovery.
- **Cross-Platform**: Automated releases for Linux, macOS, and Windows.

## 🔄 Portability & Self-Contained Design

LeakJS is designed to be completely portable and self-contained, requiring no external files or dependencies to function.

### Embedded Regex Patterns
- **Zero External Dependencies**: All regex patterns are embedded directly in the binary at compile time
- **Automatic Fallback**: If custom pattern directories are not found, LeakJS automatically uses embedded patterns
- **Always Available**: Over 500+ built-in patterns covering APIs, tokens, keys, and sensitive data
- **Customizable**: Still supports loading additional patterns from external YAML files when available

### Automatic Configuration Discovery
- **Standard Locations**: Automatically discovers config files in `~/.config/leakjs/config.yaml` or `~/.leakjs/config.yaml`
- **No Config Required**: Works out-of-the-box with sensible defaults
- **Override Support**: Command-line flags still take precedence over config files

### Portable Binary
- **Single Executable**: Move the binary anywhere - it works without needing the source directory
- **Cross-Platform Releases**: Automated GitHub releases provide binaries for all major platforms
- **No Installation Required**: Download and run immediately
## 📥 Installation

### Option 1: Download Pre-built Binaries (Recommended)

Visit the [Releases](https://github.com/the5orcerer/LeakJS/releases) page and download the appropriate binary for your platform:

- **Linux**: `leakjs-linux-amd64` or `leakjs-linux-arm64`
- **macOS**: `leakjs-macos-amd64` or `leakjs-macos-arm64`
- **Windows**: `leakjs-windows-amd64.exe` or `leakjs-windows-arm64.exe`

Make the binary executable and move it to your PATH:
```bash
chmod +x leakjs-linux-amd64
sudo mv leakjs-linux-amd64 /usr/local/bin/leakjs
```

### Option 2: Build from Source

To install LeakJS, follow these steps:

```bash
git clone https://github.com/the5orcerer/LeakJS
cd LeakJS
make build
sudo make install
leakjs -h (For testing purpose)
```

Or using Go directly:

```bash
go build -o leakjs main.go
sudo mv leakjs /usr/local/bin/
```

## 🛠️ Usage

LeakJS can be used as a command-line tool to scan JavaScript files and URLs for patterns.

### Scan a Single URL

```bash
leakjs -u "https://example.com/script.js" -p patterns.yaml -c 5 -o results.txt

[ INF ] Current version v1.2.4 [updated]
[ INF ] Templates loaded: 6

[ SUCC ] https://example.com/index.js (URL)

Regex name: Stripe Secret Token
let google_map_api = "stripe:7b1a32f7h8377c0a"
Confidence: Critical
```

### Scan Multiple URLs from a File

```bash
leakjs -l urls.txt -p patterns.yaml -c 5 -o results.txt

[ INF ] Current version v1.2.4 [updated]
[ INF ] Templates loaded: 15

[ SUCC ] https://example.com/index.js (URL)

Regex name: Google Map API Key
let google_map_api = "Aizashgkhheuuii746hg9875487hgjg"
Confidence: Unknown
```

### Scan with All Embedded Patterns

```bash
leakjs -f script.js --all regex/

[INF] Templates loaded: 517

script.js [AKIA1234567890123456] [High]
script.js [sk_live_1234567890] [High]
```

### Scan with Pattern Exclusion

```bash
leakjs -u "https://example.com/app.js" --exclude "AWS Access Key ID,Generic API Key"

[INF] Templates loaded: 515
```

### Benchmark Performance

```bash
leakjs -f script.js --benchmark 5

[INF] Running benchmark with 5 iterations...
[BENCH] Iteration 1/5
...
[BENCH] Benchmark completed!
Average scan time: 15.2ms
Average matches per iteration: 3
```

### Output in JSON Format

```bash
leakjs -u "https://example.com/script.js" -j

{
  "Stripe Secret Token": {
    "matches": ["stripe:7b1a32f7h8377c0a"],
    "confidence": "Critical"
  }
}
```

### Scan Multiple URLs from a File

```bash
leakjs -l urls.txt -p patterns.yaml -c 5 -o results.txt

[ INF ] Current version v1.2.4 [updated]
[ INF ] Templates Loaded: 15

[ SUCC ] https://example.com/index.js
Regex Name: Google Map API Key
let google_map_api = "Aizashgkhheuuii746hg9875487hgjg"
Confidence: Unknown

[ SUCC ] https://example.com/index.js
Regex Name: Google Drive API Key
let google_map_api = "Aizashgkhcgdheuuii746hg9875487hhsx"
Confidence: Unknown
```

### Scan a Local JavaScript File

```bash
leakjs -f script.js -p patterns.yaml -c 5 -o results.txt

[ INF ] Current version v1.2.4 [updated]
[ INF ] Templates Loaded: 34

[ SUCC ] https://example.com/index.js
Regex Name: Email Addresses
subscriber:"johndoe@gmail.com"
Confidence: Low
```

### Directly Input Regex Patterns
```bash
leakjs -u "https://example.com/script.js" -r "[\w\.-]+@[a-zA-Z\d\.-]+\.[a-zA-Z]{2,};\b\d{3}-\d{2}-\d{4}\b" -c 5 -o results.txt

[ INF ] Current version v1.2.4 [updated]
[ INF ] Templates Loaded: 2

[ SUCC ] https://example.com/index.js
Regex Name: Social Security Number
_social_security:"A334-5677-982"
Confidence: High

Regex Name: Google Meet-ID
compact_meeting = 0; var let id="https://meet.google.com/AED-56GBE-904"
Confidence: Medium
```

### Display Help

```bash
leakjs -h

                        __    v1.2.4     __       _______
                       / /   ___  ____ _/ /__    / / ___/
                      / /   / _ \/ __ `/ //_/_  / /\__ \
                     / /___/  __/ /_/ / ,< / /_/ /___/ /
                    /_____/\___/\__,_/_/|_|\____//____/
                                               @rootplinix


    LeakJS - A JavaScript leak detection tool

    Usage: leakjs [options]

    Options:
      -a, --all string        Path to directory containing all regex YAML files to load
      -b, --benchmark int     Run benchmark with specified number of iterations
      -C, --concurrency int   Number of concurrent requests (default 1)
      -c, --config string     Path to configuration YAML file
      -e, --exclude string    Comma-separated list of pattern names to exclude
      -f, --file string       Path to a JavaScript file to scan
      -h, --help              Show help message and exit
      -j, --json              Output results in JSON format
      -l, --list string       Path to the file containing URLs
      -o, --output string     Path to the output file to save results
      -p, --patterns string   Path to the YAML file containing patterns
      -r, --regex string      Directly input regex patterns separated by ';'
      -s, --silent            Show progress bar without any output in the terminal
      -t, --stats             Show scan statistics at the end
      -u, --url string        Single URL to scan
      -v, --verbose           Enable verbose logging

    Examples:
      leakjs -u http://example.com/app.js
      leakjs -l urls.txt --all regex/
      leakjs -f script.js --exclude "AWS Access Key ID"
      leakjs --benchmark 3 -f script.js
```

## ⚙️ Configuration

LeakJS automatically discovers configuration files in standard locations, but you can also specify custom config files.

### Automatic Config Discovery
LeakJS looks for config files in the following order:
1. `~/.config/leakjs/config.yaml` (recommended)
2. `~/.leakjs/config.yaml` (fallback)
3. Command-line `--config` flag (overrides automatic discovery)

### Example Configuration File
Create `~/.config/leakjs/config.yaml`:

```yaml
# Concurrency settings
concurrency: 4

# Output options
verbose: true
silent: false
json: false
stats: true

# Pattern configuration
patterns_dir: "regex/"  # Optional: custom pattern directory
patterns_file: ""       # Optional: custom pattern file
exclude: "AWS Access Key ID,Generic API Key"  # Comma-separated patterns to exclude

# Output settings
output: "results.txt"
```

### Custom Pattern Files
You can still load additional patterns from YAML files:

```yaml
patterns:
  - pattern:
      name: "Custom API Key"
      regex: "custom[_-]?key[_-]?[=:]\s*[\"']?([a-zA-Z0-9_-]{20,})[\"']?"
      confidence: "High"
  - pattern:
      name: "Internal Token"
      regex: "internal[_-]?token[_-]?[=:]\s*[\"']?([a-zA-Z0-9_-]{32,})[\"']?"
      confidence: "Medium"
```

## 🤖 Automated Releases

LeakJS uses GitHub Actions for automated building and releasing:

- **Trigger**: Automatic releases on every push to the `main` branch
- **Platforms**: Linux (AMD64/ARM64), macOS (AMD64/ARM64), Windows (AMD64/ARM64)
- **Artifacts**: Optimized binaries with stripped debug information
- **Versioning**: Uses Git tags and commit hashes for version identification

### Manual Release
You can also trigger releases manually using the GitHub Actions workflow dispatch feature.

---

## 🤝 Contributing

We welcome contributions! Please read our [CONTRIBUTING.md](https://github.com/the5orcerer/LeakJS/blob/main/CONTRIBUTING.md) for more information on how to get started.

### Contributors

- **the5orcerer**
  - GitHub: [the5orcerer](https://github.com/the5orcerer)
  - Twitter: [n30ron](https://twitter.com/n30ron)

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/the5orcerer/LeakJS/blob/main/LICENSE) file for details.

## 📧 Contact

For any questions or suggestions, feel free to open an issue or contact us directly.

![Twitter](https://img.shields.io/twitter/follow/n30ron?style=social) ![GitHub](https://img.shields.io/github/followers/the5orcerer?style=social)

---

<p align="center">
  <img src="https://forthebadge.com/images/badges/made-with-go.svg">
  <img src="https://forthebadge.com/images/badges/built-with-love.svg">
</p>
