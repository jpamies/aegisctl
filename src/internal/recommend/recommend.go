// Package recommend generates Azure architecture recommendations based on
// repository analysis. Uses GitHub Copilot (GitHub Models API) when available,
// with a heuristic fallback for offline use.
package recommend

import (
	"encoding/json"
	"fmt"

	"github.com/jpamies/aegisctl/internal/analyzer"
	"github.com/jpamies/aegisctl/internal/copilot"
	"github.com/jpamies/aegisctl/internal/state"
)

// EnrichAnalysis takes heuristic findings and enriches them with LLM analysis.
// If client is nil, falls back to heuristic enrichment.
func EnrichAnalysis(client *copilot.Client, findings *analyzer.Findings) (*state.EnrichedAnalysis, error) {
	if client == nil {
		return enrichHeuristic(findings), nil
	}
	return enrichWithCopilot(client, findings)
}

// GeneratePlan produces architecture options for the given state.
// If client is nil, falls back to heuristic recommendations.
func GeneratePlan(client *copilot.Client, s *state.AnalysisState) (*state.Plan, error) {
	if client == nil {
		return planHeuristic(s), nil
	}
	return planWithCopilot(client, s)
}

// --- Copilot-powered ---

func enrichWithCopilot(client *copilot.Client, findings *analyzer.Findings) (*state.EnrichedAnalysis, error) {
	// Serialize findings summary for the prompt
	summary := buildFindingsSummary(findings)
	prompt := fmt.Sprintf(copilot.AnalyzePrompt, summary)

	messages := []copilot.Message{
		{Role: "system", Content: copilot.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	var enriched state.EnrichedAnalysis
	if err := client.ChatJSON(messages, &enriched); err != nil {
		return nil, fmt.Errorf("copilot enrichment failed: %w", err)
	}

	return &enriched, nil
}

func planWithCopilot(client *copilot.Client, s *state.AnalysisState) (*state.Plan, error) {
	// Build full analysis context for the prompt
	analysisJSON, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("serializing state: %w", err)
	}

	prompt := fmt.Sprintf(copilot.RecommendPrompt, string(analysisJSON))

	messages := []copilot.Message{
		{Role: "system", Content: copilot.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	// The LLM returns a structure with options, secrets_remediation, cicd_recommendation
	var result struct {
		Options            []state.ArchOption              `json:"options"`
		SecretsRemediation []state.SecretsRemediationItem  `json:"secrets_remediation"`
		CICDRecommendation *state.CICDRecommendation       `json:"cicd_recommendation"`
	}

	if err := client.ChatJSON(messages, &result); err != nil {
		return nil, fmt.Errorf("copilot recommendation failed: %w", err)
	}

	// Find the recommended option index
	selected := 0
	for i, opt := range result.Options {
		if opt.Recommended {
			selected = i
			break
		}
	}

	plan := &state.Plan{
		RepoPath:           s.RepoPath,
		SelectedOption:     selected,
		Options:            result.Options,
		SecretsRemediation: result.SecretsRemediation,
		CICDRecommendation: result.CICDRecommendation,
		Mode:               "copilot",
	}

	return plan, nil
}

// --- Heuristic fallback ---

func enrichHeuristic(f *analyzer.Findings) *state.EnrichedAnalysis {
	e := &state.EnrichedAnalysis{
		Complexity: "moderate",
	}

	// App type
	if f.HasDocker {
		e.AppType = "web-api"
		e.IsContainerized = true
		e.ContainerReady = true
	} else {
		e.AppType = "web-app"
		e.IsContainerized = false
		e.ContainerReady = false
	}

	// Primary language
	if len(f.Languages) > 0 {
		e.PrimaryLanguage = f.Languages[0].Language
		switch f.Languages[0].Language {
		case "Go":
			e.RecommendedRuntime = "GO|1.22"
			e.AppDescription = "Go application"
		case "JavaScript/TypeScript":
			e.RecommendedRuntime = "NODE|20-lts"
			e.AppDescription = "Node.js application"
		case "Python":
			e.RecommendedRuntime = "PYTHON|3.12"
			e.AppDescription = "Python application"
		case "Java":
			e.RecommendedRuntime = "JAVA|17-java17"
			e.AppDescription = "Java application"
		case "C#/.NET":
			e.RecommendedRuntime = "DOTNETCORE|8.0"
			e.AppDescription = ".NET application"
		default:
			e.RecommendedRuntime = "DOCKER|mcr.microsoft.com/appsvc/staticsite:latest"
			e.AppDescription = "Application"
		}
	}

	// IaC
	e.HasExistingIaC = f.IaC.HasBicep || f.IaC.HasTerraform || f.IaC.HasARM || f.IaC.HasCloudFormation
	if f.IaC.HasBicep {
		e.ExistingIaCQuality = "good"
	} else if f.IaC.HasTerraform || f.IaC.HasARM {
		e.ExistingIaCQuality = "basic"
		e.IaCIssues = append(e.IaCIssues, "Migrate to Bicep recommended")
	} else {
		e.ExistingIaCQuality = "none"
	}

	// CI/CD
	e.HasExistingCICD = f.CI.HasGitHubActions || f.CI.HasOtherCI
	if f.CI.HasGitHubActions && len(f.CI.Workflows) >= 3 {
		e.CICDQuality = "good"
	} else if f.CI.HasGitHubActions {
		e.CICDQuality = "basic"
	} else {
		e.CICDQuality = "none"
	}

	// Security
	if len(f.Secrets) > 0 {
		for _, s := range f.Secrets {
			e.SecurityConcerns = append(e.SecurityConcerns, fmt.Sprintf("%s in %s (line %d)", s.Type, s.File, s.Line))
		}
	}

	// AWS
	e.AWSMigrationNeeded = len(f.AWSHints) > 0
	seen := map[string]bool{}
	for _, h := range f.AWSHints {
		if !seen[h.Service] {
			seen[h.Service] = true
			e.AWSServicesFound = append(e.AWSServicesFound, h.Service)
		}
	}

	return e
}

func planHeuristic(s *state.AnalysisState) *state.Plan {
	h := s.Heuristic
	e := s.Enriched

	// Determine compute type
	computeService := "App Service"
	skuDev := "F1"
	skuProd := "S1"
	computeRationale := "PaaS with built-in scaling and health checks."

	if h.HasDocker {
		computeService = "Container Apps"
		skuDev = "Consumption"
		skuProd = "Consumption"
		computeRationale = "Scale-to-zero container hosting, consumption pricing."
	}

	// Build options
	option1 := state.ArchOption{
		Name:        computeService + " + Key Vault + App Config",
		Description: fmt.Sprintf("Recommended: %s for compute with Managed Identity, Key Vault for secrets, App Configuration for settings.", computeService),
		Recommended: true,
		Compute: state.ComputeChoice{
			Service:   computeService,
			SKU:       skuDev,
			SKUProd:   skuProd,
			Rationale: computeRationale,
		},
		Security: state.SecurityChoice{
			Identity:     "System-assigned Managed Identity",
			SecretsStore: "Azure Key Vault",
			ConfigStore:  "Azure App Configuration",
			Additional:   []string{"HTTPS-only", "TLS 1.2 minimum", "RBAC least-privilege"},
		},
		Monitoring: state.MonitoringChoice{
			Service:    "Application Insights + Log Analytics",
			Additional: []string{"Health probes on compute", "Alert rules for errors"},
		},
		Networking: state.NetworkingChoice{
			HTTPSOnly:  true,
			MinTLS:     "1.2",
			Additional: []string{"FTPS disabled"},
		},
		EstimatedCostDev:  "~$0-5/month",
		EstimatedCostProd: "~$50-100/month",
		WAFScores: state.WAFScores{
			Reliability:           3,
			Security:              4,
			CostOptimization:      4,
			OperationalExcellence: 3,
			PerformanceEfficiency: 3,
		},
		FilesToGenerate: []string{
			"infra/main.bicep",
			"infra/parameters.dev.json",
			"infra/parameters.prod.json",
			".github/workflows/ci.yml",
			".github/workflows/iac-validate.yml",
			".github/workflows/deploy.yml",
			"docs/ARCHITECTURE.md",
			"docs/WAF_CHECKLIST.md",
			"docs/SECURITY.md",
		},
	}

	// Option 2: Azure Functions (if Lambda hints found)
	option2Name := "App Service (container)"
	option2Desc := "Alternative: App Service with Docker container deployment."
	option2Compute := "App Service"
	if e != nil && e.AWSMigrationNeeded {
		for _, svc := range e.AWSServicesFound {
			if svc == "Lambda" {
				option2Name = "Azure Functions (Consumption)"
				option2Desc = "Serverless: Azure Functions with consumption pricing. Good fit for Lambda migration."
				option2Compute = "Azure Functions"
				break
			}
		}
	}

	option2 := state.ArchOption{
		Name:        option2Name,
		Description: option2Desc,
		Recommended: false,
		Compute: state.ComputeChoice{
			Service:   option2Compute,
			SKU:       "Consumption",
			SKUProd:   "Consumption",
			Rationale: "Alternative compute option.",
		},
		Security:         option1.Security,
		Monitoring:       option1.Monitoring,
		Networking:       option1.Networking,
		EstimatedCostDev: "~$0-10/month",
		EstimatedCostProd: "~$30-80/month",
		WAFScores: state.WAFScores{
			Reliability:           3,
			Security:              4,
			CostOptimization:      4,
			OperationalExcellence: 3,
			PerformanceEfficiency: 3,
		},
		FilesToGenerate: option1.FilesToGenerate,
	}

	// Option 3: AKS (for complex cases)
	option3 := state.ArchOption{
		Name:        "AKS (Kubernetes)",
		Description: "Full Kubernetes: AKS with managed node pools. Higher cost but maximum flexibility.",
		Recommended: false,
		Compute: state.ComputeChoice{
			Service:   "Azure Kubernetes Service",
			SKU:       "Standard_B2s",
			SKUProd:   "Standard_D2s_v5",
			Rationale: "Maximum flexibility for complex microservices.",
		},
		Security:          option1.Security,
		Monitoring:        option1.Monitoring,
		Networking:        option1.Networking,
		EstimatedCostDev:  "~$50-100/month",
		EstimatedCostProd: "~$200-500/month",
		WAFScores: state.WAFScores{
			Reliability:           4,
			Security:              4,
			CostOptimization:      2,
			OperationalExcellence: 4,
			PerformanceEfficiency: 4,
		},
		FilesToGenerate: option1.FilesToGenerate,
	}

	// Secrets remediation
	var secretsRemediation []state.SecretsRemediationItem
	for _, sec := range h.Secrets {
		secretsRemediation = append(secretsRemediation, state.SecretsRemediationItem{
			Current:     fmt.Sprintf("%s in %s (line %d)", sec.Type, sec.File, sec.Line),
			Recommended: sec.Remediation,
			Steps:       []string{"Remove from source code", "Store in Key Vault", "Reference via Managed Identity"},
		})
	}

	// CICD recommendation
	cicd := &state.CICDRecommendation{
		Workflows: []state.WorkflowRecommendation{
			{Name: "CI", File: ".github/workflows/ci.yml", Purpose: "Build, test, and lint", Triggers: []string{"push to main", "pull_request"}},
			{Name: "IaC Validate", File: ".github/workflows/iac-validate.yml", Purpose: "Validate Bicep templates", Triggers: []string{"push to infra/", "pull_request"}},
			{Name: "Deploy", File: ".github/workflows/deploy.yml", Purpose: "Gated deployment to Azure", Triggers: []string{"workflow_dispatch"}},
		},
	}

	return &state.Plan{
		RepoPath:           s.RepoPath,
		SelectedOption:     0,
		Options:            []state.ArchOption{option1, option2, option3},
		SecretsRemediation: secretsRemediation,
		CICDRecommendation: cicd,
		Mode:               "heuristic",
	}
}

// --- Helpers ---

func buildFindingsSummary(f *analyzer.Findings) string {
	data, _ := json.MarshalIndent(struct {
		RepoPath  string                 `json:"repo_path"`
		FileCount int                    `json:"file_count"`
		Languages []analyzer.LanguageHint `json:"languages"`
		HasDocker bool                   `json:"has_docker"`
		CI        analyzer.CIInfo        `json:"ci"`
		IaC       analyzer.IaCInfo       `json:"iac"`
		Deps      []analyzer.DependencyHint `json:"deps"`
		AWSHints  []analyzer.AWSHint     `json:"aws_hints"`
		Secrets   int                    `json:"secret_count"`
	}{
		RepoPath:  f.RepoPath,
		FileCount: f.FileCount,
		Languages: f.Languages,
		HasDocker: f.HasDocker,
		CI:        f.CI,
		IaC:       f.IaC,
		Deps:      f.Deps,
		AWSHints:  f.AWSHints,
		Secrets:   len(f.Secrets),
	}, "", "  ")
	return string(data)
}
