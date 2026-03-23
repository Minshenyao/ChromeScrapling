# Chrome Scraper

[中文说明](./README_ZH_CN.md)

A high-performance Chrome-based website information scraper built with Go and Rod.

## Features

- Fetch page titles
- Extract JavaScript asset URLs
- Detect web technologies
- Retrieve favicon URL and hash
- Capture screenshots
- Run with concurrent workers
- Support proxy and resume mode
- Export results as JSON or TXT

## Installation

### Build from source

```bash
git clone <repository-url>
cd ChromeScrapling
go mod tidy
go build -trimpath -ldflags="-s -w" -o chrome-scraper
```

### Run directly

```bash
go run . -u https://example.com
```

## Usage

### Examples

```bash
# Scan a single target
./chrome-scraper -u https://example.com

# Scan multiple targets
./chrome-scraper -u https://example.com,https://google.com

# Read targets from a file
./chrome-scraper -f urls.txt

# Delay collection for 2 seconds after page load
./chrome-scraper -u https://example.com --delay 2s

# Enable screenshots
./chrome-scraper -u https://example.com -s

# Use a proxy
./chrome-scraper -u https://example.com --proxy http://127.0.0.1:8080

# Increase concurrency
./chrome-scraper -u https://example.com -t 10

# Resume an interrupted run
./chrome-scraper -f urls.txt --resume resume.json
```

### Flags

| Flag | Short | Default | Description |
|------|------|---------|-------------|
| `--url` | `-u` | - | Comma-separated target URLs |
| `--file` | `-f` | - | File containing target URLs |
| `--threads` | `-t` | 3 | Number of concurrent workers |
| `--screenshot` | `-s` | false | Capture a screenshot |
| `--delay` | - | 0 | Delay after page load before collecting title, JS, fingerprint, favicon, and screenshot data. Supports values like `500ms`, `2s`, `1m` |
| `--proxy` | `-p` | - | Proxy server address |
| `--timeout` | - | 30 | Request timeout in seconds |
| `--output` | `-o` | - | Output file path |
| `--headless` | - | true | Run in headless mode |
| `--user-agent` | - | Mozilla/5.0... | Custom User-Agent |
| `--verbose` | `-v` | false | Enable verbose output |
| `--resume` | - | - | Resume state file path |

## Output

### Console

```text
[Start] starting scrape...    screenshots: false        delay: 0s        threads: 3        proxy: off
[+] https://google.com | 200 | Google
Scrape progress 100% [██████████████████████████████████████████████████] (1/1, 14 it/min)
[Stats] total: 1  success: 1  failed: 0  total time: 3.554s  average: 3.554s
```

### JSON

```json
[
  {
    "target_url": "https://google.com",
    "final_url": "https://www.google.com/",
    "status_code": 200,
    "title": "Google",
    "js_links": null,
    "fingerprint": [
      "HTTP/3"
    ],
    "favicon_icon": "https://www.google.com/favicon.ico",
    "favicon": 708578229
  }
]
```

### Screenshots

When `-s` is enabled, screenshots are saved immediately to the local `screenshots/` directory. Files are named from the target URL, for example:

```text
screenshots/https-google-com.png
```

## Notes

1. Chrome or Chromium must be installed.
2. `--delay` applies even when screenshots are disabled.
3. Screenshot files are saved in real time before resume state is marked complete.
4. Resume mode skips only targets that completed successfully.

## Development

```text
ChromeScrapling/
├── main.go
├── scraper.go
├── types.go
├── go.mod
├── README.md
└── README_ZH_CN.md
```
