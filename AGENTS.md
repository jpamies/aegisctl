# AGENTS.md — Aegis Agent Instructions

> This file provides instructions, guardrails, and policies for any AI agent
> (including GitHub Copilot, Copilot Chat, or custom agents) operating inside
> this repository.

---

## 1. Project identity

| Field | Value |
|---|---|
| Project name | **Aegis** |
| CLI binary | `aegisctl` |
| Language | Go (stdlib only) |
| Primary framework | Azure Well-Architected Framework (WAF) |
| Default behaviour | **Generate-only** — never deploy unless explicitly opted in |

## 2. Repository conventions

- **Go stdlib only** — do not introduce third-party Go modules.
- **Bicep** for all Azure IaC — no ARM JSON, no Terraform.
- **GitHub Actions** for CI/CD. Releases are published via **GitHub Releases**.
- No secrets, tokens, or credentials may be committed. Use `REDACTED` if an
  example is needed. Example values must be clearly labelled `<!-- EXAMPLE ONLY -->`.
- Terminology: **WAF** always means **Well-Architected Framework**. If you need
  to reference Web Application Firewall, spell it out in full.

## 3. Agent guardrails

### 3.1 Do-no-harm principle

1. **Never** generate code that deploys, mutates, or deletes Azure resources
   without the user explicitly requesting it AND confirming autodeploy mode.
2. **Never** embed real secrets — always use placeholders marked `REDACTED`.
3. **Never** produce ARM JSON. Always use Bicep.
4. **Never** assume the user wants to deploy. Default is generate-only.

### 3.2 File-safety rules

- Do not modify files outside this repository.
- Do not create files in `/infra` that contain `Microsoft.Resources/deployments`
  with mode `Complete` (risk of resource deletion).
- Always validate Bicep with `bicep build` before suggesting a deployment.

### 3.3 Scope boundaries

- Aegis outputs **advisory artefacts** (docs, checklists, IaC scaffolds).
  It is not a deployment orchestrator by default.
- The WAF scorecard is **heuristic and non-official**. Always include the
  disclaimer that scores are not endorsed by Microsoft.
- AWS migration mapping is **best-effort** and must be flagged as such.

## 4. CLI workflow (init / plan / apply)

aegisctl uses a Terraform-style three-step workflow:

```bash
aegisctl init <repoPath>                        # Scan + enrich → .aegis/state.json
aegisctl plan <repoPath>                        # Recommend options → .aegis/plan.json
aegisctl apply <repoPath> --output <dir>        # Write IaC + pipelines + docs
```

- **init** — heuristic repo scan + optional Copilot enrichment (requires `AEGIS_GITHUB_TOKEN` or `GITHUB_TOKEN`).
- **plan** — generates 3 architecture options with WAF scores; interactive selection.
- **apply** — produces Bicep, GitHub Actions workflows, and documentation.
- **analyze** — quick print-only analysis (no state).

### Deploy modes

| Mode | Flag | Approval required | Default |
|---|---|---|---|
| `off` | `--deploy off` | No | **Yes** |
| `manual` | `--deploy manual` | GitHub Environment approval | No |
| `auto` | `--deploy auto` | GitHub Environment approval | No |

> **Agents must never switch deploy mode on behalf of the user.**
> If a user asks to deploy, confirm the mode change explicitly.

### State directory

aegisctl persists state in `.aegis/` under the repo root:
- `.aegis/state.json` — analysis output (created by `init`)
- `.aegis/plan.json` — selected plan (created by `plan`)

Add `.aegis/` to `.gitignore`.

## 5. Scoring policy (WAF scorecard)

- Each WAF pillar is scored **0–5** using heuristic checks.
- Scores are **non-official** and must always carry a disclaimer.
- Scoring inputs:
  - Repository analysis (secrets, config, IaC presence).
  - Architecture alignment (managed services, redundancy, monitoring).
  - CI/CD maturity (tests, gates, approvals).
- A "Top 5 improvements" list must accompany every scorecard.

## 6. Interaction patterns

When an agent is asked to:

| Request | Correct response |
|---|---|
| "Analyze my repo" | Run `aegisctl init .` then `aegisctl plan .` |
| "Generate architecture" | Run `aegisctl init .` → `plan .` → `apply . --output out/` |
| "Deploy to Azure" | Confirm deploy mode, require environment approval |
| "Score my repo" | Run `aegisctl plan .`, review WAF scores in the plan options |
| "Migrate from AWS" | Run `aegisctl init .` → `plan .` (AWS hints auto-detected) |
| "Add a secret to config" | Refuse; recommend Managed Identity → Key Vault pattern |
| "Use Copilot" | Ensure `GITHUB_TOKEN` is set, then run `init` + `plan` |

## 7. References

- [Azure Well-Architected Framework](https://learn.microsoft.com/azure/well-architected/)
- [Azure Cloud Adoption Framework](https://learn.microsoft.com/azure/cloud-adoption-framework/)
- [Bicep documentation](https://learn.microsoft.com/azure/azure-resource-manager/bicep/)
- [GitHub Actions documentation](https://docs.github.com/actions)
