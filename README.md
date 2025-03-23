
# LeakJS

![Python](https://img.shields.io/badge/python-3.6%2B-blue) ![Issues](https://img.shields.io/github/issues/the5orcerer/LeakJS) ![Forks](https://img.shields.io/github/forks/the5orcerer/LeakJS) ![Stars](https://img.shields.io/github/stars/the5orcerer/LeakJS) ![License](https://img.shields.io/github/license/the5orcerer/LeakJS)

## Getting Started

**LeakJS** is a robust tool designed to scan JavaScript files and URLs for predefined patterns, aiding in the identification of potential leaks and vulnerabilities. Leveraging `asyncio` and `aiohttp`, it ensures fast and efficient scanning. Users can define custom patterns in YAML files for tailored scans. Detailed logging and comprehensive configuration options make LeakJS a versatile solution for securing JavaScript code.

## 🚀 Key Features

- **Asynchronous Scanning**: Fast and efficient scanning using `asyncio` and `aiohttp`.
- **Custom Patterns**: Define your own patterns in YAML files.
- **Concurrent Execution**: Utilize multiple threads for concurrent scanning.
- **Detailed Logging**: Comprehensive logging for debugging and analysis.
- **Configurable**: Easily configure headers, patterns, and more.
## 📥 Installation

To install LeakJS, follow these steps:

```bash
git clone https://github.com/the5orcerer/LeakJS
cd LeakJS
python3 install.py
leakjs -h (For testing purpose)
```

## 🛠️ Usage

LeakJS can be used as a command-line tool to scan JavaScript files and URLs for patterns.

### Scan a Single URL

```bash
leakjs -u "https://example.com/script.js" -p patterns.yaml -c 5 -o results.txt

[ INF ] Current version v1.2.4 [updated]
[ INF ] Templates Loaded: 6

[ SUCC ] https://example.com/index.js
Regex Name: Stripe Secret Token
let google_map_api = "stripe:7b1a32f7h8377c0a"
Confidence: Critical
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
      -u, --url         Single URL to scan
      -l, --list        Path to the file containing URLs (one per line)
      -p, --patterns    Path to the YAML file containing patterns (optional)
      -r, --regex       Directly input regex patterns separated by ";"
      -f, --file        Path to a JavaScript file to scan
      -c, --concurrency Number of concurrent threads (default: 1)
      -o, --output      Path to the output file to save results
      -v, --verbose     Enable verbose logging
      -s, --silent      Show progress bar without any output in the terminal
      -t, --threads     Number of threads to use
      -up, --update     Update the tool automatically
      -upt, --updatetemplates Update the templates
      -h, --help        Show this help message and exit

    Examples:
      leakjs -u http://example.com/app.js
      leakjs -l urls.txt -p patterns.yaml
      leakjs -r "regex1;regex2" -f app.js -c 5
      leakjs -up
```

## ⚙️ Configuration

LeakJS will look for default pattern files in the `~/.config/LeakJS` directory. You can place your custom pattern YAML files in this directory for automatic loading.

### Example of a Custom Pattern YAML

```yaml
patterns:
  - pattern:
      name: "Email Address"
      regex: "[\\w\\.-]+@[a-zA-Z\\d\\.-]+\\.[a-zA-Z]{2,}"
      confidence: "High"
  - pattern:
      name: "Social Security Number"
      regex: "\\b\\d{3}-\\d{2}-\\d{4}\\b"
      confidence: "Medium"
```

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
  <img src="https://forthebadge.com/images/badges/made-with-python.svg">
  <img src="https://forthebadge.com/images/badges/built-with-love.svg">
</p>
