# Decisions, Assumptions & Backlog

> Living document tracking architectural assumptions, open questions, and
> backlog items for Aegis.

---

## Assumptions

| # | Assumption | Rationale | Status |
|---|---|---|---|
| A1 | A CAF-compliant landing zone exists or will be provisioned separately. | Aegis focuses on workload architecture (WAF), not subscription/management group topology (CAF). | ✅ Active |
| A2 | GitHub is the source control and CI/CD platform. | Aegis generates GitHub Actions workflows. Azure DevOps support is out of scope for v1. | ✅ Active |
| A3 | Azure is the target cloud. | All IaC, recommendations, and service mappings target Azure. | ✅ Active |
| A4 | Go stdlib is sufficient for the CLI. | Avoids dependency management complexity; Go's stdlib covers HTTP, JSON, file I/O, templates. | ✅ Active |
| A5 | Users have Azure CLI and Bicep CLI installed. | Required for IaC validation and deployment; not bundled with Aegis. | ✅ Active |
| A6 | The analysed repository is a single workload. | Multi-workload monorepos would need one Aegis run per workload. | ⚠️ To validate |
| A7 | Secrets detection uses pattern matching, not live credential testing. | Safer and offline; may produce false positives. | ✅ Active |

---

## Open questions

| # | Question | Context | Owner | Status |
|---|---|---|---|---|
| Q1 | Should Aegis support multi-region architectures by default? | Currently single-region is the minimum-cost default. Multi-region increases cost significantly. | `TODO:` | Open |
| Q2 | What is the minimum Go version to support? | Currently targeting Go 1.22+. | `TODO:` | Open |
| Q3 | Should the WAF scorecard support custom weightings? | Different teams may prioritise different pillars. | `TODO:` | Open |
| Q4 | How should Aegis handle monorepos with multiple services? | Current assumption is single-workload. Could support `--service` flag. | `TODO:` | Open |
| Q5 | Should Aegis generate Terraform as an alternative to Bicep? | Out of scope for v1; community request possible. | `TODO:` | Deferred |
| Q6 | What level of AWS migration depth is expected? | Currently best-effort service mapping. Deep migration analysis is a different tool. | `TODO:` | Open |

---

## Decision log

| # | Date | Decision | Rationale | Alternatives considered |
|---|---|---|---|---|
| D1 | 2026-02-14 | Use Bicep exclusively (no ARM JSON, no Terraform). | Azure-native, no state file, readable DSL. | Terraform (multi-cloud but adds complexity), ARM JSON (verbose). |
| D2 | 2026-02-14 | Default to generate-only mode with opt-in deployment. | "Do no harm" — users must explicitly request deployment. | Default to manual deploy (too risky for a new tool). |
| D3 | 2026-02-14 | Use GitHub Actions exclusively for CI/CD. | Simplifies pipeline generation; GitHub is the assumed platform. | Azure DevOps (adds second platform), Jenkins (legacy). |
| D4 | 2026-02-14 | WAF scorecard is heuristic with explicit disclaimer. | Automated scoring is useful but cannot replace official review. | No scoring (less actionable), official review integration (not feasible). |
| D5 | 2026-02-14 | Secrets remediation hierarchy: MI → Key Vault → App Config. | Follows Azure security best practices. | All secrets to Key Vault (wasteful for non-secrets). |
| D6 | 2026-02-14 | Go stdlib only — no third-party modules. | Reduces supply chain risk, simplifies builds. | Allow curated modules (adds maintenance burden). |

---

## Backlog

| # | Item | Priority | Effort | Status |
|---|---|---|---|---|
| B1 | Implement `aegisctl analyze` command | P0 | L | `TODO:` |
| B2 | Implement `aegisctl pack` command | P0 | L | `TODO:` |
| B3 | Implement `aegisctl score` command | P0 | M | `TODO:` |
| B4 | Implement `aegisctl migrate-aws` command | P1 | M | `TODO:` |
| B5 | Secrets pattern database (regex patterns for common secret formats) | P0 | M | `TODO:` |
| B6 | Bicep module library (App Service, Key Vault, App Config, Monitor) | P1 | L | `TODO:` |
| B7 | Interactive mode for `aegisctl pack` (prompt-based configuration) | P2 | M | `TODO:` |
| B8 | HTML/PDF output format for scorecard | P2 | S | `TODO:` |
| B9 | VS Code extension for inline findings | P3 | L | `TODO:` |
| B10 | Multi-workload / monorepo support | P2 | L | `TODO:` |

---

## Glossary

| Term | Definition |
|---|---|
| **WAF** | Azure Well-Architected Framework (NOT Web Application Firewall) |
| **CAF** | Azure Cloud Adoption Framework |
| **MI** | Managed Identity |
| **IaC** | Infrastructure as Code |
| **SKU** | Stock Keeping Unit (Azure pricing tier) |
