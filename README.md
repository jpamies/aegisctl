# Aegis

**SRE/DevOps architecture pack generator for Azure — powered by the Well-Architected Framework.**

Aegis (`aegisctl`) analyzes an application repository and produces an
**Azure Well-Architected, minimum-cost architecture pack**: documentation,
WAF checklist & scorecard, secrets remediation guidance, Bicep IaC, GitHub
Actions CI/CD, and an optional AWS→Azure migration assessment.

> **Default behaviour:** generate-only — nothing is deployed unless you
> explicitly opt in to autodeploy with approval gates.

---

## Quick links

| Resource | Path |
|---|---|
| Full documentation | [docs/README.md](docs/README.md) |
| Architecture & WAF alignment | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |
| WAF Checklist | [docs/WAF_CHECKLIST.md](docs/WAF_CHECKLIST.md) |
| WAF Scorecard | [docs/WAF_SCORECARD.md](docs/WAF_SCORECARD.md) |
| Security & Secrets guidance | [docs/SECURITY_AND_SECRETS.md](docs/SECURITY_AND_SECRETS.md) |
| IaC & Pipeline strategy | [docs/IAC_AND_PIPELINE.md](docs/IAC_AND_PIPELINE.md) |
| AWS → Azure migration | [docs/AWS_MIGRATION.md](docs/AWS_MIGRATION.md) |
| Responsible AI notes | [docs/RAI_NOTES.md](docs/RAI_NOTES.md) |
| Decision log | [docs/DECISIONS.md](docs/DECISIONS.md) |
| Demo script (3 min) | [docs/DEMO_SCRIPT.md](docs/DEMO_SCRIPT.md) |
| Presentation deck outline | [presentations/DECK_OUTLINE.md](presentations/DECK_OUTLINE.md) |
| Agent instructions | [AGENTS.md](AGENTS.md) |

## Repository layout

```
aegisctl/
├── AGENTS.md
├── README.md          ← you are here
├── LICENSE
├── .gitignore
├── mcp.json
├── docs/              ← full documentation pack
├── src/               ← Go source (stdlib-only)
│   └── cmd/aegisctl/  ← CLI entry point
├── infra/             ← Bicep IaC templates
├── .github/workflows/ ← CI, IaC validation, deploy
└── presentations/     ← deck outline
```

## License

[MIT](LICENSE)
