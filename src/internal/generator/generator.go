// Package generator produces the final artefacts (Bicep, pipelines, docs)
// from an aegisctl plan. Uses GitHub Copilot when available for adaptive
// generation, with template-based fallback for offline use.
package generator

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aegis/aegisctl/internal/copilot"
	"github.com/aegis/aegisctl/internal/output"
	"github.com/aegis/aegisctl/internal/state"
)

// Config holds apply-time configuration.
type Config struct {
	OutputDir  string
	DeployMode string // "off", "manual", "auto"
}

// Generate produces all artefacts for the selected plan option.
// If client is nil, falls back to template-based generation.
func Generate(client *copilot.Client, s *state.AnalysisState, p *state.Plan, cfg Config) error {
	if cfg.DeployMode == "" {
		cfg.DeployMode = "off"
	}

	option := p.Options[p.SelectedOption]

	fmt.Printf("  Generating artefacts for: %s\n", option.Name)

	// 1. Generate Bicep IaC
	if err := generateBicep(client, s, &option, cfg); err != nil {
		return fmt.Errorf("generating Bicep: %w", err)
	}

	// 2. Generate Azure DevOps pipelines
	if err := generatePipelines(client, s, &option, cfg); err != nil {
		return fmt.Errorf("generating pipelines: %w", err)
	}

	// 3. Generate documentation
	if err := generateDocs(client, s, p, cfg); err != nil {
		return fmt.Errorf("generating docs: %w", err)
	}

	return nil
}

// --- Bicep generation ---

func generateBicep(client *copilot.Client, s *state.AnalysisState, option *state.ArchOption, cfg Config) error {
	var bicepContent string

	if client != nil {
		content, err := generateBicepWithCopilot(client, option)
		if err != nil {
			fmt.Printf("  ⚠ Copilot Bicep generation failed, using template: %v\n", err)
			bicepContent = generateBicepFromTemplate(s, option, cfg)
		} else {
			bicepContent = content
		}
	} else {
		bicepContent = generateBicepFromTemplate(s, option, cfg)
	}

	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "infra", "main.bicep"), bicepContent); err != nil {
		return err
	}

	// Generate parameter files
	if err := generateParameterFiles(option, cfg); err != nil {
		return err
	}

	fmt.Println("  ✓ infra/main.bicep")
	fmt.Println("  ✓ infra/parameters.dev.json")
	fmt.Println("  ✓ infra/parameters.prod.json")
	return nil
}

func generateBicepWithCopilot(client *copilot.Client, option *state.ArchOption) (string, error) {
	optionJSON, _ := json.MarshalIndent(option, "", "  ")
	prompt := fmt.Sprintf(copilot.BicepPrompt, string(optionJSON))

	messages := []copilot.Message{
		{Role: "system", Content: copilot.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	content, err := client.ChatWithOptions(messages, 8192, 0.1)
	if err != nil {
		return "", err
	}

	// Strip markdown fences if present
	content = strings.TrimPrefix(content, "```bicep\n")
	content = strings.TrimPrefix(content, "```\n")
	if idx := strings.LastIndex(content, "```"); idx >= 0 {
		content = content[:idx]
	}

	return strings.TrimSpace(content), nil
}

func generateBicepFromTemplate(s *state.AnalysisState, option *state.ArchOption, cfg Config) string {
	runtime := "DOCKER|mcr.microsoft.com/appsvc/staticsite:latest"
	if s.Enriched != nil && s.Enriched.RecommendedRuntime != "" {
		runtime = s.Enriched.RecommendedRuntime
	}

	data := struct {
		RepoPath       string
		DeployMode     string
		LinuxFxVersion string
	}{
		RepoPath:       s.RepoPath,
		DeployMode:     cfg.DeployMode,
		LinuxFxVersion: runtime,
	}

	content, err := output.RenderTemplate("bicep", output.BicepMainTmpl, data)
	if err != nil {
		return "// Template rendering failed: " + err.Error()
	}
	return content
}

func generateParameterFiles(option *state.ArchOption, cfg Config) error {
	devParams := map[string]interface{}{
		"$schema":        "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
		"contentVersion": "1.0.0.0",
		"parameters": map[string]interface{}{
			"workloadName": map[string]string{"value": "aegis"},
			"environment":  map[string]string{"value": "dev"},
		},
	}
	prodParams := map[string]interface{}{
		"$schema":        "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
		"contentVersion": "1.0.0.0",
		"parameters": map[string]interface{}{
			"workloadName": map[string]string{"value": "aegis"},
			"environment":  map[string]string{"value": "prod"},
		},
	}

	devJSON, _ := json.MarshalIndent(devParams, "", "  ")
	prodJSON, _ := json.MarshalIndent(prodParams, "", "  ")

	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "infra", "parameters.dev.json"), string(devJSON)); err != nil {
		return err
	}
	return output.WriteFile(filepath.Join(cfg.OutputDir, "infra", "parameters.prod.json"), string(prodJSON))
}

// --- Pipeline generation ---

func generatePipelines(client *copilot.Client, s *state.AnalysisState, option *state.ArchOption, cfg Config) error {
	if client != nil {
		err := generatePipelinesWithCopilot(client, s, option, cfg)
		if err != nil {
			fmt.Printf("  ⚠ Copilot pipeline generation failed, using templates: %v\n", err)
			return generatePipelinesFromTemplates(s, option, cfg)
		}
		return nil
	}
	return generatePipelinesFromTemplates(s, option, cfg)
}

func generatePipelinesWithCopilot(client *copilot.Client, s *state.AnalysisState, option *state.ArchOption, cfg Config) error {
	analysisJSON, _ := json.MarshalIndent(s, "", "  ")
	optionJSON, _ := json.MarshalIndent(option, "", "  ")

	prompt := fmt.Sprintf(copilot.PipelinePrompt, string(analysisJSON), string(optionJSON))

	messages := []copilot.Message{
		{Role: "system", Content: copilot.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	var result struct {
		Workflows []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"workflows"`
	}

	if err := client.ChatJSON(messages, &result); err != nil {
		return err
	}

	for _, wf := range result.Workflows {
		if err := output.WriteFile(filepath.Join(cfg.OutputDir, wf.Path), wf.Content); err != nil {
			return err
		}
		fmt.Printf("  ✓ %s\n", wf.Path)
	}

	return nil
}

func generatePipelinesFromTemplates(s *state.AnalysisState, option *state.ArchOption, cfg Config) error {
	deployData := struct{ DeployMode string }{DeployMode: cfg.DeployMode}

	// CI
	ciContent, err := output.RenderTemplate("ci", output.CIPipelineTmpl, nil)
	if err != nil {
		return err
	}
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "pipelines", "ci.yml"), ciContent); err != nil {
		return err
	}
	fmt.Println("  ✓ pipelines/ci.yml")

	// IaC validate
	iacContent, err := output.RenderTemplate("iac-validate", output.IaCValidatePipelineTmpl, nil)
	if err != nil {
		return err
	}
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "pipelines", "iac-validate.yml"), iacContent); err != nil {
		return err
	}
	fmt.Println("  ✓ pipelines/iac-validate.yml")

	// Deploy
	deployContent, err := output.RenderTemplate("deploy", output.DeployPipelineTmpl, deployData)
	if err != nil {
		return err
	}
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "pipelines", "deploy.yml"), deployContent); err != nil {
		return err
	}
	fmt.Println("  ✓ pipelines/deploy.yml")

	return nil
}

// --- Documentation generation ---

func generateDocs(client *copilot.Client, s *state.AnalysisState, p *state.Plan, cfg Config) error {
	option := p.Options[p.SelectedOption]

	// Architecture doc
	archContent := generateArchDoc(s, &option, p)
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "docs", "ARCHITECTURE.md"), archContent); err != nil {
		return err
	}
	fmt.Println("  ✓ docs/ARCHITECTURE.md")

	// Security doc
	secContent, err := generateSecurityDoc(client, s, p, cfg)
	if err != nil {
		return err
	}
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "docs", "SECURITY.md"), secContent); err != nil {
		return err
	}
	fmt.Println("  ✓ docs/SECURITY.md")

	// WAF Checklist
	wafContent := generateWAFChecklist(s, &option)
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "docs", "WAF_CHECKLIST.md"), wafContent); err != nil {
		return err
	}
	fmt.Println("  ✓ docs/WAF_CHECKLIST.md")

	return nil
}

// workloadSlug returns a short slug suitable for Azure resource naming.
// Derives from the repo path, falling back to "app".
func workloadSlug(repoPath string) string {
	base := filepath.Base(repoPath)
	if base == "." || base == "/" || base == "" {
		// Attempt to use the absolute path
		abs, err := filepath.Abs(repoPath)
		if err == nil {
			base = filepath.Base(abs)
		}
	}
	slug := strings.ToLower(base)
	// Keep only alphanumeric, collapse to max 12 chars
	var clean strings.Builder
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			clean.WriteRune(c)
			if clean.Len() >= 12 {
				break
			}
		}
	}
	if clean.Len() == 0 {
		return "app"
	}
	return clean.String()
}

// computeAbbrev returns a short abbreviation for the compute service.
func computeAbbrev(service string) string {
	s := strings.ToLower(service)
	switch {
	case strings.Contains(s, "container app"):
		return "ca"
	case strings.Contains(s, "app service"):
		return "app"
	case strings.Contains(s, "function"):
		return "func"
	case strings.Contains(s, "aks") || strings.Contains(s, "kubernetes"):
		return "aks"
	case strings.Contains(s, "static"):
		return "swa"
	default:
		return "app"
	}
}

func generateArchDoc(s *state.AnalysisState, option *state.ArchOption, p *state.Plan) string {
	slug := workloadSlug(s.RepoPath)
	compAbbr := computeAbbrev(option.Compute.Service)

	var b strings.Builder
	b.WriteString("# Architecture — " + option.Name + "\n\n")
	b.WriteString("> Generated by aegisctl. Based on analysis of: `" + s.RepoPath + "`\n\n")
	b.WriteString("> **WAF** = Azure Well-Architected Framework (not Web Application Firewall).\n\n")
	b.WriteString("> **Disclaimer:** WAF scores are heuristic and non-official — not endorsed by Microsoft.\n\n")
	b.WriteString("---\n\n")

	// ── Application Profile ──
	b.WriteString("## 1. Application Profile\n\n")
	if s.Enriched != nil {
		b.WriteString(fmt.Sprintf("| Property | Value |\n|---|---|\n"))
		b.WriteString(fmt.Sprintf("| Type | %s |\n", s.Enriched.AppType))
		b.WriteString(fmt.Sprintf("| Description | %s |\n", s.Enriched.AppDescription))
		b.WriteString(fmt.Sprintf("| Primary language | %s |\n", s.Enriched.PrimaryLanguage))
		b.WriteString(fmt.Sprintf("| Complexity | %s |\n", s.Enriched.Complexity))
		b.WriteString(fmt.Sprintf("| Containerized | %v |\n", s.Enriched.IsContainerized))
		if s.Enriched.NeedsDatabase {
			hints := "yes"
			if len(s.Enriched.DatabaseHints) > 0 {
				hints = strings.Join(s.Enriched.DatabaseHints, ", ")
			}
			b.WriteString(fmt.Sprintf("| Database | %s |\n", hints))
		}
		if s.Enriched.NeedsCache {
			b.WriteString("| Cache | yes |\n")
		}
		if s.Enriched.NeedsQueue {
			b.WriteString("| Queue / messaging | yes |\n")
		}
		if s.Enriched.NeedsStorage {
			b.WriteString("| Blob / file storage | yes |\n")
		}
	} else {
		b.WriteString(fmt.Sprintf("- **Files scanned:** %d\n", s.Heuristic.FileCount))
		b.WriteString(fmt.Sprintf("- **Containerized:** %v\n", s.Heuristic.HasDocker))
		if len(s.Heuristic.Languages) > 0 {
			names := make([]string, len(s.Heuristic.Languages))
			for i, l := range s.Heuristic.Languages {
				names[i] = l.Language
			}
			b.WriteString(fmt.Sprintf("- **Languages:** %s\n", strings.Join(names, ", ")))
		}
	}
	b.WriteString(fmt.Sprintf("\n- **Files scanned:** %d\n", s.Heuristic.FileCount))
	if len(s.Heuristic.AWSHints) > 0 {
		b.WriteString(fmt.Sprintf("- **AWS hints:** %d (migration assistance available)\n", len(s.Heuristic.AWSHints)))
	}
	b.WriteString("\n")

	// ── Architecture Diagram ──
	b.WriteString("## 2. Architecture Diagram\n\n")
	b.WriteString("```mermaid\nflowchart TB\n")
	b.WriteString("    subgraph rg[\"Resource Group: rg-" + slug + "-dev\"]\n")

	// Compute
	b.WriteString("        " + compAbbr + "[\"" + option.Compute.Service + "\\n" + compAbbr + "-" + slug + "\"]\n")

	// Security
	b.WriteString("        kv[\"Key Vault\\nkv-" + slug + "\"]\n")
	b.WriteString("        mi[\"Managed Identity\\nid-" + slug + "\"]\n")

	// Monitoring
	monSvc := option.Monitoring.Service
	if monSvc == "" {
		monSvc = "Application Insights"
	}
	b.WriteString("        mon[\"" + monSvc + "\\nlog-" + slug + "\"]\n")
	b.WriteString("        law[\"Log Analytics Workspace\\nlaw-" + slug + "\"]\n")

	// Data services
	for i, ds := range option.Data {
		id := fmt.Sprintf("data%d", i)
		dsAbbr := dataAbbrev(ds.Service)
		b.WriteString(fmt.Sprintf("        %s[\"%s\\n%s-%s\"]\n", id, ds.Service, dsAbbr, slug))
	}

	b.WriteString("    end\n\n")

	// External actors
	b.WriteString("    user([\"Users / Clients\"]) --> " + compAbbr + "\n")
	b.WriteString("    ado([\"Azure DevOps\nCI/CD\"]) -.-> " + compAbbr + "\n")

	// Connections
	b.WriteString("    " + compAbbr + " --> kv\n")
	b.WriteString("    " + compAbbr + " --> mon\n")
	b.WriteString("    mon --> law\n")
	b.WriteString("    mi -. \"authenticates\" .-> " + compAbbr + "\n")
	b.WriteString("    mi -. \"authenticates\" .-> kv\n")

	for i := range option.Data {
		id := fmt.Sprintf("data%d", i)
		b.WriteString("    " + compAbbr + " --> " + id + "\n")
		b.WriteString("    mi -. \"authenticates\" .-> " + id + "\n")
	}

	b.WriteString("```\n\n")

	// ── Resource Inventory ──
	b.WriteString("## 3. Azure Resource Inventory\n\n")
	b.WriteString("| Resource | Type | Name (dev) | Name (prod) | SKU (dev) | SKU (prod) | Purpose |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	b.WriteString(fmt.Sprintf("| Resource Group | `Microsoft.Resources/resourceGroups` | `rg-%s-dev` | `rg-%s-prod` | — | — | Logical container |\n", slug, slug))
	b.WriteString(fmt.Sprintf("| Compute | `%s` | `%s-%s-dev` | `%s-%s-prod` | %s | %s | %s |\n",
		computeResourceType(option.Compute.Service), compAbbr, slug, compAbbr, slug,
		option.Compute.SKU, option.Compute.SKUProd, option.Compute.Rationale))
	b.WriteString(fmt.Sprintf("| Key Vault | `Microsoft.KeyVault/vaults` | `kv-%s-dev` | `kv-%s-prod` | Standard | Standard | Secrets management |\n", slug, slug))
	b.WriteString(fmt.Sprintf("| Managed Identity | `Microsoft.ManagedIdentity/userAssignedIdentities` | `id-%s-dev` | `id-%s-prod` | — | — | Passwordless auth (RBAC) |\n", slug, slug))
	b.WriteString(fmt.Sprintf("| Log Analytics | `Microsoft.OperationalInsights/workspaces` | `law-%s-dev` | `law-%s-prod` | PerGB2018 | PerGB2018 | Central logging |\n", slug, slug))
	b.WriteString(fmt.Sprintf("| App Insights | `Microsoft.Insights/components` | `appi-%s-dev` | `appi-%s-prod` | — | — | APM + diagnostics |\n", slug, slug))

	for _, ds := range option.Data {
		dsAbbr := dataAbbrev(ds.Service)
		b.WriteString(fmt.Sprintf("| %s | `%s` | `%s-%s-dev` | `%s-%s-prod` | %s | %s | %s |\n",
			ds.Service, dataResourceType(ds.Service), dsAbbr, slug, dsAbbr, slug,
			ds.SKU, ds.SKU, ds.Purpose))
	}
	b.WriteString("\n")

	// ── Compute Details ──
	b.WriteString("## 4. Compute — " + option.Compute.Service + "\n\n")
	b.WriteString(option.Description + "\n\n")
	b.WriteString(fmt.Sprintf("| Property | Dev | Prod |\n|---|---|---|\n"))
	b.WriteString(fmt.Sprintf("| SKU | %s | %s |\n", option.Compute.SKU, option.Compute.SKUProd))
	b.WriteString(fmt.Sprintf("| Estimated monthly cost | %s | %s |\n", option.EstimatedCostDev, option.EstimatedCostProd))
	b.WriteString(fmt.Sprintf("| Rationale | %s | — |\n\n", option.Compute.Rationale))

	// ── Security Architecture ──
	b.WriteString("## 5. Security Architecture\n\n")
	b.WriteString("```mermaid\nflowchart LR\n")
	b.WriteString("    app[\"" + option.Compute.Service + "\"] -- \"Managed Identity\" --> kv[\"Key Vault\"]\n")
	b.WriteString("    app -- \"Managed Identity\" --> data[(\"Data Services\")]\n")
	b.WriteString("    ado[\"Azure DevOps\"] -- \"Service Connection\" --> azure[\"Azure RBAC\"]\n")
	b.WriteString("    azure --> app\n")
	b.WriteString("```\n\n")
	b.WriteString(fmt.Sprintf("| Layer | Choice |\n|---|---|\n"))
	b.WriteString(fmt.Sprintf("| Identity | %s |\n", option.Security.Identity))
	b.WriteString(fmt.Sprintf("| Secrets store | %s |\n", option.Security.SecretsStore))
	b.WriteString(fmt.Sprintf("| Configuration | %s |\n", option.Security.ConfigStore))
	if option.Networking.HTTPSOnly {
		b.WriteString("| Transport | HTTPS-only |\n")
	}
	if option.Networking.MinTLS != "" {
		b.WriteString(fmt.Sprintf("| Minimum TLS | %s |\n", option.Networking.MinTLS))
	}
	for _, a := range option.Security.Additional {
		b.WriteString(fmt.Sprintf("| Additional | %s |\n", a))
	}
	for _, a := range option.Networking.Additional {
		b.WriteString(fmt.Sprintf("| Networking | %s |\n", a))
	}
	b.WriteString("\n")

	// ── Data Services ──
	if len(option.Data) > 0 {
		b.WriteString("## 6. Data Services\n\n")
		b.WriteString("| Service | Name (dev) | SKU | Purpose |\n|---|---|---|---|\n")
		for _, ds := range option.Data {
			dsAbbr := dataAbbrev(ds.Service)
			b.WriteString(fmt.Sprintf("| %s | `%s-%s-dev` | %s | %s |\n",
				ds.Service, dsAbbr, slug, ds.SKU, ds.Purpose))
		}
		b.WriteString("\n")
	}

	// ── Monitoring & Observability ──
	b.WriteString("## 7. Monitoring & Observability\n\n")
	b.WriteString(fmt.Sprintf("- **APM:** %s (`appi-%s-dev`)\n", monSvc, slug))
	b.WriteString(fmt.Sprintf("- **Logs:** Log Analytics Workspace (`law-%s-dev`)\n", slug))
	for _, a := range option.Monitoring.Additional {
		b.WriteString(fmt.Sprintf("- %s\n", a))
	}
	b.WriteString("\n")

	// ── WAF Scores ──
	b.WriteString("## 8. WAF Scores\n\n")
	total := option.WAFScores.Reliability + option.WAFScores.Security +
		option.WAFScores.CostOptimization + option.WAFScores.OperationalExcellence +
		option.WAFScores.PerformanceEfficiency
	b.WriteString(fmt.Sprintf("**Overall: %d / 25**\n\n", total))
	b.WriteString("| Pillar | Score | Bar |\n|---|---|---|\n")
	writePillarRow(&b, "Reliability", option.WAFScores.Reliability)
	writePillarRow(&b, "Security", option.WAFScores.Security)
	writePillarRow(&b, "Cost Optimization", option.WAFScores.CostOptimization)
	writePillarRow(&b, "Operational Excellence", option.WAFScores.OperationalExcellence)
	writePillarRow(&b, "Performance Efficiency", option.WAFScores.PerformanceEfficiency)
	b.WriteString("\n> **Disclaimer:** Scores are heuristic and non-official — not endorsed by Microsoft.\n\n")

	// ── Estimated Cost ──
	b.WriteString("## 9. Estimated Monthly Cost\n\n")
	b.WriteString(fmt.Sprintf("| Environment | Estimate |\n|---|---|\n"))
	b.WriteString(fmt.Sprintf("| Dev / Test | %s |\n", option.EstimatedCostDev))
	b.WriteString(fmt.Sprintf("| Production | %s |\n\n", option.EstimatedCostProd))

	// ── CI/CD Pipeline (Azure DevOps) ──
	b.WriteString("## 10. CI/CD Pipeline (Azure DevOps)\n\n")
	b.WriteString("```mermaid\nflowchart LR\n")
	b.WriteString("    push[\"git push\"] --> ci[\"CI Pipeline\\nbuild + test + lint\"]\n")
	b.WriteString("    ci --> iac[\"IaC Validate\\nbicep build\"]\n")
	b.WriteString("    iac --> gate{\"Environment\\nApproval\"}\n")
	b.WriteString("    gate --> deploy[\"Deploy Pipeline\\naz deployment\"]\n")
	b.WriteString("    deploy --> release[\"Release Pipeline\\nGitHub Release\"]\n")
	b.WriteString("```\n\n")
	b.WriteString("Pipelines are Azure DevOps YAML pipelines. Releases are published to **GitHub Releases**.\n\n")
	if p.CICDRecommendation != nil {
		for _, wf := range p.CICDRecommendation.Workflows {
			b.WriteString(fmt.Sprintf("- **%s** (`%s`) — %s\n", wf.Name, wf.File, wf.Purpose))
		}
		if len(p.CICDRecommendation.ImprovementsOverExisting) > 0 {
			b.WriteString("\n**Improvements over existing CI/CD:**\n")
			for _, imp := range p.CICDRecommendation.ImprovementsOverExisting {
				b.WriteString(fmt.Sprintf("- %s\n", imp))
			}
		}
		b.WriteString("\n")
	}

	// ── Alternative Options ──
	if len(p.Options) > 1 {
		b.WriteString("## 11. Alternative Options Considered\n\n")
		for i, opt := range p.Options {
			if i == p.SelectedOption {
				continue
			}
			rec := ""
			if opt.Recommended {
				rec = " ⭐"
			}
			b.WriteString(fmt.Sprintf("### Option %d: %s%s\n\n", i+1, opt.Name, rec))
			b.WriteString(opt.Description + "\n\n")
			b.WriteString(fmt.Sprintf("- Compute: %s (%s / %s)\n", opt.Compute.Service, opt.Compute.SKU, opt.Compute.SKUProd))
			b.WriteString(fmt.Sprintf("- Cost: dev %s | prod %s\n", opt.EstimatedCostDev, opt.EstimatedCostProd))
			b.WriteString(fmt.Sprintf("- WAF: R=%d S=%d C=%d O=%d P=%d\n\n",
				opt.WAFScores.Reliability, opt.WAFScores.Security,
				opt.WAFScores.CostOptimization, opt.WAFScores.OperationalExcellence,
				opt.WAFScores.PerformanceEfficiency))
		}
	}

	// ── Migration Steps ──
	if len(option.MigrationSteps) > 0 {
		b.WriteString("## 12. Migration Steps\n\n")
		for i, step := range option.MigrationSteps {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
		b.WriteString("\n")
	}

	// ── Generated Files ──
	if len(option.FilesToGenerate) > 0 {
		b.WriteString("## 13. Generated Files\n\n")
		b.WriteString("| File | Description |\n|---|---|\n")
		for _, f := range option.FilesToGenerate {
			b.WriteString(fmt.Sprintf("| `%s` | %s |\n", f, fileDescription(f)))
		}
		b.WriteString("\n")
	}

	// ── References ──
	b.WriteString("---\n\n## References\n\n")
	b.WriteString("- [Azure Well-Architected Framework](https://learn.microsoft.com/azure/well-architected/)\n")
	b.WriteString("- [Cloud Adoption Framework](https://learn.microsoft.com/azure/cloud-adoption-framework/)\n")
	b.WriteString("- [Azure naming conventions](https://learn.microsoft.com/azure/cloud-adoption-framework/ready/azure-best-practices/resource-naming)\n")
	b.WriteString("- [Bicep documentation](https://learn.microsoft.com/azure/azure-resource-manager/bicep/)\n")
	b.WriteString("- [Azure DevOps Pipelines](https://learn.microsoft.com/azure/devops/pipelines/)\n")

	return b.String()
}

// writePillarRow writes a WAF pillar row with a visual bar.
func writePillarRow(b *strings.Builder, name string, score int) {
	bar := strings.Repeat("█", score) + strings.Repeat("░", 5-score)
	b.WriteString(fmt.Sprintf("| %s | %d/5 | %s |\n", name, score, bar))
}

// dataAbbrev returns Azure naming prefix for a data service.
func dataAbbrev(service string) string {
	s := strings.ToLower(service)
	switch {
	case strings.Contains(s, "cosmos"):
		return "cosmos"
	case strings.Contains(s, "sql") && !strings.Contains(s, "postgre") && !strings.Contains(s, "mysql"):
		return "sql"
	case strings.Contains(s, "postgre"):
		return "psql"
	case strings.Contains(s, "mysql"):
		return "mysql"
	case strings.Contains(s, "redis"):
		return "redis"
	case strings.Contains(s, "storage") || strings.Contains(s, "blob"):
		return "st"
	case strings.Contains(s, "service bus"):
		return "sb"
	case strings.Contains(s, "event hub"):
		return "evh"
	case strings.Contains(s, "event grid"):
		return "evg"
	default:
		return "svc"
	}
}

// dataResourceType returns the ARM resource type for a data service.
func dataResourceType(service string) string {
	s := strings.ToLower(service)
	switch {
	case strings.Contains(s, "cosmos"):
		return "Microsoft.DocumentDB/databaseAccounts"
	case strings.Contains(s, "sql") && !strings.Contains(s, "postgre") && !strings.Contains(s, "mysql"):
		return "Microsoft.Sql/servers"
	case strings.Contains(s, "postgre"):
		return "Microsoft.DBforPostgreSQL/flexibleServers"
	case strings.Contains(s, "mysql"):
		return "Microsoft.DBforMySQL/flexibleServers"
	case strings.Contains(s, "redis"):
		return "Microsoft.Cache/redis"
	case strings.Contains(s, "storage") || strings.Contains(s, "blob"):
		return "Microsoft.Storage/storageAccounts"
	case strings.Contains(s, "service bus"):
		return "Microsoft.ServiceBus/namespaces"
	case strings.Contains(s, "event hub"):
		return "Microsoft.EventHub/namespaces"
	default:
		return "Microsoft.*"
	}
}

// computeResourceType returns the ARM resource type for a compute service.
func computeResourceType(service string) string {
	s := strings.ToLower(service)
	switch {
	case strings.Contains(s, "container app"):
		return "Microsoft.App/containerApps"
	case strings.Contains(s, "app service"):
		return "Microsoft.Web/sites"
	case strings.Contains(s, "function"):
		return "Microsoft.Web/sites (functionapp)"
	case strings.Contains(s, "aks") || strings.Contains(s, "kubernetes"):
		return "Microsoft.ContainerService/managedClusters"
	case strings.Contains(s, "static"):
		return "Microsoft.Web/staticSites"
	default:
		return "Microsoft.Web/sites"
	}
}

// fileDescription returns a human-readable description for a generated file.
func fileDescription(path string) string {
	switch {
	case strings.HasSuffix(path, "main.bicep"):
		return "Azure Bicep IaC template"
	case strings.Contains(path, "parameters") && strings.Contains(path, "dev"):
		return "Dev environment parameters"
	case strings.Contains(path, "parameters") && strings.Contains(path, "prod"):
		return "Prod environment parameters"
	case strings.Contains(path, "ci.yml"):
		return "CI pipeline (build + test + lint)"
	case strings.Contains(path, "iac-validate"):
		return "IaC validation pipeline"
	case strings.Contains(path, "deploy"):
		return "Gated deployment pipeline"
	case strings.Contains(path, "ARCHITECTURE"):
		return "Architecture decisions + WAF scores"
	case strings.Contains(path, "SECURITY"):
		return "Security posture + remediation"
	case strings.Contains(path, "WAF"):
		return "WAF pillar checklist"
	default:
		return "Generated artefact"
	}
}

func generateSecurityDoc(client *copilot.Client, s *state.AnalysisState, p *state.Plan, cfg Config) (string, error) {
	if client != nil {
		analysisJSON, _ := json.MarshalIndent(s, "", "  ")
		remediationJSON, _ := json.MarshalIndent(p.SecretsRemediation, "", "  ")
		prompt := fmt.Sprintf(copilot.SecurityDocPrompt, string(analysisJSON), string(remediationJSON))

		messages := []copilot.Message{
			{Role: "system", Content: copilot.SystemPrompt},
			{Role: "user", Content: prompt},
		}

		content, err := client.ChatWithOptions(messages, 4096, 0.2)
		if err == nil {
			return content, nil
		}
		fmt.Printf("  ⚠ Copilot security doc failed, using template: %v\n", err)
	}

	// Fallback: simple template
	var b strings.Builder
	b.WriteString("# Security & Secrets — Findings and Remediation\n\n")
	b.WriteString("> Generated by aegisctl. Repository: " + s.RepoPath + "\n\n")
	b.WriteString("## Remediation Hierarchy\n\n")
	b.WriteString("1. **Managed Identity** — eliminate credentials entirely (preferred).\n")
	b.WriteString("2. **Azure Key Vault** — for secrets that cannot use Managed Identity.\n")
	b.WriteString("3. **Azure App Configuration** — for non-secret configuration values.\n\n")

	if len(p.SecretsRemediation) > 0 {
		b.WriteString(fmt.Sprintf("## Findings (%d)\n\n", len(p.SecretsRemediation)))
		for i, r := range p.SecretsRemediation {
			b.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, r.Current))
			b.WriteString(fmt.Sprintf("**Recommended:** %s\n\n", r.Recommended))
			b.WriteString("Steps:\n")
			for _, step := range r.Steps {
				b.WriteString(fmt.Sprintf("- %s\n", step))
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("## Findings\n\nNo secrets or credentials detected.\n\n")
	}

	return b.String(), nil
}

func generateWAFChecklist(s *state.AnalysisState, option *state.ArchOption) string {
	var b strings.Builder
	b.WriteString("# WAF Checklist\n\n")
	b.WriteString("> Generated by aegisctl. Repository: " + s.RepoPath + "\n\n")
	b.WriteString("> **WAF** = Azure Well-Architected Framework (not Web Application Firewall).\n\n")
	b.WriteString("> **Disclaimer:** This checklist is heuristic. Scores are not endorsed by Microsoft.\n\n")

	pillars := []struct {
		name   string
		score  int
		checks []string
	}{
		{"Reliability", option.WAFScores.Reliability, []string{
			"Health endpoints configured",
			"Auto-restart / self-healing enabled",
			"Retry policies for external calls",
			"Data backup strategy defined",
			"Disaster recovery plan documented",
		}},
		{"Security", option.WAFScores.Security, []string{
			"Managed Identity enabled",
			"No hardcoded secrets",
			"Key Vault for residual secrets",
			"HTTPS-only enforced",
			"TLS 1.2+ minimum",
			"RBAC least-privilege",
		}},
		{"Cost Optimization", option.WAFScores.CostOptimization, []string{
			"Minimum-viable SKU selected",
			"Consumption-based pricing where possible",
			"Auto-shutdown for dev/test",
			"Cost alerts configured",
		}},
		{"Operational Excellence", option.WAFScores.OperationalExcellence, []string{
			"IaC for all resources",
			"CI pipeline (build + test)",
			"CD pipeline (gated deploy)",
			"Environment approvals",
			"Monitoring and alerting",
		}},
		{"Performance Efficiency", option.WAFScores.PerformanceEfficiency, []string{
			"Autoscale configured",
			"CDN for static assets",
			"Caching strategy",
			"Load testing plan",
		}},
	}

	for _, p := range pillars {
		b.WriteString(fmt.Sprintf("## %s (%d/5)\n\n", p.name, p.score))
		b.WriteString("| # | Check | Status |\n|---|---|---|\n")
		for i, check := range p.checks {
			b.WriteString(fmt.Sprintf("| %d | %s | TODO |\n", i+1, check))
		}
		b.WriteString("\n")
	}

	return b.String()
}
