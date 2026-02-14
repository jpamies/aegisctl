# Responsible AI Notes

> Transparency, guardrails, and ethical considerations for Aegis.

---

## What Aegis does

Aegis is an **advisory tool** that generates architecture documentation,
checklists, scorecards, IaC scaffolds, and migration assessments. It analyses
repository content (code, configuration, IaC files) and produces recommendations
based on the Azure Well-Architected Framework (WAF).

## What Aegis does NOT do

- **Does not deploy infrastructure** unless the user explicitly opts in to
  autodeploy mode — and even then, deployment requires human approval via
  GitHub Environment gates.
- **Does not access production systems** — analysis is performed against
  local repository content only.
- **Does not make security guarantees** — findings are heuristic and should
  be validated by a security professional.
- **Does not replace official Azure assessments** — the WAF scorecard is
  heuristic and non-official. Use the
  [Azure Well-Architected Review](https://learn.microsoft.com/assessments/azure-architecture-review/)
  for an official evaluation.

---

## Guardrails

### 1. Generate-only by default

Aegis defaults to `--deploy-mode=generate`. No Azure resources are created,
modified, or deleted unless the user explicitly changes the deploy mode.

### 2. Approval-gated deployment

When autodeploy is enabled:
- A GitHub Environment approval gate is **always** required.
- Deployment uses `Incremental` mode only — never `Complete` (which could
  delete resources not in the template).
- A `what-if` preview is generated before any deployment for human review.

### 3. No secrets in output

Aegis replaces any detected secrets with `REDACTED` in generated artefacts.
Generated Bicep templates reference Key Vault secrets or Managed Identity —
they never contain plaintext credentials.

### 4. Heuristic scoring transparency

The WAF scorecard uses a **0–5 heuristic scale** based on automated checks.
Scores are:
- Not endorsed by Microsoft.
- Not a substitute for a formal Well-Architected Review.
- Subject to false positives and false negatives.
- Intended as a starting point for improvement, not a certification.

### 5. AWS migration is best-effort

AWS→Azure service mappings are advisory. They:
- Do not guarantee functional equivalence.
- Require validation by the engineering team.
- Are based on common patterns, not exhaustive analysis.

---

## Data handling

| Data type | How Aegis handles it |
|---|---|
| Source code | Read-only, local analysis. Never transmitted externally. |
| Detected secrets | Flagged in findings, replaced with `REDACTED` in output. |
| IaC templates | Analysed locally. Generated Bicep is output to disk. |
| Repository metadata | Used for scoring. Not stored beyond the session. |

---

## Bias and limitations

- **Language bias**: Analysis quality may vary by programming language. Go and
  Python are best supported; other languages may have lower detection rates.
- **Azure bias**: Aegis is designed for Azure. Recommendations assume Azure as
  the target cloud. Multi-cloud or non-Azure workloads may not be well served.
- **Recency**: Aegis recommendations are based on Azure service availability
  and best practices at the time of the tool's last update. Azure evolves
  rapidly — always cross-reference with current documentation.
- **Scope limitation**: Aegis analyses the repository. It does not have
  visibility into runtime behaviour, network topology, or Azure portal
  configuration unless represented in IaC.

---

## Human oversight

Aegis is designed for **human-in-the-loop** operation:

1. **Review before acting** — all generated artefacts should be reviewed by
   the team before adoption.
2. **Approve before deploying** — environment gates ensure a human approves
   every deployment.
3. **Validate scores** — heuristic scores should be discussed, not taken at
   face value.
4. **Iterate** — run Aegis after implementing improvements to track progress.

---

## Feedback

If Aegis produces incorrect, misleading, or harmful output, please file an
issue in this repository with the `rai-concern` label.

---

## References

- [Microsoft Responsible AI principles](https://www.microsoft.com/ai/responsible-ai)
- [Azure Well-Architected Framework](https://learn.microsoft.com/azure/well-architected/)
- [GitHub Copilot Trust Center](https://resources.github.com/copilot-trust-center/)
