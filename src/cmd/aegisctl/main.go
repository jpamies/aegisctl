// aegisctl — Azure architecture advisor (WAF-aligned).
//
// Commands (Terraform-style workflow):
//
//	init  [repoPath]                       Scan repo → enrich with Copilot → save state.
//	plan  [repoPath]                       Generate multi-option recommendations → save plan.
//	apply [repoPath] [--output <dir>]      Write IaC, pipelines, and docs from the plan.
//
// Default behaviour is generate-only. Nothing is deployed unless explicitly enabled.
// All paths default to "." (current directory). Output defaults to "out/".
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aegis/aegisctl/internal/analyzer"
	"github.com/aegis/aegisctl/internal/copilot"
	"github.com/aegis/aegisctl/internal/generator"
	"github.com/aegis/aegisctl/internal/output"
	"github.com/aegis/aegisctl/internal/recommend"
	"github.com/aegis/aegisctl/internal/state"
)

const version = "0.3.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "init":
		runInit(os.Args[2:])
	case "plan":
		runPlan(os.Args[2:])
	case "apply":
		runApply(os.Args[2:])
	case "analyze":
		runAnalyze(os.Args[2:])
	case "--version", "version":
		fmt.Printf("aegisctl %s\n", version)
	case "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "aegisctl: unknown command %q\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `aegisctl — Azure architecture advisor (WAF-aligned)

Usage:
  aegisctl <command> [flags]

Commands:
  init    [repoPath]                          Scan repository and save analysis state.
  plan    [repoPath]                          Generate architecture recommendations.
  apply   [repoPath] [--output <dir>] [flags] Write IaC, pipelines, and docs from the plan.

  analyze [repoPath]                          Quick analysis (no state, printed to stdout).

All commands default to the current directory (".") if repoPath is omitted.

Apply flags:
  --output <dir>              Output directory (default: out/)
  --deploy off|manual|auto    Deploy mode (default: off)
  --option <number>           Select architecture option (1-based, overrides plan)

Global flags:
  --help       Show this help message
  --version    Print version

Environment:
  AEGIS_GITHUB_TOKEN  Fine-grained PAT for GitHub Models API (preferred).
  GITHUB_TOKEN        Fallback if AEGIS_GITHUB_TOKEN is not set.
                      Copilot features require one of the above; falls back
                      to heuristic mode if neither is set.

Workflow:
  1. aegisctl init              → scans repo, saves .aegis/state.json
  2. aegisctl plan              → generates options, saves .aegis/plan.json
  3. aegisctl apply             → writes IaC + pipelines + docs to out/
  4. aegisctl apply -o custom/  → writes to a custom directory

Default: generate-only. No deployment unless explicitly enabled with approval gates.
Idempotent: all commands are safe to run multiple times.`)
}

// parseFlags parses simple key-value flags from args. Supports --key value, -k value, and --flag (bool).
func parseFlags(args []string) (positional []string, flags map[string]string) {
	flags = make(map[string]string)
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") || (strings.HasPrefix(args[i], "-") && len(args[i]) == 2) {
			key := strings.TrimLeft(args[i], "-")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags[key] = args[i+1]
				i++
			} else {
				flags[key] = "true"
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return
}

// initCopilotClient creates a Copilot client if GITHUB_TOKEN is set.
// Returns nil (not an error) if the token is unavailable.
func initCopilotClient() *copilot.Client {
	if !copilot.IsAvailable() {
		return nil
	}
	client, err := copilot.NewClient()
	if err != nil {
		return nil
	}
	return client
}

// --- init command ---

func runInit(args []string) {
	pos, flags := parseFlags(args)
	repoPath := "."
	if len(pos) > 0 {
		repoPath = pos[0]
	}
	_ = flags

	fmt.Println("aegisctl init — scanning repository...")
	fmt.Println()

	// Idempotency: warn if state already exists, then overwrite
	if state.StateExists(repoPath) {
		fmt.Println("  ℹ Existing state found — will be refreshed.")
	}

	// 1. Heuristic scan
	fmt.Println("  [1/3] Running heuristic analysis...")
	findings, err := analyzer.Analyze(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl init: analysis failed: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("  ✓ Scanned %d files\n", findings.FileCount)

	// 2. Copilot enrichment
	client := initCopilotClient()
	mode := "heuristic"
	if client != nil {
		mode = "copilot"
		fmt.Println("  [2/3] Enriching with GitHub Copilot...")
	} else {
		fmt.Println("  [2/3] Copilot unavailable (no GITHUB_TOKEN), using heuristic enrichment...")
	}

	enriched, err := recommend.EnrichAnalysis(client, findings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Copilot enrichment failed, falling back to heuristic: %v\n", err)
		enriched, _ = recommend.EnrichAnalysis(nil, findings)
		mode = "heuristic"
	}
	fmt.Printf("  ✓ App type: %s (%s)\n", enriched.AppType, enriched.Complexity)

	// 3. Save state
	fmt.Println("  [3/3] Saving state...")
	s := &state.AnalysisState{
		Version:   version,
		CreatedAt: time.Now(),
		RepoPath:  repoPath,
		Heuristic: toHeuristicFindings(findings),
		Enriched:  enriched,
		Mode:      mode,
	}

	if err := state.SaveState(repoPath, s); err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl init: saving state: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("  ✓ State saved to %s/state.json\n", state.DirName)
	fmt.Println()
	printInitSummary(s)
	fmt.Println()
	fmt.Println("Next step: aegisctl plan")
}

func printInitSummary(s *state.AnalysisState) {
	fmt.Println("  Summary:")
	fmt.Printf("    Mode:       %s\n", s.Mode)
	fmt.Printf("    Files:      %d\n", s.Heuristic.FileCount)
	fmt.Printf("    Languages:  ")
	if len(s.Heuristic.Languages) > 0 {
		names := make([]string, len(s.Heuristic.Languages))
		for i, l := range s.Heuristic.Languages {
			names[i] = l.Language
		}
		fmt.Println(strings.Join(names, ", "))
	} else {
		fmt.Println("(none detected)")
	}
	fmt.Printf("    Docker:     %v\n", s.Heuristic.HasDocker)
	fmt.Printf("    CI/CD:      %v\n", s.Heuristic.CI.HasGitHubActions || s.Heuristic.CI.HasOtherCI)
	fmt.Printf("    IaC:        %v\n", s.Heuristic.IaC.HasBicep || s.Heuristic.IaC.HasTerraform || s.Heuristic.IaC.HasARM)
	fmt.Printf("    AWS hints:  %d\n", len(s.Heuristic.AWSHints))
	fmt.Printf("    Secrets:    %d finding(s)\n", len(s.Heuristic.Secrets))

	if s.Enriched != nil {
		fmt.Printf("    App type:   %s\n", s.Enriched.AppType)
		fmt.Printf("    Complexity: %s\n", s.Enriched.Complexity)
	}
}

// --- plan command ---

func runPlan(args []string) {
	pos, flags := parseFlags(args)
	repoPath := "."
	if len(pos) > 0 {
		repoPath = pos[0]
	}
	_ = flags

	fmt.Println("aegisctl plan — generating architecture recommendations...")
	fmt.Println()

	// 1. Load state
	if !state.StateExists(repoPath) {
		fmt.Fprintln(os.Stderr, "aegisctl plan: no state found. Run 'aegisctl init' first.")
		os.Exit(1)
	}

	// Idempotency: warn if plan already exists, then overwrite
	if state.PlanExists(repoPath) {
		fmt.Println("  ℹ Existing plan found — will be replaced.")
	}

	s, err := state.LoadState(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl plan: %v\n", err)
		os.Exit(2)
	}

	// 2. Generate recommendations
	client := initCopilotClient()
	if client != nil {
		fmt.Println("  Generating recommendations with GitHub Copilot...")
	} else {
		fmt.Println("  Generating heuristic recommendations...")
	}

	plan, err := recommend.GeneratePlan(client, s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Copilot recommendation failed, falling back to heuristic: %v\n", err)
		plan, _ = recommend.GeneratePlan(nil, s)
	}

	plan.Version = version
	plan.CreatedAt = time.Now()

	// 3. Display options
	fmt.Println()
	fmt.Printf("  Generated %d architecture options (mode: %s):\n\n", len(plan.Options), plan.Mode)

	for i, opt := range plan.Options {
		marker := "  "
		if opt.Recommended {
			marker = "→ "
		}
		fmt.Printf("  %s[%d] %s\n", marker, i+1, opt.Name)
		fmt.Printf("       %s\n", opt.Description)
		fmt.Printf("       Compute: %s (%s / %s)\n", opt.Compute.Service, opt.Compute.SKU, opt.Compute.SKUProd)
		fmt.Printf("       Cost: dev %s | prod %s\n", opt.EstimatedCostDev, opt.EstimatedCostProd)
		fmt.Printf("       WAF: R=%d S=%d C=%d O=%d P=%d\n",
			opt.WAFScores.Reliability, opt.WAFScores.Security,
			opt.WAFScores.CostOptimization, opt.WAFScores.OperationalExcellence,
			opt.WAFScores.PerformanceEfficiency)
		fmt.Println()
	}

	if len(plan.SecretsRemediation) > 0 {
		fmt.Printf("  Secrets remediation: %d finding(s)\n", len(plan.SecretsRemediation))
	}

	// 4. Interactive selection (if terminal)
	selected := plan.SelectedOption
	if isInteractive() {
		fmt.Printf("  Select option [1-%d] (default: %d): ", len(plan.Options), selected+1)
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			n, err := strconv.Atoi(input)
			if err != nil || n < 1 || n > len(plan.Options) {
				fmt.Fprintf(os.Stderr, "aegisctl plan: invalid option %q\n", input)
				os.Exit(1)
			}
			selected = n - 1
		}
	}
	plan.SelectedOption = selected

	// 5. Save plan
	if err := state.SavePlan(repoPath, plan); err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl plan: saving plan: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("  ✓ Plan saved to %s/plan.json (option %d: %s)\n", state.DirName, selected+1, plan.Options[selected].Name)
	fmt.Println()
	fmt.Println("  Files that will be generated:")
	for _, f := range plan.Options[selected].FilesToGenerate {
		fmt.Printf("    + %s\n", f)
	}
	fmt.Println()
	fmt.Println("Next step: aegisctl apply")
}

// --- apply command ---

func runApply(args []string) {
	pos, flags := parseFlags(args)
	repoPath := "."
	if len(pos) > 0 {
		repoPath = pos[0]
	}
	outputDir := flags["output"]
	if outputDir == "" {
		outputDir = flags["o"]
	}
	if outputDir == "" {
		outputDir = "out"
	}
	deployMode := flags["deploy"]
	if deployMode == "" {
		deployMode = "off"
	}
	if deployMode != "off" && deployMode != "manual" && deployMode != "auto" {
		fmt.Fprintf(os.Stderr, "aegisctl apply: invalid --deploy mode %q (use off|manual|auto)\n", deployMode)
		os.Exit(1)
	}

	fmt.Println("aegisctl apply — generating artefacts...")
	fmt.Printf("  Output directory: %s\n", outputDir)
	fmt.Println()

	// 1. Load state + plan
	if !state.StateExists(repoPath) {
		fmt.Fprintln(os.Stderr, "aegisctl apply: no state found. Run 'aegisctl init' first.")
		os.Exit(1)
	}
	if !state.PlanExists(repoPath) {
		fmt.Fprintln(os.Stderr, "aegisctl apply: no plan found. Run 'aegisctl plan' first.")
		os.Exit(1)
	}

	s, err := state.LoadState(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl apply: %v\n", err)
		os.Exit(2)
	}

	p, err := state.LoadPlan(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl apply: %v\n", err)
		os.Exit(2)
	}

	// Allow --option to override plan selection
	if optStr, ok := flags["option"]; ok {
		n, err := strconv.Atoi(optStr)
		if err != nil || n < 1 || n > len(p.Options) {
			fmt.Fprintf(os.Stderr, "aegisctl apply: invalid --option %q\n", optStr)
			os.Exit(1)
		}
		p.SelectedOption = n - 1
	}

	// 2. Generate
	client := initCopilotClient()

	cfg := generator.Config{
		OutputDir:  outputDir,
		DeployMode: deployMode,
	}

	if err := generator.Generate(client, s, p, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl apply: %v\n", err)
		os.Exit(5)
	}

	fmt.Println()
	fmt.Printf("Artefacts generated at: %s\n", outputDir)
	fmt.Printf("Architecture: %s\n", p.Options[p.SelectedOption].Name)
	fmt.Printf("Deploy mode: %s\n", deployMode)
	if deployMode == "off" {
		fmt.Println("No deployment activated. Use --deploy manual|auto to enable.")
	}
}

// --- analyze command (quick, no state) ---

func runAnalyze(args []string) {
	pos, flags := parseFlags(args)
	repoPath := "."
	if len(pos) > 0 {
		repoPath = pos[0]
	}
	_ = flags

	findings, err := analyzer.Analyze(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl analyze: %v\n", err)
		os.Exit(2)
	}

	out := output.FormatAnalysis(findings)
	fmt.Print(out)
}

// --- Helpers ---

// toHeuristicFindings converts analyzer.Findings to state.HeuristicFindings.
func toHeuristicFindings(f *analyzer.Findings) state.HeuristicFindings {
	h := state.HeuristicFindings{
		HasDocker:   f.HasDocker,
		Dockerfiles: f.Dockerfiles,
		FileCount:   f.FileCount,
		CI: state.CIInfo{
			HasGitHubActions: f.CI.HasGitHubActions,
			Workflows:        f.CI.Workflows,
			HasOtherCI:       f.CI.HasOtherCI,
			OtherCI:          f.CI.OtherCI,
		},
		IaC: state.IaCInfo{
			HasBicep:            f.IaC.HasBicep,
			BicepFiles:          f.IaC.BicepFiles,
			HasTerraform:        f.IaC.HasTerraform,
			TerraformFiles:      f.IaC.TerraformFiles,
			HasARM:              f.IaC.HasARM,
			ARMFiles:            f.IaC.ARMFiles,
			HasCDK:              f.IaC.HasCDK,
			HasCloudFormation:   f.IaC.HasCloudFormation,
			CloudFormationFiles: f.IaC.CloudFormationFiles,
		},
	}

	for _, l := range f.Languages {
		h.Languages = append(h.Languages, state.Language{
			Language: l.Language,
			Evidence: l.Evidence,
		})
	}

	for _, d := range f.Deps {
		h.Deps = append(h.Deps, state.Dependency{
			File:    d.File,
			Type:    d.Type,
			Details: d.Details,
		})
	}

	for _, a := range f.AWSHints {
		h.AWSHints = append(h.AWSHints, state.AWSHint{
			Service:  a.Service,
			File:     a.File,
			Line:     a.Line,
			Evidence: a.Evidence,
		})
	}

	for _, s := range f.Secrets {
		h.Secrets = append(h.Secrets, state.SecretFinding{
			File:        s.File,
			Line:        s.Line,
			Type:        s.Type,
			Severity:    s.Severity,
			Evidence:    s.Evidence,
			Remediation: s.Remediation,
		})
	}

	return h
}

// isInteractive returns true if stdin is a terminal.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
