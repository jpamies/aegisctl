// aegisctl — SRE/DevOps architecture pack generator for Azure.
//
// Commands:
//
//	analyze <repoPath>                    Detect app profile, dependencies, AWS hints, secrets.
//	pack    <repoPath> --output <dir>     Generate architecture pack (docs, IaC, workflows, patches).
//	score   <repoPath> --output <dir>     Emit WAF checklist + heuristic scorecard.
//	migrate-aws <repoPath> --output <dir> Emit AWS→Azure mapping report.
//
// Default behaviour is generate-only. Nothing is deployed unless explicitly enabled.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aegis/aegisctl/internal/analyzer"
	"github.com/aegis/aegisctl/internal/migrator"
	"github.com/aegis/aegisctl/internal/output"
	"github.com/aegis/aegisctl/internal/packer"
	"github.com/aegis/aegisctl/internal/scorer"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "analyze":
		runAnalyze(os.Args[2:])
	case "pack":
		runPack(os.Args[2:])
	case "score":
		runScore(os.Args[2:])
	case "migrate-aws":
		runMigrateAWS(os.Args[2:])
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
	fmt.Fprintln(os.Stderr, `aegisctl — SRE/DevOps architecture pack generator for Azure (WAF-aligned)

Usage:
  aegisctl <command> [flags]

Commands:
  analyze      <repoPath>                          Detect app profile, deps, AWS hints, secrets.
  pack         <repoPath> --output <dir> [flags]    Generate architecture pack.
  score        <repoPath> --output <dir>            Emit WAF checklist + heuristic scorecard.
  migrate-aws  <repoPath> --output <dir>            Emit AWS→Azure mapping report.

Pack flags:
  --output <dir>        Output directory (required)
  --deploy off|manual|auto   Deploy mode (default: off)
  --iaca                Include IaC artefacts
  --pipeline            Include pipeline artefacts
  --aws                 Include AWS migration assessment

Global flags:
  --help       Show this help message
  --version    Print version

Default: generate-only. No deployment unless explicitly enabled with approval gates.`)
}

// parseFlags parses simple key-value flags from args. Supports --key value and --flag (bool).
func parseFlags(args []string) (positional []string, flags map[string]string) {
	flags = make(map[string]string)
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			key := strings.TrimPrefix(args[i], "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
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

func runAnalyze(args []string) {
	pos, flags := parseFlags(args)
	if len(pos) == 0 {
		fmt.Fprintln(os.Stderr, "aegisctl analyze: missing <repoPath>")
		os.Exit(1)
	}
	repoPath := pos[0]
	_ = flags

	findings, err := analyzer.Analyze(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl analyze: %v\n", err)
		os.Exit(2)
	}

	out := output.FormatAnalysis(findings)
	fmt.Print(out)
}

func runPack(args []string) {
	pos, flags := parseFlags(args)
	if len(pos) == 0 {
		fmt.Fprintln(os.Stderr, "aegisctl pack: missing <repoPath>")
		os.Exit(1)
	}
	repoPath := pos[0]
	outputDir := flags["output"]
	if outputDir == "" {
		fmt.Fprintln(os.Stderr, "aegisctl pack: --output <dir> is required")
		os.Exit(1)
	}
	deployMode := flags["deploy"]
	if deployMode == "" {
		deployMode = "off"
	}
	if deployMode != "off" && deployMode != "manual" && deployMode != "auto" {
		fmt.Fprintf(os.Stderr, "aegisctl pack: invalid --deploy mode %q (use off|manual|auto)\n", deployMode)
		os.Exit(1)
	}

	includeIaC := flags["iaca"] == "true"
	includePipeline := flags["pipeline"] == "true"
	includeAWS := flags["aws"] == "true"

	// Default: include everything
	if !includeIaC && !includePipeline && !includeAWS {
		includeIaC = true
		includePipeline = true
		includeAWS = true
	}

	cfg := packer.Config{
		RepoPath:        repoPath,
		OutputDir:       outputDir,
		DeployMode:      deployMode,
		IncludeIaC:      includeIaC,
		IncludePipeline: includePipeline,
		IncludeAWS:      includeAWS,
	}

	err := packer.Pack(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl pack: %v\n", err)
		os.Exit(5)
	}

	fmt.Printf("Pack generated at: %s\n", outputDir)
	fmt.Printf("Deploy mode: %s\n", deployMode)
	if deployMode == "off" {
		fmt.Println("No deployment artefacts activated. Use --deploy manual|auto to enable.")
	}
}

func runScore(args []string) {
	pos, flags := parseFlags(args)
	if len(pos) == 0 {
		fmt.Fprintln(os.Stderr, "aegisctl score: missing <repoPath>")
		os.Exit(1)
	}
	repoPath := pos[0]
	outputDir := flags["output"]
	if outputDir == "" {
		fmt.Fprintln(os.Stderr, "aegisctl score: --output <dir> is required")
		os.Exit(1)
	}

	err := scorer.Score(repoPath, outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl score: %v\n", err)
		os.Exit(3)
	}

	fmt.Printf("WAF scorecard generated at: %s\n", outputDir)
}

func runMigrateAWS(args []string) {
	pos, flags := parseFlags(args)
	if len(pos) == 0 {
		fmt.Fprintln(os.Stderr, "aegisctl migrate-aws: missing <repoPath>")
		os.Exit(1)
	}
	repoPath := pos[0]
	outputDir := flags["output"]
	if outputDir == "" {
		fmt.Fprintln(os.Stderr, "aegisctl migrate-aws: --output <dir> is required")
		os.Exit(1)
	}

	err := migrator.MigrateAWS(repoPath, outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aegisctl migrate-aws: %v\n", err)
		os.Exit(4)
	}

	fmt.Printf("AWS migration report generated at: %s\n", outputDir)
}
