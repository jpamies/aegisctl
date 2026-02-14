package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyze_DeterministicFileOrder(t *testing.T) {
	// Create a temp repo with files in various orders
	dir := t.TempDir()

	files := []string{
		"z_last.go",
		"a_first.go",
		"m_middle.py",
		"b_second.js",
	}

	for _, f := range files {
		path := filepath.Join(dir, f)
		if err := os.WriteFile(path, []byte("// placeholder\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Run analysis multiple times — output must be identical
	var firstFiles []string
	for i := 0; i < 5; i++ {
		findings, err := Analyze(dir)
		if err != nil {
			t.Fatal(err)
		}

		if i == 0 {
			firstFiles = findings.Files
		} else {
			if len(findings.Files) != len(firstFiles) {
				t.Fatalf("run %d: file count mismatch: %d vs %d", i, len(findings.Files), len(firstFiles))
			}
			for j := range findings.Files {
				if findings.Files[j] != firstFiles[j] {
					t.Errorf("run %d: file[%d] = %q, want %q", i, j, findings.Files[j], firstFiles[j])
				}
			}
		}
	}

	// Files should be sorted alphabetically
	for i := 1; i < len(firstFiles); i++ {
		if firstFiles[i] < firstFiles[i-1] {
			t.Errorf("files not sorted: %q before %q", firstFiles[i-1], firstFiles[i])
		}
	}
}

func TestAnalyze_LanguageDetection(t *testing.T) {
	dir := t.TempDir()

	// Create a go.mod and a .py file
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "app.py"), []byte("import os\n"), 0644)

	findings, err := Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(findings.Languages) < 2 {
		t.Fatalf("expected at least 2 languages, got %d", len(findings.Languages))
	}

	langSet := map[string]bool{}
	for _, l := range findings.Languages {
		langSet[l.Language] = true
	}

	if !langSet["Go"] {
		t.Error("Go not detected")
	}
	if !langSet["Python"] {
		t.Error("Python not detected")
	}
}

func TestAnalyze_DockerDetection(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM golang:1.22\n"), 0644)

	findings, err := Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !findings.HasDocker {
		t.Error("expected Docker to be detected")
	}
	if len(findings.Dockerfiles) != 1 {
		t.Errorf("expected 1 Dockerfile, got %d", len(findings.Dockerfiles))
	}
}

func TestAnalyze_CIDetection(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	os.MkdirAll(wfDir, 0755)
	os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: CI\n"), 0644)

	findings, err := Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !findings.CI.HasGitHubActions {
		t.Error("expected GitHub Actions to be detected")
	}
	if len(findings.CI.Workflows) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(findings.CI.Workflows))
	}
}

func TestAnalyze_InvalidPath(t *testing.T) {
	_, err := Analyze("/nonexistent/path/12345")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestAnalyze_AWSDetection(t *testing.T) {
	dir := t.TempDir()
	content := `import boto3

s3 = boto3.client('s3')
dynamodb = boto3.resource('dynamodb')
`
	os.WriteFile(filepath.Join(dir, "app.py"), []byte(content), 0644)

	findings, err := Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(findings.AWSHints) == 0 {
		t.Error("expected AWS hints to be detected")
	}

	services := map[string]bool{}
	for _, h := range findings.AWSHints {
		services[h.Service] = true
	}

	if !services["AWS SDK (Python/boto3)"] {
		t.Error("boto3 not detected")
	}
}

func TestAnalyze_DeterministicLanguageOrder(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "app.py"), []byte("pass\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	os.WriteFile(filepath.Join(dir, "index.js"), []byte("//\n"), 0644)

	var firstLangs []string
	for i := 0; i < 5; i++ {
		findings, err := Analyze(dir)
		if err != nil {
			t.Fatal(err)
		}
		var langs []string
		for _, l := range findings.Languages {
			langs = append(langs, l.Language)
		}
		if i == 0 {
			firstLangs = langs
		} else {
			for j := range langs {
				if langs[j] != firstLangs[j] {
					t.Errorf("run %d: lang[%d] = %q, want %q", i, j, langs[j], firstLangs[j])
				}
			}
		}
	}
}
