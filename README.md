# LeakJS

![Go](https://img.shields.io/badge/go-1.18%2B-blue) ![Issues](https://img.shields.io/github/issues/the5orcerer/LeakJS) ![Forks](https://img.shields.io/github/forks/the5orcerer/LeakJS) ![Stars](https://img.shields.io/github/stars/the5orcerer/LeakJS)

## Getting Started

**LeakJS** is a high-performance reconnaissance and security tool designed to crawl websites, discover JavaScript files, and scan them for sensitive data leaks (API keys, tokens, credentials, secrets, endpoints, and exposed source maps). Leveraging Go's goroutines, it delivers blazingly fast scanning, crawling, and analysis.

---

## 🚀 Key Features

- **Built-in Web Crawler**: Spider entire websites to discover all JavaScript URLs (`<script src>`, `<link rel="preload">`, inline scripts, dynamic imports, webpack chunks).
- **Crawl-Only Mode**: Gather all JS URLs from a site and export them directly to files or stdout for piping into other tools.
- **Authenticated Requests**: Attach custom session cookies (`--cookie`) and custom HTTP headers (`-H`) across both crawling and scanning.
- **API Endpoint & Route Discovery**: Extract REST routes, API endpoints, GraphQL paths, and internal URLs (`-E / --endpoints`).
- **Source Map Detection**: Check for exposed JavaScript source maps (`-m / --sourcemaps`) to uncover original source code.
- **Proxy & TLS Support**: Route traffic through proxies like Burp Suite / OWASP ZAP (`--proxy`) with TLS verification bypass (`-k / --insecure`).
- **Local Directory Scanning**: Recursively scan entire directories of downloaded JS files (`-d / --dir`).
- **Embedded Regex Patterns**: 170+ high-confidence patterns embedded directly in the binary—works out-of-the-box with zero setup.
- **High Concurrency & Rate Limiting**: Fine-grained concurrency (`-C`) and request delay (`--delay`) to control speed and avoid WAF bans.

---

## 📥 Installation

### Option 1: Build from Source (Recommended)

```bash
git clone https://github.com/the5orcerer/LeakJS
cd LeakJS
make build
sudo make install
```

Or using Go directly:

```bash
go build -o leakjs ./cmd/leakjs
sudo mv leakjs /usr/local/bin/
```

### Option 2: Pre-built Binaries

Download pre-compiled binaries for Linux, macOS, or Windows from the [Releases](https://github.com/the5orcerer/LeakJS/releases) page.

---

## 🛠️ Usage & Examples

### 1. Embedded Patterns Out-of-the-Box Scan

LeakJS embeds comprehensive pattern definitions into the binary, so you can scan immediately without specifying any pattern file:

```bash
leakjs -u "https://example.com/app.js"
```

### 2. Crawl Website & Gather JS URLs (`--crawl` & `--crawl-only`)

Gather all JavaScript URLs discovered across an entire site:

```bash
# Crawl site and print all discovered JS files
leakjs -u "https://example.com" --crawl --crawl-only -o js_urls.txt

# Crawl up to depth 3 with max 100 pages, including subdomains
leakjs -u "https://example.com" --crawl --crawl-only --depth 3 --max-pages 100 --subdomains
```

### 3. Crawl Website and Scan for Leaks

Spider a target website, extract all JS files, and immediately scan them for secrets and API keys:

```bash
leakjs -u "https://example.com" --crawl -o leaks.txt
```

### 4. Authenticated Requests (Cookies & Custom Headers)

Pass session cookies or authorization tokens to crawl protected areas and scan authenticated JS bundles:

```bash
# Attach session cookie
leakjs -u "https://target.com" --crawl --cookie "session=abc123xyz; remember_me=true"

# Attach Authorization header & custom headers
leakjs -u "https://target.com/app.js" -H "Authorization: Bearer eyJhbGci..." -H "X-Custom-Auth: secret"
```

### 5. Extract API Endpoints & Routes (`-E / --endpoints`)

Extract all internal API endpoints, routes (`/api/v1/...`, `/graphql`, `/admin/...`), and URLs embedded in the JavaScript code:

```bash
leakjs -u "https://example.com/bundle.js" -E
```

### 6. Detect Exposed Source Maps (`-m / --sourcemaps`)

Detect if target JavaScript files reference exposed `.map` files that leak original unminified source code:

```bash
leakjs -u "https://example.com/main.js" -m
```

### 7. Proxy via Burp Suite or OWASP ZAP

Route requests through a local proxy with self-signed certificate support:

```bash
leakjs -u "https://example.com" --crawl --proxy "http://127.0.0.1:8080" -k
```

### 8. Recursively Scan a Local Directory (`-d / --dir`)

Scan all `.js` files in a folder:

```bash
leakjs -d ./downloaded_scripts/ -o results.txt
```

### 9. Scan Multiple Targets from a List

```bash
# Scan a list of JS URLs
leakjs -l urls.txt -C 25 -o results.txt

# Crawl a list of root target websites
leakjs -l targets.txt --crawl --crawl-only -o all_js.txt
```

### 10. Filter Patterns & Output JSON

```bash
# Show only high-confidence matches
leakjs -u "https://example.com/app.js" --high-only

# Exclude noisy patterns
leakjs -u "https://example.com/app.js" --exclude "Generic API Key,Email Addresses"

# Output structured JSON
leakjs -u "https://example.com/app.js" -j
```

---

## ⚙️ CLI Reference

```text
Usage:
  leakjs [flags]

Flags:
  -u, --url string           Single URL to scan or seed URL to crawl
  -l, --list string          Path to file containing URLs or crawl targets
  -f, --file string          Path to a local JavaScript file to scan
  -d, --dir string           Recursively scan a local directory for JavaScript files

Crawl Options:
      --crawl                Crawl website starting from seed URL(s) to discover all JS files
      --crawl-only           Only crawl and output discovered JS URLs without scanning for leaks
      --depth int            Maximum crawling depth (default 2)
      --max-pages int        Maximum number of pages to crawl per target (default 50)
      --subdomains           Include subdomains during crawling

HTTP & Auth:
      --cookie string        Cookie string attached to all requests (e.g. 'session=xyz; token=abc')
      --cookies string       Alias for --cookie
  -H, --header stringArray   Custom HTTP header(s), can be used multiple times
      --proxy string         HTTP/HTTPS/SOCKS5 proxy (e.g. http://127.0.0.1:8080)
  -k, --insecure             Skip TLS/SSL certificate verification
      --timeout int          HTTP request timeout in seconds (default 15)
      --delay int            Delay between requests in milliseconds
      --user-agent string    Custom User-Agent header

Recon Tools:
  -E, --endpoints            Extract API endpoints, REST routes, and URLs from JS files
  -m, --sourcemaps           Detect and verify exposed JavaScript source maps (.map)

Patterns:
  -p, --patterns string      Path to YAML pattern file (defaults to built-in patterns)
  -r, --regex string         Direct regex patterns separated by ';'
  -a, --all string           Path to directory containing all regex YAML files to load
  -c, --config string        Path to configuration YAML file
  -e, --exclude string       Comma-separated list of pattern names to exclude
      --high-only            Show only high-confidence pattern matches

Output & Concurrency:
  -C, --concurrency int      Number of concurrent requests (default 20)
  -o, --output string        Path to output file to save results
  -v, --verbose              Enable verbose logging
  -s, --silent               Show progress bar without console output
  -j, --json                 Output results in JSON format
  -t, --stats                Show scan statistics at the end
  -b, --benchmark int        Run benchmark with specified number of iterations
  -h, --help                 Show help message and exit
```

---

## ⚙️ Configuration File

LeakJS automatically checks for `~/.config/leakjs/config.yaml` or `~/.leakjs/config.yaml`:

```yaml
concurrency: 20
timeout: 15
delay: 0
depth: 2
max_pages: 50
subdomains: false
cookies: "session=xyz123"
headers:
  - "Authorization: Bearer tokenXYZ"
  - "X-Requested-With: XMLHttpRequest"
user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
endpoints: true
sourcemaps: true
output: "results.txt"
verbose: false
silent: false
json: false
stats: true
```

---

## 🤖 Automated Releases & Versioning

LeakJS features a fully automated CI/CD release pipeline via GitHub Actions ([.github/workflows/release.yml](.github/workflows/release.yml)):

- **Automatic Semantic Versioning**: Calculates the next version (`vMAJOR.MINOR.PATCH`) based on commits since the previous release (`feat` -> minor bump, `fix` -> patch bump, `BREAKING CHANGE` -> major bump).
- **Automated Changelog**: Generates a categorized release change log (Features, Bug Fixes, Improvements, Documentation) and publishes it with each release.
- **Cross-Platform Compilation**: Automatically builds and packages binaries for:
  - Linux (`amd64`, `arm64`)
  - macOS (`amd64`, `arm64` Apple Silicon)
  - Windows (`amd64`, `arm64`)
- **Integrity Verification**: Generates and uploads SHA-256 `checksums.txt` for all release assets.
- **Manual Trigger Support**: Releases can also be dispatched manually via GitHub Actions with optional version overrides.

---

## 🧹 Repository Size & History Cleanup

If you previously cloned the repository and noticed a larger download size (~13MB), this was due to historical snapshots of compiled binaries (`leakjs`) that were committed in past revisions.

A clean `.gitignore` is now in place. To purge past binary blobs from your local clone and reduce the Git repository to **~250KB**, run:

```bash
git filter-branch --force --index-filter 'git rm --cached --ignore-unmatch leakjs' --prune-empty --tag-name-filter cat -- --all
rm -rf .git/refs/original/
git reflog expire --expire=now --all
git gc --prune=now
```

Then push to GitHub:
```bash
git push --force --all origin
git push --force --tags origin
```

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/the5orcerer/LeakJS/issues).

### Contributors
- **the5orcerer** ([@the5orcerer](https://github.com/the5orcerer))

## 📜 License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
