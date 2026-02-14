# Presentation Deck Outline — Aegis

> Outline for a short presentation (5–10 slides) introducing Aegis.
> Use this as the basis for a slide deck in your preferred tool
> (PowerPoint, Google Slides, Marp, etc.).

---

## Slide 1 — Title

**Aegis: Azure Architecture Pack Generator**

_Analyse. Pack. Score. Deploy (safely)._

- SRE/DevOps tool for automated Azure architecture alignment
- Built on the Azure Well-Architected Framework (WAF)
- Minimum-cost defaults · Generate-only by default · Approval-gated deploy

`TODO: Add presenter name, date, event name`

---

## Slide 2 — The Problem

**Manual architecture review doesn't scale.**

- WAF reviews are expert-intensive and time-consuming
- Secrets sprawl across repos with no systematic remediation
- IaC and CI/CD are hand-crafted per project
- AWS → Azure migrations lack a consistent methodology
- Cost optimisation is an afterthought

_Result: insecure, over-provisioned, and fragile infrastructure._

---

## Slide 3 — The Solution

**Aegis automates the first 80% of Azure architecture alignment.**

| Command | What it does |
|---|---|
| `aegisctl analyze` | Scan repo for languages, services, secrets, architecture signals |
| `aegisctl pack` | Generate docs, Bicep, GitHub Actions, scorecard, security report |
| `aegisctl score` | Heuristic WAF scorecard (0–5 per pillar) + top 5 improvements |
| `aegisctl migrate-aws` | Best-effort AWS → Azure service mapping |

---

## Slide 4 — Architecture Diagram

> _Insert the Mermaid diagram from [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md)
> or a rendered PNG version._

Key points:
- Developer runs `aegisctl` locally
- Generates docs, Bicep, and workflows to disk
- Optional deployment via GitHub Actions with Environment approval gates
- Azure resources: App Service, Key Vault, App Configuration, Managed Identity, Monitor

---

## Slide 5 — WAF Alignment

**Every decision maps to a WAF pillar:**

| Pillar | Key decision |
|---|---|
| Reliability | PaaS with health probes, auto-restart |
| Security | Managed Identity first → Key Vault → App Configuration |
| Cost Optimization | Minimum-viable SKUs, consumption-based where possible |
| Operational Excellence | Three-workflow CI/CD, IaC-only changes |
| Performance Efficiency | Autoscale, CDN, caching guidance |

_Scores are heuristic and non-official — always disclose this._

---

## Slide 6 — Business Value

- **Time**: Reduce architecture setup from days → minutes
- **Cost**: Minimum-cost defaults prevent over-provisioning from day 1
- **Security**: Automated secrets scanning + remediation hierarchy
- **Compliance**: WAF checklist provides audit trail
- **Migration**: Structured AWS → Azure path for hybrid/multi-cloud teams

---

## Slide 7 — Safety & Guardrails

- **Generate-only by default** — nothing deployed without opt-in
- **Approval-gated deploy** — GitHub Environment approval always required
- **Incremental mode only** — never `Complete` (prevents accidental deletion)
- **No secrets in output** — all detected secrets replaced with `REDACTED`
- **Heuristic scores with disclaimer** — transparency, not false authority

---

## Slide 8 — Demo

> _Run the 3-minute demo from [docs/DEMO_SCRIPT.md](../docs/DEMO_SCRIPT.md)._

1. `aegisctl analyze --repo .` → show findings
2. `aegisctl pack --repo . --output ./output` → show generated pack
3. `aegisctl score --repo .` → show scorecard + top 5 improvements

---

## Slide 9 — Repo & Links

| Resource | Link |
|---|---|
| Repository | `TODO: https://github.com/<org>/aegisctl` |
| Documentation | `docs/README.md` |
| WAF Scorecard | `docs/WAF_SCORECARD.md` |
| Responsible AI | `docs/RAI_NOTES.md` |
| Azure WAF | https://learn.microsoft.com/azure/well-architected/ |

---

## Slide 10 — Q&A

**Questions?**

`TODO: Add contact information or discussion link`
