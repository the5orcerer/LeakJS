
# LeakJS
![Python](https://img.shields.io/badge/python-3.6%2B-blue) ![Issues](https://img.shields.io/github/issues/the5orcerer/LeakJS) ![Forks](https://img.shields.io/github/forks/the5orcerer/LeakJS) ![Stars](https://img.shields.io/github/stars/the5orcerer/LeakJS)![Contributors](https://img.shields.io/github/contributors/the5orcerer/LeakJS)

LeakJS is a powerful tool for scanning JavaScript files and URLs for predefined patterns. It supports concurrent scanning, custom pattern definitions, and more. 

## Features

- **Asynchronous Scanning**: Fast and efficient scanning using asyncio and aiohttp.
- **Custom Patterns**: Define your own patterns in YAML files.
- **Concurrent Execution**: Utilize multiple threads for concurrent scanning.
- **Detailed Logging**: Comprehensive logging for debugging and analysis.
- **Configurable**: Easily configure headers, patterns, and more.

## Installation

```bash
git clone https://github.com/the5orcerer/LeakJS
cd LeakJS
pip install .
```

## Usage

LeakJS can be used as a command-line tool to scan JavaScript files and URLs for patterns.

### Scan a Single URL

```bash
leakjs -u "https://example.com/script.js" -p patterns.yaml -c 5 -o results.txt
```

### Scan Multiple URLs from a File

```bash
leakjs -l urls.txt -p patterns.yaml -c 5 -o results.txt
```

### Scan a Local JavaScript File

```bash
leakjs -f script.js -p patterns.yaml -c 5 -o results.txt
```

### Directly Input Regex Patterns

```bash
leakjs -u "https://example.com/script.js" -r "[\w\.-]+@[a-zA-Z\d\.-]+\.[a-zA-Z]{2,};\b\d{3}-\d{2}-\d{4}\b" -c 5 -o results.txt
```

### Display Help

```bash
leakjs -h
```

## Configuration

LeakJS will look for default pattern files in the `~/.config/LeakJS` directory. You can place your custom pattern YAML files in this directory for automatic loading.

## Contribution

We welcome contributions! Please read our [CONTRIBUTING.md](https://github.com/yourusername/LeakJS/blob/main/CONTRIBUTING.md) for more information on how to get started.

## License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/yourusername/LeakJS/blob/main/LICENSE) file for details.

## Contact

For any questions or suggestions, feel free to open an issue or contact us directly.

![Twitter](https://img.shields.io/twitter/follow/n30ron?style=social) ![GitHub](https://img.shields.io/github/followers/the5orcerer?style=social) 

---

<p align="center">
  <img src="https://forthebadge.com/images/badges/made-with-python.svg">
  <img src="https://forthebadge.com/images/badges/built-with-love.svg">
</p>
