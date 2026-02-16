# Azure DevOps Pipelines

Pipeline definitions for building, testing, validating, deploying, and releasing aegisctl.

## Pipelines

| File | Purpose | Trigger |
|---|---|---|
| `ci.yml` | Build, test, lint, cross-compile | Push to main/develop, PRs |
| `iac-validate.yml` | Bicep lint & build | Changes to `infra/` |
| `deploy.yml` | Approval-gated Azure deployment | Manual (with parameters) |
| `release.yml` | Cross-compile + publish GitHub Release | Tags matching `v*` |

## Setup

### 1. Create service connections

In Azure DevOps → Project Settings → Service connections:

- **Azure Resource Manager** — for `AzureCLI@2` tasks (named in `AZURE_SERVICE_CONNECTION` variable)
- **GitHub** — for `GitHubRelease@1` task (named `github-release`)

### 2. Configure pipeline variables

| Variable | Description | Used by |
|---|---|---|
| `AZURE_SERVICE_CONNECTION` | Name of the Azure service connection | iac-validate, deploy |
| `AZURE_RESOURCE_GROUP` | Base resource group name (e.g., `rg-aegis`) | deploy |

### 3. Configure environments

In Azure DevOps → Pipelines → Environments, create:
- `dev`
- `staging`
- `production`

Add approval gates to `staging` and `production` environments.

### 4. Import pipelines

In Azure DevOps → Pipelines → New pipeline → Azure Repos Git (or GitHub) → Existing YAML:
- Point to `pipelines/ci.yml`
- Point to `pipelines/iac-validate.yml`
- Point to `pipelines/deploy.yml`
- Point to `pipelines/release.yml`

## Release workflow

1. Tag a commit: `git tag v0.3.0 && git push --tags`
2. The `release.yml` pipeline triggers automatically
3. Binaries are cross-compiled for Linux, macOS, and Windows (amd64 + arm64)
4. A GitHub Release is created with all binaries + SHA-256 checksums
5. Users download from GitHub Releases: `https://github.com/<org>/aegisctl/releases`
