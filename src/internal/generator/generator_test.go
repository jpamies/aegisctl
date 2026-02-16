package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aegis/aegisctl/internal/state"
)

func testState() *state.AnalysisState {
	return &state.AnalysisState{
		Version:  "test",
		RepoPath: "/tmp/test",
		Mode:     "heuristic",
		Heuristic: state.HeuristicFindings{
			FileCount: 10,
			HasDocker: true,
			Languages: []state.Language{{Language: "Go", Evidence: "go.mod"}},
			CI:        state.CIInfo{HasGitHubActions: true},
			IaC:       state.IaCInfo{HasBicep: true},
		},
		Enriched: &state.EnrichedAnalysis{
			AppType:            "web-api",
			PrimaryLanguage:    "Go",
			RecommendedRuntime: "GO|1.22",
			Complexity:         "moderate",
		},
	}
}

func testPlan() *state.Plan {
	return &state.Plan{
		Version:        "test",
		RepoPath:       "/tmp/test",
		SelectedOption: 0,
		Mode:           "heuristic",
		Options: []state.ArchOption{
			{
				Name:        "Container Apps + Key Vault",
				Description: "Scale-to-zero container hosting.",
				Recommended: true,
				Compute:     state.ComputeChoice{Service: "Container Apps", SKU: "Consumption", SKUProd: "Consumption", Rationale: "Best fit"},
				Security:    state.SecurityChoice{Identity: "Managed Identity", SecretsStore: "Key Vault", ConfigStore: "App Configuration", Additional: []string{"HTTPS"}},
				Monitoring:  state.MonitoringChoice{Service: "App Insights"},
				Networking:  state.NetworkingChoice{HTTPSOnly: true, MinTLS: "1.2"},
				EstimatedCostDev:  "$0-5/month",
				EstimatedCostProd: "$50-100/month",
				WAFScores:   state.WAFScores{Reliability: 3, Security: 4, CostOptimization: 4, OperationalExcellence: 3, PerformanceEfficiency: 3},
				FilesToGenerate: []string{"infra/main.bicep"},
			},
		},
		SecretsRemediation: []state.SecretsRemediationItem{
			{Current: "API Key in code", Recommended: "Key Vault", Steps: []string{"Remove", "Store in KV"}},
		},
	}
}

func TestGenerate_CreatesFiles(t *testing.T) {
	outDir := t.TempDir()

	s := testState()
	p := testPlan()

	cfg := Config{
		OutputDir:  outDir,
		DeployMode: "off",
	}

	if err := Generate(nil, s, p, cfg); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Check that key files were created
	expectedFiles := []string{
		filepath.Join("infra", "main.bicep"),
		filepath.Join("infra", "parameters.dev.json"),
		filepath.Join("infra", "parameters.prod.json"),
		filepath.Join("pipelines", "ci.yml"),
		filepath.Join("pipelines", "iac-validate.yml"),
		filepath.Join("pipelines", "deploy.yml"),
		filepath.Join("docs", "ARCHITECTURE.md"),
		filepath.Join("docs", "SECURITY.md"),
		filepath.Join("docs", "WAF_CHECKLIST.md"),
	}

	for _, f := range expectedFiles {
		path := filepath.Join(outDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s not found: %v", f, err)
			continue
		}
		content, _ := os.ReadFile(path)
		if len(content) == 0 {
			t.Errorf("file %s is empty", f)
		}
	}
}

func TestGenerate_BicepContent(t *testing.T) {
	outDir := t.TempDir()

	if err := Generate(nil, testState(), testPlan(), Config{OutputDir: outDir, DeployMode: "off"}); err != nil {
		t.Fatal(err)
	}

	bicep, _ := os.ReadFile(filepath.Join(outDir, "infra", "main.bicep"))
	content := string(bicep)

	// Should contain key Bicep elements
	checks := []string{
		"targetScope",
		"param workloadName",
		"param environment",
		"resource",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("Bicep should contain %q", check)
		}
	}
}

func TestGenerate_ArchDocContent(t *testing.T) {
	outDir := t.TempDir()

	if err := Generate(nil, testState(), testPlan(), Config{OutputDir: outDir}); err != nil {
		t.Fatal(err)
	}

	doc, _ := os.ReadFile(filepath.Join(outDir, "docs", "ARCHITECTURE.md"))
	content := string(doc)

	if !strings.Contains(content, "Container Apps") {
		t.Error("Architecture doc should mention the selected compute service")
	}
	if !strings.Contains(content, "WAF Scores") {
		t.Error("Architecture doc should include WAF scores")
	}
	if !strings.Contains(content, "Disclaimer") {
		t.Error("Architecture doc should include WAF disclaimer")
	}
}

func TestGenerate_SecurityDocContent(t *testing.T) {
	outDir := t.TempDir()

	if err := Generate(nil, testState(), testPlan(), Config{OutputDir: outDir}); err != nil {
		t.Fatal(err)
	}

	doc, _ := os.ReadFile(filepath.Join(outDir, "docs", "SECURITY.md"))
	content := string(doc)

	if !strings.Contains(content, "Managed Identity") {
		t.Error("Security doc should mention Managed Identity")
	}
	if !strings.Contains(content, "API Key in code") {
		t.Error("Security doc should include the finding")
	}
}

func TestGenerate_WAFChecklistContent(t *testing.T) {
	outDir := t.TempDir()

	if err := Generate(nil, testState(), testPlan(), Config{OutputDir: outDir}); err != nil {
		t.Fatal(err)
	}

	doc, _ := os.ReadFile(filepath.Join(outDir, "docs", "WAF_CHECKLIST.md"))
	content := string(doc)

	if !strings.Contains(content, "Reliability") {
		t.Error("WAF checklist should include Reliability pillar")
	}
	if !strings.Contains(content, "Security") {
		t.Error("WAF checklist should include Security pillar")
	}
	if !strings.Contains(content, "not endorsed by Microsoft") {
		t.Error("WAF checklist should include disclaimer")
	}
}

func TestGenerate_DeployModeDefault(t *testing.T) {
	outDir := t.TempDir()

	// Empty deploy mode should default to "off"
	if err := Generate(nil, testState(), testPlan(), Config{OutputDir: outDir}); err != nil {
		t.Fatal(err)
	}

	deploy, _ := os.ReadFile(filepath.Join(outDir, "pipelines", "deploy.yml"))
	content := string(deploy)

	if !strings.Contains(content, "Deploy") {
		t.Error("deploy workflow should exist")
	}
}
