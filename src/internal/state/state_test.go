package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadState(t *testing.T) {
	dir := t.TempDir()

	s := &AnalysisState{
		Version:   "0.2.0",
		CreatedAt: time.Now().Truncate(time.Second),
		RepoPath:  dir,
		Mode:      "heuristic",
		Heuristic: HeuristicFindings{
			FileCount: 42,
			HasDocker: true,
			Dockerfiles: []string{"Dockerfile"},
			Languages: []Language{
				{Language: "Go", Evidence: "go.mod"},
			},
			CI: CIInfo{HasGitHubActions: true, Workflows: []string{"ci.yml"}},
			IaC: IaCInfo{HasBicep: true, BicepFiles: []string{"main.bicep"}},
		},
		Enriched: &EnrichedAnalysis{
			AppType:         "web-api",
			PrimaryLanguage: "Go",
			Complexity:      "moderate",
		},
	}

	if err := SaveState(dir, s); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// File should exist
	path := filepath.Join(dir, DirName, "state.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state.json not created: %v", err)
	}

	// Load back
	loaded, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	if loaded.Version != s.Version {
		t.Errorf("version: got %q, want %q", loaded.Version, s.Version)
	}
	if loaded.Mode != "heuristic" {
		t.Errorf("mode: got %q, want heuristic", loaded.Mode)
	}
	if loaded.Heuristic.FileCount != 42 {
		t.Errorf("file_count: got %d, want 42", loaded.Heuristic.FileCount)
	}
	if !loaded.Heuristic.HasDocker {
		t.Error("has_docker should be true")
	}
	if loaded.Enriched == nil {
		t.Fatal("enriched should not be nil")
	}
	if loaded.Enriched.AppType != "web-api" {
		t.Errorf("app_type: got %q, want web-api", loaded.Enriched.AppType)
	}
}

func TestStateExists(t *testing.T) {
	dir := t.TempDir()

	if StateExists(dir) {
		t.Error("StateExists should be false before init")
	}

	SaveState(dir, &AnalysisState{Version: "test", RepoPath: dir})

	if !StateExists(dir) {
		t.Error("StateExists should be true after SaveState")
	}
}

func TestSaveAndLoadPlan(t *testing.T) {
	dir := t.TempDir()

	p := &Plan{
		Version:        "0.2.0",
		CreatedAt:      time.Now().Truncate(time.Second),
		RepoPath:       dir,
		SelectedOption: 1,
		Mode:           "heuristic",
		Options: []ArchOption{
			{
				Name:        "App Service",
				Description: "Basic PaaS",
				Recommended: true,
				Compute:     ComputeChoice{Service: "App Service", SKU: "F1", SKUProd: "S1"},
				WAFScores:   WAFScores{Reliability: 3, Security: 4, CostOptimization: 4},
			},
			{
				Name:        "Container Apps",
				Description: "Scale to zero",
				Recommended: false,
				Compute:     ComputeChoice{Service: "Container Apps", SKU: "Consumption"},
			},
		},
		SecretsRemediation: []SecretsRemediationItem{
			{Current: "API Key in code", Recommended: "Key Vault", Steps: []string{"Step 1", "Step 2"}},
		},
	}

	if err := SavePlan(dir, p); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	if !PlanExists(dir) {
		t.Error("PlanExists should be true after SavePlan")
	}

	loaded, err := LoadPlan(dir)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}

	if loaded.SelectedOption != 1 {
		t.Errorf("selected_option: got %d, want 1", loaded.SelectedOption)
	}
	if len(loaded.Options) != 2 {
		t.Fatalf("options: got %d, want 2", len(loaded.Options))
	}
	if loaded.Options[0].Name != "App Service" {
		t.Errorf("option[0].name: got %q", loaded.Options[0].Name)
	}
	if loaded.Options[0].WAFScores.Security != 4 {
		t.Errorf("waf security: got %d, want 4", loaded.Options[0].WAFScores.Security)
	}
	if len(loaded.SecretsRemediation) != 1 {
		t.Fatalf("secrets_remediation: got %d", len(loaded.SecretsRemediation))
	}
}

func TestLoadState_NoFile(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadState(dir)
	if err == nil {
		t.Error("LoadState should fail when no state.json exists")
	}
}

func TestLoadPlan_NoFile(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadPlan(dir)
	if err == nil {
		t.Error("LoadPlan should fail when no plan.json exists")
	}
}
