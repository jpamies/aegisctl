// Package packer orchestrates the generation of a complete architecture pack
// from repository analysis findings.
package packer

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/aegis/aegisctl/internal/analyzer"
	"github.com/aegis/aegisctl/internal/migrator"
	"github.com/aegis/aegisctl/internal/output"
	"github.com/aegis/aegisctl/internal/scorer"
)

// Config holds pack generation configuration.
type Config struct {
	RepoPath        string
	OutputDir       string
	DeployMode      string // "off", "manual", "auto"
	IncludeIaC      bool
	IncludePipeline bool
	IncludeAWS      bool
}

// templateFuncs returns the common template function map.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"inc": func(i int) int { return i + 1 },
		"relyOnCompute": func(f *analyzer.Findings) string {
			if f.HasDocker {
				return "Container Apps (built-in)"
			}
			return "App Service (built-in)"
		},
		"deployStatus": func(f *analyzer.Findings) string {
			if f.CI.HasGitHubActions {
				return "Detected (review gates)"
			}
			return "Recommended"
		},
	}
}

// Pack generates the full architecture pack.
func Pack(cfg Config) error {
	// 1. Analyze the repository
	findings, err := analyzer.Analyze(cfg.RepoPath)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// 2. Generate docs
	if err := generateDocs(findings, cfg); err != nil {
		return fmt.Errorf("generating docs: %w", err)
	}

	// 3. Generate IaC
	if cfg.IncludeIaC {
		if err := generateIaC(findings, cfg); err != nil {
			return fmt.Errorf("generating IaC: %w", err)
		}
	}

	// 4. Generate workflows
	if cfg.IncludePipeline {
		if err := generateWorkflows(cfg); err != nil {
			return fmt.Errorf("generating workflows: %w", err)
		}
	}

	// 5. Generate AWS migration report (if signals exist and enabled)
	if cfg.IncludeAWS && len(findings.AWSHints) > 0 {
		if err := migrator.MigrateAWS(cfg.RepoPath, cfg.OutputDir); err != nil {
			return fmt.Errorf("generating AWS migration: %w", err)
		}
	}

	// 6. Generate scorecard
	if err := scorer.Score(cfg.RepoPath, cfg.OutputDir); err != nil {
		return fmt.Errorf("generating scorecard: %w", err)
	}

	return nil
}

func generateDocs(f *analyzer.Findings, cfg Config) error {
	funcs := templateFuncs()

	// Architecture doc
	archContent, err := renderWithFuncs("architecture", output.ArchitectureDocTmpl, f, funcs)
	if err != nil {
		return err
	}
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "docs", "ARCHITECTURE.md"), archContent); err != nil {
		return err
	}

	// Security doc
	secContent, err := renderWithFuncs("security", output.SecurityDocTmpl, f, funcs)
	if err != nil {
		return err
	}
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "docs", "SECURITY_AND_SECRETS.md"), secContent); err != nil {
		return err
	}

	// WAF Checklist
	checkContent, err := renderWithFuncs("checklist", output.WAFChecklistTmpl, f, funcs)
	if err != nil {
		return err
	}
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, "docs", "WAF_CHECKLIST.md"), checkContent); err != nil {
		return err
	}

	return nil
}

func generateIaC(f *analyzer.Findings, cfg Config) error {
	// Determine Linux FX version from detected language
	linuxFx := "DOCKER|mcr.microsoft.com/appsvc/staticsite:latest"
	for _, l := range f.Languages {
		switch l.Language {
		case "Go":
			linuxFx = "GO|1.22"
		case "JavaScript/TypeScript":
			linuxFx = "NODE|20-lts"
		case "Python":
			linuxFx = "PYTHON|3.12"
		case "Java":
			linuxFx = "JAVA|17-java17"
		case "C#/.NET":
			linuxFx = "DOTNETCORE|8.0"
		}
		break // use first detected
	}

	data := struct {
		RepoPath       string
		DeployMode     string
		LinuxFxVersion string
	}{
		RepoPath:       cfg.RepoPath,
		DeployMode:     cfg.DeployMode,
		LinuxFxVersion: linuxFx,
	}

	bicepContent, err := output.RenderTemplate("bicep", output.BicepMainTmpl, data)
	if err != nil {
		return err
	}
	return output.WriteFile(filepath.Join(cfg.OutputDir, "infra", "main.bicep"), bicepContent)
}

func generateWorkflows(cfg Config) error {
	deployData := struct {
		DeployMode string
	}{
		DeployMode: cfg.DeployMode,
	}

	// CI workflow
	ciContent, err := output.RenderTemplate("ci", output.CIWorkflowTmpl, nil)
	if err != nil {
		return err
	}
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, ".github", "workflows", "ci.yml"), ciContent); err != nil {
		return err
	}

	// IaC validate workflow
	iacContent, err := output.RenderTemplate("iac-validate", output.IaCValidateWorkflowTmpl, nil)
	if err != nil {
		return err
	}
	if err := output.WriteFile(filepath.Join(cfg.OutputDir, ".github", "workflows", "iac-validate.yml"), iacContent); err != nil {
		return err
	}

	// Deploy workflow
	deployContent, err := output.RenderTemplate("deploy", output.DeployWorkflowTmpl, deployData)
	if err != nil {
		return err
	}
	return output.WriteFile(filepath.Join(cfg.OutputDir, ".github", "workflows", "deploy.yml"), deployContent)
}

// renderWithFuncs renders a template with custom functions.
func renderWithFuncs(name, tmplStr string, data interface{}, funcs template.FuncMap) (string, error) {
	t, err := template.New(name).Funcs(funcs).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %s: %w", name, err)
	}
	return buf.String(), nil
}
