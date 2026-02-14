// Package scorer generates WAF heuristic scorecards based on repository analysis.
package scorer

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/aegis/aegisctl/internal/analyzer"
	"github.com/aegis/aegisctl/internal/output"
)

// Scores holds the 0–5 score for each WAF pillar.
type Scores struct {
	Reliability int
	Security    int
	CostOpt     int
	OpsExcel    int
	PerfEff     int
	Overall     int
}

// Rationale holds the one-line rationale per pillar.
type Rationale struct {
	Reliability string
	Security    string
	CostOpt     string
	OpsExcel    string
	PerfEff     string
}

// Details holds strengths and gaps per pillar.
type Details struct {
	ReliabilityStrengths string
	ReliabilityGaps      string
	SecurityStrengths    string
	SecurityGaps         string
	CostOptStrengths     string
	CostOptGaps          string
	OpsExcelStrengths    string
	OpsExcelGaps         string
	PerfEffStrengths     string
	PerfEffGaps          string
}

// Improvement is a prioritized recommendation.
type Improvement struct {
	Title       string
	Pillar      string
	Impact      string
	Description string
}

// ScorecardData is the full data passed to the scorecard template.
type ScorecardData struct {
	RepoPath     string
	Scores       Scores
	Rationale    Rationale
	Details      Details
	Improvements []Improvement
}

// Score analyzes a repo and writes the scorecard + checklist to outputDir.
func Score(repoPath, outputDir string) error {
	findings, err := analyzer.Analyze(repoPath)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	data := computeScorecard(findings)

	funcMap := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}

	// Render scorecard
	t, err := template.New("scorecard").Funcs(funcMap).Parse(output.WAFScorecardTmpl)
	if err != nil {
		return fmt.Errorf("parsing scorecard template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing scorecard template: %w", err)
	}
	if err := output.WriteFile(filepath.Join(outputDir, "docs", "WAF_SCORECARD.md"), buf.String()); err != nil {
		return err
	}

	return nil
}

// computeScorecard generates heuristic scores from findings.
func computeScorecard(f *analyzer.Findings) ScorecardData {
	s := Scores{}
	r := Rationale{}
	d := Details{}

	// --- Reliability ---
	s.Reliability = 2 // baseline
	r.Reliability = "Baseline score."
	d.ReliabilityStrengths = "Repository exists with structured code."
	d.ReliabilityGaps = "No health endpoints, retry policies, or DR plan detected."
	if f.HasDocker {
		s.Reliability++
		r.Reliability = "Container support detected (enables health probes)."
		d.ReliabilityStrengths = "Dockerfile present — container orchestration possible."
	}
	if f.CI.HasGitHubActions {
		s.Reliability++
		d.ReliabilityStrengths += " CI pipeline provides build validation."
	}

	// --- Security ---
	s.Security = 3 // baseline
	r.Security = "Baseline score."
	d.SecurityStrengths = "No secrets detected."
	d.SecurityGaps = "Review recommended. Managed Identity not yet confirmed."
	if len(f.Secrets) > 0 {
		s.Security = max(1, s.Security-len(f.Secrets))
		r.Security = fmt.Sprintf("%d secret(s) detected — remediation required.", len(f.Secrets))
		d.SecurityStrengths = "Secrets were identified (first step to remediation)."
		d.SecurityGaps = fmt.Sprintf("%d hardcoded secret(s) need migration to Key Vault.", len(f.Secrets))
	}
	if f.IaC.HasBicep {
		s.Security = min(5, s.Security+1)
		d.SecurityStrengths += " Bicep IaC enables consistent security configuration."
	}

	// --- Cost Optimization ---
	s.CostOpt = 2
	r.CostOpt = "Baseline — no cost signals analyzed."
	d.CostOptStrengths = "Using open-source tooling (Go, GitHub Actions)."
	d.CostOptGaps = "No SKU selection, autoscale, or cost alerting detected."
	if f.IaC.HasBicep {
		s.CostOpt++
		r.CostOpt = "Bicep IaC enables parameterized SKU selection."
		d.CostOptStrengths += " IaC supports environment-based SKU tiers."
	}

	// --- Operational Excellence ---
	s.OpsExcel = 1
	r.OpsExcel = "No CI/CD or IaC detected."
	d.OpsExcelStrengths = "Repository is version-controlled."
	d.OpsExcelGaps = "No CI, no IaC, no deployment pipeline detected."
	if f.CI.HasGitHubActions {
		s.OpsExcel += 2
		r.OpsExcel = "GitHub Actions CI detected."
		d.OpsExcelStrengths = "GitHub Actions workflows present."
		d.OpsExcelGaps = "Review test coverage and deployment gates."
	}
	if f.IaC.HasBicep {
		s.OpsExcel++
		r.OpsExcel += " Bicep IaC present."
		d.OpsExcelStrengths += " Bicep IaC for infrastructure management."
	}
	if len(f.CI.Workflows) >= 3 {
		s.OpsExcel = min(5, s.OpsExcel+1)
		d.OpsExcelStrengths += " Multiple workflow files suggest mature pipeline."
	}

	// --- Performance Efficiency ---
	s.PerfEff = 2
	r.PerfEff = "Baseline — no performance signals analyzed."
	d.PerfEffStrengths = "Application code exists for optimization."
	d.PerfEffGaps = "No autoscale, caching, CDN, or load testing detected."
	if f.HasDocker {
		s.PerfEff++
		r.PerfEff = "Container support enables scaling strategies."
		d.PerfEffStrengths += " Container-based deployment supports horizontal scaling."
	}

	// Clamp all scores
	s.Reliability = clamp(s.Reliability, 0, 5)
	s.Security = clamp(s.Security, 0, 5)
	s.CostOpt = clamp(s.CostOpt, 0, 5)
	s.OpsExcel = clamp(s.OpsExcel, 0, 5)
	s.PerfEff = clamp(s.PerfEff, 0, 5)

	// Overall = average
	s.Overall = (s.Reliability + s.Security + s.CostOpt + s.OpsExcel + s.PerfEff) / 5

	// Generate improvements
	improvements := generateImprovements(f, s)

	return ScorecardData{
		RepoPath:     f.RepoPath,
		Scores:       s,
		Rationale:    r,
		Details:      d,
		Improvements: improvements,
	}
}

func generateImprovements(f *analyzer.Findings, s Scores) []Improvement {
	var imps []Improvement

	if len(f.Secrets) > 0 {
		imps = append(imps, Improvement{
			Title:       "Remediate hardcoded secrets",
			Pillar:      "Security",
			Impact:      "High",
			Description: "Migrate detected secrets to Azure Key Vault. Use Managed Identity where possible.",
		})
	}

	if !f.CI.HasGitHubActions {
		imps = append(imps, Improvement{
			Title:       "Add CI/CD pipeline",
			Pillar:      "Operational Excellence",
			Impact:      "High",
			Description: "Implement GitHub Actions with build, test, and gated deploy workflows.",
		})
	}

	if !f.IaC.HasBicep {
		imps = append(imps, Improvement{
			Title:       "Adopt Infrastructure as Code (Bicep)",
			Pillar:      "Operational Excellence",
			Impact:      "High",
			Description: "Define all Azure resources in Bicep for repeatability and drift prevention.",
		})
	}

	if !f.HasDocker {
		imps = append(imps, Improvement{
			Title:       "Containerize the application",
			Pillar:      "Reliability",
			Impact:      "Medium",
			Description: "Add a Dockerfile to enable consistent deployments and container orchestration.",
		})
	}

	imps = append(imps, Improvement{
		Title:       "Implement health probes and monitoring",
		Pillar:      "Reliability",
		Impact:      "Medium",
		Description: "Add health check endpoints and configure Application Insights for observability.",
	})

	// Cap at 5
	if len(imps) > 5 {
		imps = imps[:5]
	}

	return imps
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
