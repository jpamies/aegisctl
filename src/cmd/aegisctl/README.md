# aegisctl — CLI UX

> Command-line interface for Aegis.

---

## Usage

```
aegisctl <command> [flags]
```

## Commands

### `analyze` — Scan repository

```bash
aegisctl analyze --repo <path>
```

| Flag | Type | Default | Description |
|---|---|---|---|
| `--repo` | string | `.` | Path to the application repository to analyse. |
| `--format` | string | `text` | Output format: `text`, `json`, `markdown`. |
| `--verbose` | bool | `false` | Show detailed findings. |

**Example:**
```bash
aegisctl analyze --repo ../my-app --format json
```

### `pack` — Generate architecture pack

```bash
aegisctl pack --repo <path> --output <path> [--deploy-mode <mode>]
```

| Flag | Type | Default | Description |
|---|---|---|---|
| `--repo` | string | `.` | Path to the application repository. |
| `--output` | string | `./output` | Directory for generated artefacts. |
| `--deploy-mode` | string | `generate` | Deploy mode: `generate`, `manual`, `auto`. |
| `--env` | string | `dev` | Target environment: `dev`, `staging`, `prod`. |
| `--migrate-aws` | bool | `false` | Include AWS → Azure migration assessment. |

**Example:**
```bash
# Generate-only (default, safe)
aegisctl pack --repo ../my-app --output ./output

# Generate with manual deploy workflow configured
aegisctl pack --repo ../my-app --output ./output --deploy-mode manual

# Generate with auto deploy (still requires Environment approval)
aegisctl pack --repo ../my-app --output ./output --deploy-mode auto
```

### `score` — WAF scorecard

```bash
aegisctl score --repo <path>
```

| Flag | Type | Default | Description |
|---|---|---|---|
| `--repo` | string | `.` | Path to the application repository. |
| `--format` | string | `text` | Output format: `text`, `json`, `markdown`. |
| `--output` | string | (stdout) | File path to write scorecard (default: stdout). |

**Example:**
```bash
aegisctl score --repo ../my-app --format markdown --output ./scorecard.md
```

### `migrate-aws` — AWS → Azure mapping

```bash
aegisctl migrate-aws --repo <path>
```

| Flag | Type | Default | Description |
|---|---|---|---|
| `--repo` | string | `.` | Path to the application repository. |
| `--output` | string | `./output` | Directory for migration report. |
| `--format` | string | `markdown` | Output format: `text`, `json`, `markdown`. |

**Example:**
```bash
aegisctl migrate-aws --repo ../my-app --output ./output
```

---

## Global flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--help` | bool | `false` | Show help for any command. |
| `--version` | bool | `false` | Print version and exit. |
| `--quiet` | bool | `false` | Suppress non-essential output. |
| `--no-color` | bool | `false` | Disable coloured output. |

---

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | General error |
| `2` | Invalid arguments |
| `3` | Repository not found |
| `4` | Analysis failed |
| `5` | Pack generation failed |

---

## Deploy mode behaviour

| Mode | What happens |
|---|---|
| `generate` (default) | Generates all artefacts locally. No Azure interaction. No deploy workflow triggered. |
| `manual` | Same as `generate`, but deploy.yml is configured for `workflow_dispatch` (manual trigger). Deployment still requires GitHub Environment approval. |
| `auto` | Same as `generate`, but deploy.yml also triggers on push to `main`. Deployment **still** requires GitHub Environment approval. |

> **Important:** Deploy mode `auto` does NOT mean unattended deployment.
> Environment approval is always required. The "auto" only refers to the
> _trigger_ — the approval gate remains mandatory.
