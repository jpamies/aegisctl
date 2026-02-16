package recommend

import (
	"testing"

	"github.com/jpamies/aegisctl/internal/analyzer"
	"github.com/jpamies/aegisctl/internal/state"
)

func TestEnrichHeuristic_Go(t *testing.T) {
	findings := &analyzer.Findings{
		RepoPath:  "/tmp/test",
		FileCount: 10,
		Languages: []analyzer.LanguageHint{
			{Language: "Go", Evidence: "go.mod found"},
		},
		HasDocker: true,
		Dockerfiles: []string{"Dockerfile"},
		CI: analyzer.CIInfo{
			HasGitHubActions: true,
			Workflows:        []string{"ci.yml", "deploy.yml", "iac.yml"},
		},
		IaC: analyzer.IaCInfo{HasBicep: true},
	}

	enriched, err := EnrichAnalysis(nil, findings)
	if err != nil {
		t.Fatalf("EnrichAnalysis: %v", err)
	}

	if enriched.AppType != "web-api" {
		t.Errorf("app_type: got %q, want web-api", enriched.AppType)
	}
	if enriched.PrimaryLanguage != "Go" {
		t.Errorf("primary_language: got %q, want Go", enriched.PrimaryLanguage)
	}
	if enriched.RecommendedRuntime != "GO|1.22" {
		t.Errorf("recommended_runtime: got %q, want GO|1.22", enriched.RecommendedRuntime)
	}
	if !enriched.IsContainerized {
		t.Error("is_containerized should be true")
	}
	if enriched.ExistingIaCQuality != "good" {
		t.Errorf("iac_quality: got %q, want good", enriched.ExistingIaCQuality)
	}
	if enriched.CICDQuality != "good" {
		t.Errorf("cicd_quality: got %q, want good", enriched.CICDQuality)
	}
}

func TestEnrichHeuristic_NoDcocker(t *testing.T) {
	findings := &analyzer.Findings{
		RepoPath:  "/tmp/test",
		FileCount: 5,
		Languages: []analyzer.LanguageHint{
			{Language: "Python", Evidence: "requirements.txt"},
		},
		HasDocker: false,
	}

	enriched, err := EnrichAnalysis(nil, findings)
	if err != nil {
		t.Fatalf("EnrichAnalysis: %v", err)
	}

	if enriched.AppType != "web-app" {
		t.Errorf("app_type: got %q, want web-app", enriched.AppType)
	}
	if enriched.IsContainerized {
		t.Error("is_containerized should be false")
	}
	if enriched.RecommendedRuntime != "PYTHON|3.12" {
		t.Errorf("recommended_runtime: got %q, want PYTHON|3.12", enriched.RecommendedRuntime)
	}
}

func TestEnrichHeuristic_AWSMigration(t *testing.T) {
	findings := &analyzer.Findings{
		RepoPath: "/tmp/test",
		AWSHints: []analyzer.AWSHint{
			{Service: "S3", File: "main.go", Line: 10},
			{Service: "Lambda", File: "handler.go", Line: 5},
			{Service: "S3", File: "upload.go", Line: 20}, // duplicate S3
		},
	}

	enriched, err := EnrichAnalysis(nil, findings)
	if err != nil {
		t.Fatalf("EnrichAnalysis: %v", err)
	}

	if !enriched.AWSMigrationNeeded {
		t.Error("aws_migration_needed should be true")
	}
	if len(enriched.AWSServicesFound) != 2 {
		t.Errorf("expected 2 unique AWS services, got %d: %v", len(enriched.AWSServicesFound), enriched.AWSServicesFound)
	}
}

func TestGeneratePlanHeuristic(t *testing.T) {
	s := &state.AnalysisState{
		Version:  "test",
		RepoPath: "/tmp/test",
		Heuristic: state.HeuristicFindings{
			HasDocker:   true,
			Dockerfiles: []string{"Dockerfile"},
			Languages:   []state.Language{{Language: "Go", Evidence: "go.mod"}},
		},
		Enriched: &state.EnrichedAnalysis{
			AppType:         "web-api",
			PrimaryLanguage: "Go",
		},
	}

	plan, err := GeneratePlan(nil, s)
	if err != nil {
		t.Fatalf("GeneratePlan: %v", err)
	}

	if plan.Mode != "heuristic" {
		t.Errorf("mode: got %q, want heuristic", plan.Mode)
	}
	if len(plan.Options) != 3 {
		t.Fatalf("expected 3 options, got %d", len(plan.Options))
	}

	// First option should be recommended and use Container Apps (because HasDocker)
	if !plan.Options[0].Recommended {
		t.Error("option[0] should be recommended")
	}
	if plan.Options[0].Compute.Service != "Container Apps" {
		t.Errorf("option[0] compute: got %q, want Container Apps", plan.Options[0].Compute.Service)
	}

	// Third option should be AKS
	if plan.Options[2].Compute.Service != "Azure Kubernetes Service" {
		t.Errorf("option[2] compute: got %q, want AKS", plan.Options[2].Compute.Service)
	}

	// All options should have WAF scores
	for i, opt := range plan.Options {
		if opt.WAFScores.Security == 0 {
			t.Errorf("option[%d] security score should not be 0", i)
		}
	}
}

func TestGeneratePlanHeuristic_NoDocker(t *testing.T) {
	s := &state.AnalysisState{
		RepoPath: "/tmp/test",
		Heuristic: state.HeuristicFindings{
			HasDocker: false,
		},
		Enriched: &state.EnrichedAnalysis{AppType: "web-app"},
	}

	plan, _ := GeneratePlan(nil, s)

	if plan.Options[0].Compute.Service != "App Service" {
		t.Errorf("without Docker, should recommend App Service, got %q", plan.Options[0].Compute.Service)
	}
}

func TestGeneratePlanHeuristic_SecretsRemediation(t *testing.T) {
	s := &state.AnalysisState{
		RepoPath: "/tmp/test",
		Heuristic: state.HeuristicFindings{
			Secrets: []state.SecretFinding{
				{File: "config.go", Line: 10, Type: "API Key", Severity: "HIGH", Remediation: "Use Key Vault"},
			},
		},
		Enriched: &state.EnrichedAnalysis{},
	}

	plan, _ := GeneratePlan(nil, s)

	if len(plan.SecretsRemediation) != 1 {
		t.Fatalf("expected 1 remediation, got %d", len(plan.SecretsRemediation))
	}
	if plan.SecretsRemediation[0].Steps == nil || len(plan.SecretsRemediation[0].Steps) == 0 {
		t.Error("remediation should have steps")
	}
}
