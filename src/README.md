# Source — Go Module Layout

> This directory contains the Go source code for Aegis (`aegisctl`).

---

## Intended module layout

```
src/
├── cmd/
│   └── aegisctl/         ← CLI entry point (main package)
│       ├── main.go       ← TODO: CLI dispatcher
│       └── README.md     ← CLI UX documentation
├── internal/
│   ├── analyzer/         ← TODO: repo analysis engine
│   │   ├── analyzer.go   ← Scan repo for languages, services, secrets
│   │   ├── secrets.go    ← Secret pattern matching
│   │   └── services.go   ← Cloud service detection
│   ├── packer/           ← TODO: architecture pack generator
│   │   ├── packer.go     ← Orchestrate pack generation
│   │   ├── docs.go       ← Generate Markdown documentation
│   │   ├── bicep.go      ← Generate Bicep templates
│   │   └── workflows.go  ← Generate GitHub Actions YAML
│   ├── scorer/           ← TODO: WAF scorecard engine
│   │   ├── scorer.go     ← Score calculation logic
│   │   └── rules.go      ← Scoring rules per pillar
│   ├── migrator/         ← TODO: AWS → Azure migration mapper
│   │   ├── migrator.go   ← Migration assessment orchestrator
│   │   └── mapping.go    ← Service mapping database
│   └── output/           ← TODO: output formatting
│       ├── markdown.go   ← Markdown renderer
│       └── writer.go     ← File output writer
├── go.mod                ← TODO: Go module definition
└── README.md             ← you are here
```

## Commands

| Command | Description |
|---|---|
| `aegisctl analyze` | Scan a repository and report findings (languages, services, secrets, architecture signals). |
| `aegisctl pack` | Generate the full architecture pack (docs, Bicep, workflows, scorecard). |
| `aegisctl score` | Produce a WAF scorecard with heuristic scoring (0–5 per pillar). |
| `aegisctl migrate-aws` | Best-effort AWS → Azure service mapping based on repo analysis. |

## Design principles

1. **Go stdlib only** — no third-party modules. All functionality uses the Go
   standard library (net/http, encoding/json, text/template, os, filepath, etc.).
2. **Single binary** — `aegisctl` compiles to one static binary with zero
   runtime dependencies.
3. **Offline-first** — analysis and pack generation work entirely offline.
   Only deployment-related commands need Azure connectivity.
4. **Template-driven output** — all generated Markdown and Bicep use Go
   `text/template` for consistency and customisability.

## Building

```bash
cd src
go build -o ../bin/aegisctl ./cmd/aegisctl
```

## Testing

```bash
cd src
go test ./...
```
