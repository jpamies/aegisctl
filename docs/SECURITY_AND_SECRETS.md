# Security & Secrets — Findings and Remediation

> This document captures secrets/configuration findings from `aegisctl analyze`
> and provides remediation patterns aligned to Azure best practices.

---

## Remediation hierarchy

Aegis follows a strict remediation priority order:

```
1. Managed Identity (preferred — no secrets at all)
       ↓
2. Key Vault (for secrets that cannot use Managed Identity)
       ↓
3. App Configuration (for non-secret configuration values)
```

### Why this order?

| Approach | When to use | Benefit |
|---|---|---|
| **Managed Identity** | Azure-to-Azure service auth (SQL, Storage, Key Vault, etc.) | Zero secrets to manage, rotate, or leak. |
| **Key Vault** | Third-party API keys, legacy connection strings, certificates | Centralised, audited, auto-rotatable. |
| **App Configuration** | Feature flags, environment URLs, non-secret settings | Centralised config without secret exposure risk. |

> **Rule:** If a value is not a secret, it does **not** belong in Key Vault.
> Use App Configuration instead.

---

## Findings template

> Populated by `aegisctl analyze`. Template shown below for manual use.

### Finding #`TODO: N`

| Field | Value |
|---|---|
| **File** | `TODO: path/to/file` |
| **Line** | `TODO: line number` |
| **Type** | `TODO: connection-string / api-key / password / token / certificate` |
| **Severity** | `TODO: critical / high / medium / low` |
| **Current state** | `TODO: hardcoded / env-var / config-file` |
| **Recommended remediation** | `TODO: managed-identity / key-vault / app-configuration` |

**Description:**
`TODO: What was found and why it is a risk.`

**Remediation steps:**

1. `TODO: Step 1`
2. `TODO: Step 2`
3. `TODO: Step 3`

---

## Common remediation patterns

### Pattern A: Replace hardcoded connection string with Managed Identity

**Before (insecure):**
```go
// ❌ INSECURE — hardcoded connection string
connStr := "Server=myserver.database.windows.net;Database=mydb;User Id=REDACTED;Password=REDACTED;"
```

**After (Managed Identity):**
```go
// ✅ SECURE — uses DefaultAzureCredential (Managed Identity in Azure, CLI locally)
// TODO: import azidentity and azcore from Azure SDK for Go
cred, err := azidentity.NewDefaultAzureCredential(nil)
// Use token-based auth — no password needed
```

**Bicep change:**
```bicep
// Assign SQL Server the managed identity of the App Service
resource sqlRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  // TODO: specify role definition ID for Azure SQL DB Contributor
  scope: sqlServer
  properties: {
    principalId: appService.identity.principalId
    principalType: 'ServicePrincipal'
  }
}
```

### Pattern B: Move API key to Key Vault

**Before (insecure):**
```yaml
# ❌ INSECURE — API key in config file
third_party_api_key: "REDACTED"
```

**After (Key Vault reference):**
```yaml
# ✅ SECURE — reference to Key Vault secret
third_party_api_key: "@Microsoft.KeyVault(SecretUri=https://REDACTED.vault.azure.net/secrets/third-party-api-key)"
```

**Bicep change:**
```bicep
resource keyVault 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: 'kv-${workloadName}-${environment}'
  location: location
  properties: {
    sku: { family: 'A', name: 'standard' }
    tenantId: subscription().tenantId
    enableRbacAuthorization: true
    // TODO: configure access policies or RBAC roles
  }
}
```

### Pattern C: Move non-secret config to App Configuration

**Before:**
```env
# ❌ Mixed secrets and config in .env
DATABASE_URL=REDACTED
FEATURE_FLAG_NEW_UI=true
API_BASE_URL=https://api.example.com
```

**After:**
```
# .env only references (no secrets)
# Secrets → Managed Identity or Key Vault
# Config  → App Configuration

AZURE_APP_CONFIG_ENDPOINT=https://REDACTED.azconfig.io
```

**Bicep change:**
```bicep
resource appConfig 'Microsoft.AppConfiguration/configurationStores@2023-03-01' = {
  name: 'appcs-${workloadName}-${environment}'
  location: location
  sku: { name: 'free' }  // Use 'standard' for production
  // TODO: add key-value pairs for non-secret configuration
}
```

---

## CI/CD integration notes

- **Pre-commit hooks**: Consider adding `detect-secrets` or `gitleaks` to catch
  secrets before they reach the repository.
- **GitHub Actions**: Use `github.com/gitleaks/gitleaks-action` in `ci.yml` as
  a secrets scanning step.
- **Key Vault in pipelines**: Use `Azure/login` + `Azure/get-keyvault-secrets`
  actions to inject secrets at deploy time — never store them in workflow files.

---

## References

- [Managed identities for Azure resources](https://learn.microsoft.com/azure/active-directory/managed-identities-azure-resources/overview)
- [Azure Key Vault](https://learn.microsoft.com/azure/key-vault/general/overview)
- [Azure App Configuration](https://learn.microsoft.com/azure/azure-app-configuration/overview)
- [Key Vault references in App Service](https://learn.microsoft.com/azure/app-service/app-service-key-vault-references)
