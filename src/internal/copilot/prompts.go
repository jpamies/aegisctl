package copilot

// This file contains prompt templates used across the init/plan/apply pipeline.
// Each prompt is designed to produce structured JSON output from the LLM.

// SystemPrompt is the base system prompt establishing the assistant's role.
const SystemPrompt = `You are Aegis, an expert Azure Solutions Architect assistant.
You follow the Azure Well-Architected Framework (WAF) and Cloud Adoption Framework (CAF).
WAF always means Well-Architected Framework (not Web Application Firewall).
You recommend minimum-cost, production-ready Azure architectures.
You always prefer:
- Managed Identity over credentials
- Key Vault for secrets that cannot use Managed Identity
- App Configuration for non-secret settings
- Bicep for IaC (never ARM JSON or Terraform)
- GitHub Actions for CI/CD
- PaaS over IaaS when possible
- Consumption/serverless pricing when applicable
- Scale-to-zero for dev/test environments

You produce structured JSON output when asked. No markdown, no explanation — only valid JSON.`

// AnalyzePrompt is used during "init" to enrich heuristic findings with deeper analysis.
const AnalyzePrompt = `Analyze the following repository scan results and provide an enriched assessment.

Repository scan (heuristic):
%s

Based on these findings, produce a JSON object with:
{
  "app_type": "web-api|web-app|static-site|worker|function|cli|monorepo|unknown",
  "app_description": "one-line description of what this app likely does",
  "primary_language": "main language/runtime",
  "recommended_runtime": "specific runtime version for Azure (e.g., NODE|20-lts, DOTNETCORE|8.0, GO|1.22, PYTHON|3.12)",
  "needs_database": true/false,
  "database_hints": ["list of database signals found"],
  "needs_cache": true/false,
  "needs_queue": true/false,
  "needs_storage": true/false,
  "is_containerized": true/false,
  "container_ready": true/false,
  "has_existing_iac": true/false,
  "existing_iac_quality": "none|basic|good|production-ready",
  "iac_issues": ["list of issues with existing IaC if any"],
  "has_existing_cicd": true/false,
  "cicd_quality": "none|basic|good|production-ready",
  "cicd_issues": ["list of issues with existing CI/CD if any"],
  "security_concerns": ["list of security issues beyond secrets"],
  "aws_migration_needed": true/false,
  "aws_services_detected": ["list of AWS services to migrate"],
  "complexity": "simple|moderate|complex"
}

Respond with ONLY the JSON object, no other text.`

// RecommendPrompt is used during "plan" to generate architecture options.
const RecommendPrompt = `Based on the following enriched analysis of a repository, generate Azure architecture recommendations.

Enriched analysis:
%s

Generate exactly 3 architecture options ranked by fit, from most to least recommended.
Each option should follow Azure WAF principles and target minimum cost.

Produce a JSON object:
{
  "options": [
    {
      "name": "short name",
      "description": "one paragraph explaining the architecture",
      "recommended": true/false,
      "compute": {
        "service": "Container Apps|App Service|Azure Functions|Static Web Apps|AKS",
        "sku": "specific SKU for dev",
        "sku_prod": "specific SKU for prod",
        "rationale": "why this compute choice"
      },
      "data": [
        {
          "service": "Azure service name",
          "purpose": "what it's used for",
          "sku": "specific SKU"
        }
      ],
      "security": {
        "identity": "System-assigned Managed Identity",
        "secrets_store": "Key Vault",
        "config_store": "App Configuration",
        "additional": ["other security recommendations"]
      },
      "monitoring": {
        "service": "Application Insights + Log Analytics",
        "additional": ["other monitoring recommendations"]
      },
      "networking": {
        "https_only": true,
        "min_tls": "1.2",
        "additional": ["other networking recommendations"]
      },
      "estimated_monthly_cost_dev": "$X/month estimate",
      "estimated_monthly_cost_prod": "$X/month estimate",
      "waf_scores": {
        "reliability": 0-5,
        "security": 0-5,
        "cost_optimization": 0-5,
        "operational_excellence": 0-5,
        "performance_efficiency": 0-5
      },
      "migration_steps": ["ordered steps if AWS migration is needed"],
      "files_to_generate": ["list of file paths that would be created"]
    }
  ],
  "secrets_remediation": [
    {
      "current": "description of current secret issue",
      "recommended": "Azure service/pattern to use instead",
      "steps": ["migration steps"]
    }
  ],
  "cicd_recommendation": {
    "workflows": [
      {
        "name": "workflow name",
        "file": "file path",
        "purpose": "what it does",
        "triggers": ["push to main", "pull_request", etc.]
      }
    ],
    "improvements_over_existing": ["list of improvements if CI/CD already exists"]
  }
}

Respond with ONLY the JSON object, no other text.`

// BicepPrompt is used during "apply" to generate Bicep IaC.
const BicepPrompt = `Generate a production-ready Bicep template for the following Azure architecture.

Architecture plan:
%s

Requirements:
- targetScope = 'resourceGroup'
- Use parameters for: workloadName, environment (dev/staging/prod), location
- SKU selection must vary by environment (minimum cost for dev, production-ready for prod)
- Include system-assigned Managed Identity on compute resources
- Include Key Vault with RBAC authorization
- Include App Configuration if needed
- Include Application Insights + Log Analytics workspace
- Include proper resource tags
- Deploy mode: Incremental ONLY (never Complete)
- Follow Azure naming conventions (e.g., app-, kv-, log-, appi-)
- Add TODO comments for items that need manual configuration
- Output key resource names and endpoints

Respond with ONLY the Bicep code, no markdown fences, no explanation.`

// PipelinePrompt is used during "apply" to generate Azure DevOps pipeline YAML.
const PipelinePrompt = `Generate GitHub Actions workflow YAML for the following application and architecture.

Application details:
%s

Architecture plan:
%s

Generate a JSON object with the workflow files:
{
  "workflows": [
    {
      "path": ".github/workflows/ci.yml",
      "content": "full YAML content"
    },
    {
      "path": ".github/workflows/iac-validate.yml",
      "content": "full YAML content"
    },
    {
      "path": ".github/workflows/deploy.yml",
      "content": "full YAML content"
    }
  ]
}

Requirements for CI workflow:
- Trigger on push to main/develop and pull_request to main
- Use 'runs-on: ubuntu-latest'
- Build and test the application
- Use correct language/runtime setup (actions/setup-go, actions/setup-node, actions/setup-python, etc.)
- Include linting and formatting checks
- Upload build artefacts with actions/upload-artifact@v4

Requirements for IaC validation workflow:
- Trigger on changes to infra/ directory
- Validate Bicep with az bicep build

Requirements for Deploy workflow:
- Manual trigger (workflow_dispatch) with inputs for environment and dry_run
- Use OIDC federation (azure/login@v2 with client-id, tenant-id, subscription-id)
- Require GitHub Environment approval
- Run what-if preview before deploy
- Deploy with mode Incremental ONLY (never Complete)
- Use stages: Preview → Deploy
- Deployment stage uses condition on dryRun parameter

Respond with ONLY the JSON object, no other text.`

// SecurityDocPrompt generates security guidance adapted to the specific findings.
const SecurityDocPrompt = `Generate a security remediation document for the following repository findings.

Analysis:
%s

Secrets remediation plan:
%s

Generate a Markdown document (raw markdown, no code fences) that includes:
1. Executive summary of security posture
2. Remediation hierarchy (Managed Identity → Key Vault → App Configuration)
3. Specific remediation steps for each finding, with code examples where applicable
4. Azure RBAC role assignments needed
5. CI/CD security recommendations (OIDC federation, GitHub secret scanning, etc.)
6. Compliance considerations

Be specific to the actual findings — do not generate generic content.
Respond with ONLY the Markdown content.`
