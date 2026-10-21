# Changelog

All notable changes to **LeakJS** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v1.3.0] - 2026-10-22

### 🚀 Features
- **Built-in Web Crawler Mode (`--crawl`)**: Spiders entire websites from seed URLs to discover JavaScript files across `<script src>`, `<link rel="preload">`, dynamic imports, inline scripts, and webpack runtime chunks.
- **Crawl-Only Mode (`--crawl-only`)**: Gathers all discovered JavaScript URLs across a target site and outputs them directly to console or file without running pattern scans.
- **Authenticated Requests (Cookies & Custom Headers)**:
  - Added `--cookie` / `--cookies` flag to pass session cookies to both crawler and scanner.
  - Added `-H` / `--header` flag to pass custom headers (e.g. `Authorization: Bearer <token>`, `X-Custom-Auth: ...`).
- **Proxy Support (`--proxy`) & Insecure TLS (`-k` / `--insecure`)**: Route requests through proxies like Burp Suite or OWASP ZAP with self-signed certificate support.
- **API Endpoint & Route Discovery (`-E` / `--endpoints`)**: Automatically extract REST routes (`/api/...`, `/v1/...`), GraphQL endpoints, and internal paths from JavaScript code.
- **Source Map Detection (`-m` / `--sourcemaps`)**: Detect and verify exposed source map (`.js.map`) files that leak original source code.
- **Recursive Directory Scan (`-d` / `--dir`)**: Recursively scan entire local folders of JavaScript files.
- **Embedded Comprehensive Patterns**: Embedded 170+ patterns via `//go:embed` so LeakJS works out of the box with zero flags.
- **Automated Versioning & Releases**: GitHub Actions workflow that automatically computes semantic versioning, generates categorized changelogs, builds multi-arch binaries, generates checksums, and publishes releases.

### 🐛 Bug Fixes
- Fixed query parameter truncation in URL parsing (`.js?v=...`).
- Fixed concurrency deadlock when concurrency flag was omitted in subroutines.
- Fixed Git repository size issue caused by historical compiled binary commits.

---

## [v1.2.4] - Previous Release
- Initial Go conversion and regex pattern engine.
- YAML pattern file loading and multi-threaded scanning.

