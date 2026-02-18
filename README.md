# LogPilot 🧭

[![CI](https://github.com/clarabennett2626/logpilot/actions/workflows/ci.yml/badge.svg)](https://github.com/clarabennett2626/logpilot/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/clarabennett2626/logpilot)](https://github.com/clarabennett2626/logpilot/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/clarabennett2626/logpilot)](https://goreportcard.com/report/github.com/clarabennett2626/logpilot)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Multi-source structured log viewer for the terminal.**

Stream, search, and correlate logs from files, Kubernetes, Docker, SSH, and more — all in one TUI.

![LogPilot Demo](docs/demos/demo-json.gif)

## Features

### ✅ Implemented
- 🔍 **Format auto-detection** — Automatically identifies JSON, logfmt, and plain text log formats
- 🎨 **Color-coded rendering** — Log levels rendered with distinct colors (DEBUG=gray, INFO=blue, WARN=yellow, ERROR=red, FATAL=red bold)
- ⏰ **Flexible timestamps** — Configurable display: relative ("2m ago"), ISO 8601, or local time
- 🌗 **Theme support** — Dark and light terminal themes
- 📂 **File reader** — Read and tail local log files with rotation handling and glob patterns
- 📥 **Stdin/pipe support** — Composable with any command: `kubectl logs -f | logpilot`
- ⚡ **Backpressure handling** — Configurable strategies (block or drop-oldest) for high-throughput streams
- 🔄 **Log rotation** — Detects file truncation and replacement, reopens automatically
- 📊 **Multi-file tailing** — Monitor multiple log files simultaneously with glob patterns

### 🚧 Coming Soon
- ☸️ **Kubernetes source** — Stream logs directly from pods
- 🐳 **Docker source** — Tail container logs
- 🔗 **SSH remote** — Read logs from remote servers
- 🔎 **Field-based filtering** — Queries like `level=error service=auth latency>500ms`
- 📈 **Timeline visualization** — ASCII sparklines for error rates
- 🔗 **Trace correlation** — Follow request IDs across sources
- ⌨️ **Vim keybindings** — Navigate logs like code

## Installation

### From Release (recommended)

Download the latest binary for your platform from [Releases](https://github.com/clarabennett2626/logpilot/releases/latest).

```bash
# Linux (amd64)
curl -L https://github.com/clarabennett2626/logpilot/releases/latest/download/logpilot_0.1.0_linux_amd64.tar.gz | tar xz
sudo mv logpilot /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/clarabennett2626/logpilot/releases/latest/download/logpilot_0.1.0_darwin_arm64.tar.gz | tar xz
sudo mv logpilot /usr/local/bin/
```

### From Source

```bash
go install github.com/clarabennett2626/logpilot/cmd/logpilot@latest
```

### Verify Installation

```bash
logpilot --version
# logpilot 0.1.0 (abc1234) built 2026-02-17T...
```

## Quick Start

```bash
# View a local log file
logpilot app.log

# Tail a log file (follows new lines)
logpilot -f /var/log/app.log

# Pipe from another command
kubectl logs -f my-pod | logpilot

# Streaming demo
![Pipe demo](docs/demos/demo-pipe.gif)
docker logs -f my-container | logpilot
cat /var/log/syslog | logpilot

# Multiple files with glob
logpilot /var/log/*.log
```

## Supported Log Formats

LogPilot auto-detects the format from the first few lines:

### JSON

![JSON logs](docs/demos/demo-json.gif)

```json
{"timestamp":"2026-02-17T20:30:00Z","level":"error","message":"connection timeout","service":"api","latency_ms":1523}
```

### Logfmt

![Logfmt logs](docs/demos/demo-logfmt.gif)

```
ts=2026-02-17T20:30:00Z level=error msg="connection timeout" service=api latency_ms=1523
```

### Plain Text

![Plain text logs](docs/demos/demo-plain.gif)

```
2026-02-17 20:30:00 ERROR connection timeout
Feb 17 20:30:00 myhost app[1234]: connection timeout
```

## Architecture

```
cmd/logpilot/        → CLI entry point
internal/
  parser/            → Format detection & parsing (JSON, logfmt, plain)
  source/            → Log sources (file, stdin; k8s, docker coming soon)
  tui/               → Terminal UI rendering (Bubble Tea + Lipgloss)
  config/            → Configuration
  filter/            → Query engine (coming soon)
  merge/             → Multi-source merge (coming soon)
```

## Development

```bash
# Build
go build -o logpilot ./cmd/logpilot/

# Test
go test ./... -v -race

# Benchmark
go test -bench=. ./internal/parser/

# Lint
go vet ./...
```

## CI/CD

- **CI**: Tests run on Go 1.22, 1.23, and 1.24 for every PR and push to main
- **Release**: GoReleaser builds binaries for linux/darwin/windows × amd64/arm64 on version tags

## Keybindings

| Key | Action |
|-----|--------|
| `j`/`k` | Scroll down/up |
| `G` | Jump to bottom |
| `gg` | Jump to top |
| `/` | Search |
| `n`/`N` | Next/previous match |
| `q` | Quit |

## License

MIT
