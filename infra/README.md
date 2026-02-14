# Infrastructure — Bicep Templates

> This directory contains Azure Bicep IaC templates generated or scaffolded by
> Aegis.

---

## Structure

```
infra/
├── main.bicep              ← Orchestration entry point
├── modules/                ← TODO: Reusable Bicep modules
│   ├── appService.bicep    ← TODO: App Service + Plan
│   ├── keyVault.bicep      ← TODO: Key Vault with RBAC
│   ├── appConfig.bicep     ← TODO: App Configuration store
│   ├── monitoring.bicep    ← TODO: App Insights + Log Analytics
│   └── identity.bicep      ← TODO: Managed Identity + role assignments
├── parameters.dev.json     ← Dev environment parameters
└── parameters.prod.json    ← Production environment parameters
```

## Usage

### Validate (no deployment)

```bash
# Lint
az bicep lint --file infra/main.bicep

# Build (compile to ARM JSON for validation)
az bicep build --file infra/main.bicep

# What-if (preview changes against Azure — requires auth)
az deployment group what-if \
  --resource-group rg-aegis-dev \
  --template-file infra/main.bicep \
  --parameters infra/parameters.dev.json
```

### Deploy (requires approval via GitHub Actions)

Deployments should be executed through the `deploy.yml` GitHub Actions workflow,
which enforces:
- Environment approval gates
- Incremental mode only (never Complete)
- What-if preview before apply

For local testing only:
```bash
# ⚠️ Use only for dev/test — production deploys go through CI/CD
az deployment group create \
  --resource-group rg-aegis-dev \
  --template-file infra/main.bicep \
  --parameters infra/parameters.dev.json \
  --mode Incremental
```

## Design principles

1. **Minimum-cost defaults** — Free/Basic SKUs for dev, Standard for production.
2. **Managed Identity first** — no stored credentials in templates.
3. **Incremental deployments only** — never use `Complete` mode.
4. **Parameterised** — environment-specific values in parameter files.
5. **Modular** — one module per resource type for reusability.

## References

- [Bicep documentation](https://learn.microsoft.com/azure/azure-resource-manager/bicep/)
- [Azure Verified Modules](https://aka.ms/avm)
- [Naming conventions](https://learn.microsoft.com/azure/cloud-adoption-framework/ready/azure-best-practices/resource-naming)
