# Aegis — Documentation Hub

> **Aegis** (`aegisctl`) is an SRE/DevOps tool that analyzes an application
> repository and generates an **Azure Well-Architected, minimum-cost
> architecture pack**.

---

## Problem → Solution

### The problem

Engineering teams building on Azure face a recurring challenge: translating
application code into a production-ready, cost-optimised, secure architecture
that aligns with the **Azure Well-Architected Framework (WAF)**. Today this
requires:

1. Manual WAF review across five pillars.
2. Secrets/config audit with no automated remediation path.
3. Hand-crafted Bicep IaC and GitHub Actions pipelines.
4. Separate AWS-to-Azure migration analysis when applicable.

This is time-consuming, error-prone, and often skipped — leading to insecure,
over-provisioned, or fragile infrastructure.

### The solution

Aegis automates the first 80 % of that work:

```
aegisctl analyze   → scan repo for architecture signals
aegisctl pack      → generate the full architecture pack
aegisctl score     → produce a heuristic WAF scorecard
aegisctl migrate-aws → best-effort AWS → Azure mapping
```

**Output artefacts:**

| Artefact | Description |
|---|---|
| Architecture docs | WAF-aligned decision records |
| WAF checklist | Per-pillar checklist with status |
| WAF scorecard | 0–5 heuristic score per pillar + top 5 improvements |
| Security report | Secrets findings + remediation (Managed Identity → Key Vault → App Configuration) |
| Bicep templates | Minimum-cost IaC scaffold |
| GitHub Actions | CI, IaC validation, gated deploy workflows |
| AWS migration map | Service-by-service mapping (best-effort) |

---

## Prerequisites

| Requirement | Version |
|---|---|
| Go | 1.22+ |
| Azure CLI | 2.60+ |
| Bicep CLI | 0.25+ |
| Git | 2.x |
| GitHub CLI (optional) | 2.x |

No third-party Go modules are required — Aegis uses the Go standard library
only.

---

## Quickstart

```bash
# Clone
git clone https://github.com/<org>/aegisctl.git
cd aegisctl

# Build
go build -o bin/aegisctl ./src/cmd/aegisctl

# Analyze a target repo
aegisctl analyze --repo ../my-app

# Generate the architecture pack (generate-only, default)
aegisctl pack --repo ../my-app --output ./output

# Score the target repo against WAF
aegisctl score --repo ../my-app

# (Optional) AWS → Azure migration assessment
aegisctl migrate-aws --repo ../my-app --output ./output
```

---

## Demo flow (3-minute version)

See [DEMO_SCRIPT.md](DEMO_SCRIPT.md) for a step-by-step walkthrough.

1. **Analyze** — scan repo, detect languages, secrets, IaC, cloud services.
2. **Pack** — generate docs, Bicep, workflows, scorecard.
3. **Review** — open scorecard, walk through top 5 improvements.
4. **Deploy (optional)** — trigger `deploy.yml` via `workflow_dispatch` with
   Environment approval.

---

## Pack outputs

After running `aegisctl pack`, the `./output` directory contains:

```
output/
├── docs/
│   ├── ARCHITECTURE.md
│   ├── WAF_CHECKLIST.md
│   ├── WAF_SCORECARD.md
│   ├── SECURITY_AND_SECRETS.md
│   ├── IAC_AND_PIPELINE.md
│   └── AWS_MIGRATION.md      (if --migrate-aws flag used)
├── infra/
│   ├── main.bicep
│   ├── parameters.dev.json
│   └── parameters.prod.json
└── .github/workflows/
    ├── ci.yml
    ├── iac-validate.yml
    └── deploy.yml
```

---

## Responsible AI

Aegis generates advisory artefacts — it does not make deployment decisions on
your behalf. See [RAI_NOTES.md](RAI_NOTES.md) for full transparency and
guardrail documentation.

---

## Documentation index

| Document | Purpose |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | WAF-aligned architecture decisions + Mermaid diagram |
| [WAF_CHECKLIST.md](WAF_CHECKLIST.md) | Per-pillar checklist |
| [WAF_SCORECARD.md](WAF_SCORECARD.md) | Heuristic scoring (0–5) + top improvements |
| [SECURITY_AND_SECRETS.md](SECURITY_AND_SECRETS.md) | Secrets findings + remediation patterns |
| [IAC_AND_PIPELINE.md](IAC_AND_PIPELINE.md) | Bicep strategy + CI/CD design |
| [AWS_MIGRATION.md](AWS_MIGRATION.md) | AWS → Azure service mapping |
| [RAI_NOTES.md](RAI_NOTES.md) | Responsible AI notes |
| [DECISIONS.md](DECISIONS.md) | Assumptions, open questions, backlog |
| [DEMO_SCRIPT.md](DEMO_SCRIPT.md) | 3-minute demo walkthrough |
