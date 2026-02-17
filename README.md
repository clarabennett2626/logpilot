# LogPilot 🧭

**Multi-source structured log viewer for the terminal.**

Stream, search, and correlate logs from files, Kubernetes, Docker, SSH, and more — all in one TUI.

## Features (Roadmap)

- 📂 **Multi-source**: Local files, Kubernetes pods, Docker containers, SSH remote, stdin/pipe
- 🔍 **Smart parsing**: Auto-detects JSON, logfmt, syslog, CLF, plain text
- ⚡ **Fast filtering**: Field-based queries (`level=error service=auth latency>500ms`)
- 📊 **Timeline visualization**: ASCII sparklines showing error rates over time
- 🔗 **Trace correlation**: Follow request IDs across multiple log sources
- ⌨️ **Vim keybindings**: Navigate logs like you navigate code
- 📦 **Zero infrastructure**: No Elasticsearch, no Loki — runs entirely in your terminal

## Installation

```bash
# Go install
go install github.com/clarabennett2626/logpilot/cmd/logpilot@latest

# Or download from releases
# https://github.com/clarabennett2626/logpilot/releases
```

## Quick Start

```bash
# View a local log file
logpilot app.log

# Pipe from another command
kubectl logs -f my-pod | logpilot

# Multiple sources (coming soon)
logpilot app.log k8s://default/api-server docker://redis
```

## Keybindings

| Key | Action |
|-----|--------|
| `j`/`k` | Scroll down/up |
| `G` | Jump to bottom |
| `gg` | Jump to top |
| `/` | Search |
| `n`/`N` | Next/previous match |
| `q` | Quit |

## Status

🚧 **Early development** — Phase 1 (MVP) in progress.

## License

MIT
