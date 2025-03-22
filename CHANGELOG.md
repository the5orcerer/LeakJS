# Changelog

All notable changes to this project will be documented in this file.

## [1.2.4] - 2025-03-22

### Added
- **Asynchronous Scanning**: Enhanced performance using `asyncio` and `aiohttp`.
- **Custom Patterns**: Support for user-defined patterns in YAML files.
- **Tab Completion**: Added bash and fish shell tab completion support.
- **Remote Version Check**: Implemented version checking against the remote repository.

### Fixed
- Various minor bugs and stability improvements.

### Security
- Integrated `bandit` for security checks.

## [1.2.0] - 2025-01-15

### Added
- **Detailed Logging**: Comprehensive logging for debugging and analysis.
- **Concurrent Execution**: Utilization of multiple threads for concurrent scanning.

### Changed
- Updated dependency versions in `requirements.txt`.

### Fixed
- Fixed issues with pattern matching and scanning efficiency.

## [1.1.0] - 2024-12-01

### Added
- **Configuration Options**: Easily configurable headers, patterns, and more.
- **Verbose Logging**: Option to enable verbose logging.

### Changed
- Improved CLI help text and usage instructions.

### Fixed
- Addressed issues with URL scanning and result output formatting.

## [1.0.0] - 2024-10-01

### Initial Release
- **LeakJS**: A robust tool for scanning JavaScript files and URLs for predefined patterns.
- **Features**: Initial support for pattern matching, concurrent scanning, and detailed logging.
