# Demo Script — 3-Minute Walkthrough

> Use this script for live demos or recorded walkthroughs of Aegis. Target
> duration: **3 minutes**.

---

## Setup (before demo)

1. Have a sample application repository ready (e.g., a Node.js or Go web app
   with a few intentional security findings).
2. Ensure `aegisctl` is built: `go build -o bin/aegisctl ./src/cmd/aegisctl`
3. Have a terminal open in the sample repo directory.
4. (Optional) Have VS Code open with the sample repo for visual context.

---

## Script

### Slide 0 — Title (10 seconds)

> _"Aegis is an SRE/DevOps tool that analyses your application code and
> generates a complete Azure architecture pack — aligned to the
> Well-Architected Framework, with minimum-cost defaults."_

---

### Act 1 — Analyze (40 seconds)

```bash
aegisctl analyze --repo .
```

> _"First, we scan the repository. Aegis detects the application language,
> identifies cloud service dependencies, finds hardcoded secrets, and maps
> the current architecture."_

**Show:** Terminal output with detected languages, services, and secret findings.

> _"We found 3 hardcoded secrets and 2 AWS services that need Azure
> equivalents. Let's generate the full architecture pack."_

---

### Act 2 — Pack (60 seconds)

```bash
aegisctl pack --repo . --output ./output
```

> _"The pack command generates everything: architecture docs, WAF checklist,
> Bicep templates, GitHub Actions workflows, and a security remediation
> plan."_

**Show:** File tree of the `./output` directory.

> _"Notice the Bicep templates use minimum-cost SKUs — Free tier for dev,
> Standard for production. Secrets use Managed Identity first, Key Vault
> only when MI isn't possible."_

**Show:** Open `output/docs/SECURITY_AND_SECRETS.md` — highlight the finding
and remediation guidance (MI → Key Vault → App Config pattern).

---

### Act 3 — Score (50 seconds)

```bash
aegisctl score --repo .
```

> _"Now let's see how this repo scores against the Well-Architected Framework."_

**Show:** WAF Scorecard output — table with 0–5 scores per pillar.

> _"Security scores a 2 because of the hardcoded secrets. Cost Optimization
> scores a 3 — the architecture uses PaaS but hasn't configured autoscale yet.
> And here are the top 5 improvements — prioritised actions to raise the
> score."_

**Show:** "Top 5 Improvements" table.

> _"Important: these scores are heuristic — they're a starting point, not a
> certification. For an official review, use the Azure Well-Architected Review
> tool."_

---

### Act 4 — Deploy (optional, 20 seconds)

> _"By default, Aegis only generates — it never deploys. But if you're ready,
> you can trigger the deploy workflow manually."_

**Show:** GitHub Actions → `deploy.yml` → `workflow_dispatch` button.

> _"Deployment always requires a GitHub Environment approval. No unattended
> production changes, ever."_

---

### Closing (10 seconds)

> _"Aegis: analyse, pack, score, deploy — all aligned to the Azure
> Well-Architected Framework, with minimum-cost defaults and a 'do no harm'
> philosophy."_

---

## Key talking points

- **Generate-only by default** — safe for any environment.
- **WAF-aligned** — every decision maps to a WAF pillar.
- **Minimum cost** — Free/Basic for dev, Standard for prod, consumption where
  possible.
- **Secrets remediation** — Managed Identity first, then Key Vault, then App
  Config.
- **Heuristic scoring** — actionable but explicitly non-official.
- **Approval-gated deploy** — human-in-the-loop always.
