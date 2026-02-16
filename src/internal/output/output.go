// Package output provides template rendering utilities for generating
// Markdown documents, Bicep templates, and workflow files from analysis data.
// Uses Go text/template exclusively — no external dependencies.
package output

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aegis/aegisctl/internal/analyzer"
)

// WriteFile writes content to a file, creating directories as needed.
func WriteFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// RenderTemplate renders a named Go text/template with the given data.
func RenderTemplate(name, tmpl string, data interface{}) (string, error) {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parsing template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %s: %w", name, err)
	}
	return buf.String(), nil
}

// FormatAnalysis renders a human-readable analysis report to stdout.
func FormatAnalysis(f *analyzer.Findings) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Repository Analysis: %s\n\n", f.RepoPath))
	b.WriteString(fmt.Sprintf("Files scanned: %d\n\n", f.FileCount))

	// Languages
	b.WriteString("## Languages Detected\n\n")
	if len(f.Languages) == 0 {
		b.WriteString("No languages detected.\n\n")
	} else {
		for _, l := range f.Languages {
			b.WriteString(fmt.Sprintf("- **%s** — %s\n", l.Language, l.Evidence))
		}
		b.WriteString("\n")
	}

	// Docker
	b.WriteString("## Container Support\n\n")
	if f.HasDocker {
		for _, d := range f.Dockerfiles {
			b.WriteString(fmt.Sprintf("- %s\n", d))
		}
	} else {
		b.WriteString("No Dockerfile detected.\n")
	}
	b.WriteString("\n")

	// CI
	b.WriteString("## CI/CD Configuration\n\n")
	if f.CI.HasGitHubActions {
		b.WriteString("CI/CD pipelines:\n")
		for _, w := range f.CI.Workflows {
			b.WriteString(fmt.Sprintf("- %s\n", w))
		}
	}
	if f.CI.HasOtherCI {
		b.WriteString(fmt.Sprintf("Other CI detected: %s\n", f.CI.OtherCI))
	}
	if !f.CI.HasGitHubActions && !f.CI.HasOtherCI {
		b.WriteString("No CI/CD configuration detected.\n")
	}
	b.WriteString("\n")

	// IaC
	b.WriteString("## Infrastructure as Code\n\n")
	if f.IaC.HasBicep {
		b.WriteString(fmt.Sprintf("Bicep files: %s\n", strings.Join(f.IaC.BicepFiles, ", ")))
	}
	if f.IaC.HasTerraform {
		b.WriteString(fmt.Sprintf("Terraform files: %s\n", strings.Join(f.IaC.TerraformFiles, ", ")))
	}
	if f.IaC.HasARM {
		b.WriteString(fmt.Sprintf("ARM templates: %s\n", strings.Join(f.IaC.ARMFiles, ", ")))
	}
	if f.IaC.HasCloudFormation {
		b.WriteString(fmt.Sprintf("CloudFormation templates: %s\n", strings.Join(f.IaC.CloudFormationFiles, ", ")))
	}
	if f.IaC.HasCDK {
		b.WriteString("AWS CDK detected\n")
	}
	if !f.IaC.HasBicep && !f.IaC.HasTerraform && !f.IaC.HasARM && !f.IaC.HasCloudFormation && !f.IaC.HasCDK {
		b.WriteString("No IaC detected.\n")
	}
	b.WriteString("\n")

	// Dependencies
	b.WriteString("## Dependencies\n\n")
	if len(f.Deps) == 0 {
		b.WriteString("No dependency files detected.\n")
	} else {
		for _, d := range f.Deps {
			b.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", d.File, d.Type, d.Details))
		}
	}
	b.WriteString("\n")

	// AWS Hints
	b.WriteString("## AWS Service Usage\n\n")
	if len(f.AWSHints) == 0 {
		b.WriteString("No AWS service usage detected.\n")
	} else {
		b.WriteString("| Service | File | Line | Evidence |\n")
		b.WriteString("|---|---|---|---|\n")
		for _, h := range f.AWSHints {
			b.WriteString(fmt.Sprintf("| %s | %s | %d | %s |\n", h.Service, h.File, h.Line, h.Evidence))
		}
	}
	b.WriteString("\n")

	// Secrets
	b.WriteString("## Secret / Credential Findings\n\n")
	if len(f.Secrets) == 0 {
		b.WriteString("No secrets or credentials detected.\n")
	} else {
		b.WriteString(fmt.Sprintf("**%d finding(s)**\n\n", len(f.Secrets)))
		b.WriteString("| # | File | Line | Type | Severity | Evidence | Remediation |\n")
		b.WriteString("|---|---|---|---|---|---|---|\n")
		for i, s := range f.Secrets {
			b.WriteString(fmt.Sprintf("| %d | %s | %d | %s | %s | %s | %s |\n",
				i+1, s.File, s.Line, s.Type, s.Severity, s.Evidence, s.Remediation))
		}
	}
	b.WriteString("\n")

	return b.String()
}
