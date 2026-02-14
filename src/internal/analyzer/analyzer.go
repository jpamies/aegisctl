// Package analyzer scans a repository to detect its application profile:
// language/runtime, Dockerfile presence, CI configuration, IaC files,
// dependency hints, AWS service usage, and secret patterns.
package analyzer

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Findings holds the complete analysis results for a repository.
type Findings struct {
	RepoPath    string
	Languages   []LanguageHint
	HasDocker   bool
	Dockerfiles []string
	CI          CIInfo
	IaC         IaCInfo
	Deps        []DependencyHint
	AWSHints    []AWSHint
	Secrets     []SecretFinding
	FileCount   int
	Files       []string // sorted, relative paths
}

// LanguageHint records a detected language/runtime and evidence.
type LanguageHint struct {
	Language string
	Evidence string // e.g. "go.mod found", "package.json found"
}

// CIInfo describes detected CI/CD configuration.
type CIInfo struct {
	HasGitHubActions bool
	Workflows        []string
	HasOtherCI       bool
	OtherCI          string // e.g. "Jenkinsfile", ".gitlab-ci.yml"
}

// IaCInfo describes detected Infrastructure-as-Code files.
type IaCInfo struct {
	HasBicep            bool
	BicepFiles          []string
	HasTerraform        bool
	TerraformFiles      []string
	HasARM              bool
	ARMFiles            []string
	HasCDK              bool
	HasCloudFormation   bool
	CloudFormationFiles []string
}

// DependencyHint records a dependency signal.
type DependencyHint struct {
	File    string
	Type    string // e.g. "go.mod", "package.json", "requirements.txt"
	Details string
}

// AWSHint records an AWS service usage signal.
type AWSHint struct {
	Service  string // e.g. "S3", "Lambda", "DynamoDB"
	File     string
	Line     int
	Evidence string
}

// Analyze performs a full repository scan and returns structured findings.
func Analyze(repoPath string) (*Findings, error) {
	info, err := os.Stat(repoPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access repo path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("repo path is not a directory: %s", repoPath)
	}

	f := &Findings{RepoPath: repoPath}

	// Walk files deterministically (sorted)
	err = filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		// Skip hidden dirs (except .github), vendor, node_modules, .git
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "vendor" || name == "node_modules" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(repoPath, path)
		rel = filepath.ToSlash(rel)
		f.Files = append(f.Files, rel)
		f.FileCount++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking repo: %w", err)
	}
	sort.Strings(f.Files)

	// Detect languages
	f.Languages = detectLanguages(f.Files)

	// Detect Dockerfiles
	f.Dockerfiles, f.HasDocker = detectDocker(f.Files)

	// Detect CI
	f.CI = detectCI(f.Files)

	// Detect IaC
	f.IaC = detectIaC(f.Files, repoPath)

	// Detect dependencies
	f.Deps = detectDeps(f.Files, repoPath)

	// Detect AWS hints
	f.AWSHints = detectAWSHints(f.Files, repoPath)

	// Detect secrets
	f.Secrets = DetectSecrets(f.Files, repoPath)

	return f, nil
}

// detectLanguages infers languages from file extensions and marker files.
func detectLanguages(files []string) []LanguageHint {
	seen := map[string]string{}
	for _, f := range files {
		base := filepath.Base(f)
		ext := strings.ToLower(filepath.Ext(f))
		switch {
		case base == "go.mod" || ext == ".go":
			seen["Go"] = base + " found"
		case base == "package.json" || ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx":
			seen["JavaScript/TypeScript"] = base + " found"
		case base == "requirements.txt" || base == "Pipfile" || base == "pyproject.toml" || ext == ".py":
			seen["Python"] = base + " found"
		case base == "Cargo.toml" || ext == ".rs":
			seen["Rust"] = base + " found"
		case ext == ".java" || base == "pom.xml" || base == "build.gradle":
			seen["Java"] = base + " found"
		case ext == ".cs" || ext == ".csproj" || ext == ".sln":
			seen["C#/.NET"] = base + " found"
		case ext == ".rb" || base == "Gemfile":
			seen["Ruby"] = base + " found"
		case ext == ".php" || base == "composer.json":
			seen["PHP"] = base + " found"
		}
	}
	var hints []LanguageHint
	// Sorted output for determinism
	var langs []string
	for l := range seen {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	for _, l := range langs {
		hints = append(hints, LanguageHint{Language: l, Evidence: seen[l]})
	}
	return hints
}

func detectDocker(files []string) ([]string, bool) {
	var dockerfiles []string
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f))
		if base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") || base == "docker-compose.yml" || base == "docker-compose.yaml" || base == "compose.yml" || base == "compose.yaml" {
			dockerfiles = append(dockerfiles, f)
		}
	}
	sort.Strings(dockerfiles)
	return dockerfiles, len(dockerfiles) > 0
}

func detectCI(files []string) CIInfo {
	ci := CIInfo{}
	for _, f := range files {
		if strings.HasPrefix(f, ".github/workflows/") && (strings.HasSuffix(f, ".yml") || strings.HasSuffix(f, ".yaml")) {
			ci.HasGitHubActions = true
			ci.Workflows = append(ci.Workflows, f)
		}
		base := filepath.Base(f)
		switch base {
		case "Jenkinsfile":
			ci.HasOtherCI = true
			ci.OtherCI = "Jenkins"
		case ".gitlab-ci.yml":
			ci.HasOtherCI = true
			ci.OtherCI = "GitLab CI"
		case "azure-pipelines.yml":
			ci.HasOtherCI = true
			ci.OtherCI = "Azure DevOps"
		case ".circleci":
			ci.HasOtherCI = true
			ci.OtherCI = "CircleCI"
		case ".travis.yml":
			ci.HasOtherCI = true
			ci.OtherCI = "Travis CI"
		}
	}
	sort.Strings(ci.Workflows)
	return ci
}

func detectIaC(files []string, repoPath string) IaCInfo {
	iac := IaCInfo{}
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		base := strings.ToLower(filepath.Base(f))
		switch {
		case ext == ".bicep":
			iac.HasBicep = true
			iac.BicepFiles = append(iac.BicepFiles, f)
		case ext == ".tf":
			iac.HasTerraform = true
			iac.TerraformFiles = append(iac.TerraformFiles, f)
		case base == "template.json" || base == "azuredeploy.json":
			iac.HasARM = true
			iac.ARMFiles = append(iac.ARMFiles, f)
		case base == "cdk.json":
			iac.HasCDK = true
		case strings.Contains(f, "cloudformation") && (ext == ".yml" || ext == ".yaml" || ext == ".json"):
			iac.HasCloudFormation = true
			iac.CloudFormationFiles = append(iac.CloudFormationFiles, f)
		}
	}
	// Also check for CloudFormation templates by content (look for AWSTemplateFormatVersion)
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == ".yml" || ext == ".yaml" || ext == ".json" {
			full := filepath.Join(repoPath, filepath.FromSlash(f))
			if containsString(full, "AWSTemplateFormatVersion") {
				if !iac.HasCloudFormation {
					iac.HasCloudFormation = true
				}
				found := false
				for _, cf := range iac.CloudFormationFiles {
					if cf == f {
						found = true
						break
					}
				}
				if !found {
					iac.CloudFormationFiles = append(iac.CloudFormationFiles, f)
				}
			}
		}
	}
	sort.Strings(iac.BicepFiles)
	sort.Strings(iac.TerraformFiles)
	sort.Strings(iac.ARMFiles)
	sort.Strings(iac.CloudFormationFiles)
	return iac
}

func detectDeps(files []string, repoPath string) []DependencyHint {
	var deps []DependencyHint
	depFiles := map[string]string{
		"go.mod":            "Go modules",
		"go.sum":            "Go modules (lock)",
		"package.json":      "Node.js/npm",
		"package-lock.json": "Node.js/npm (lock)",
		"yarn.lock":         "Yarn",
		"requirements.txt":  "Python pip",
		"Pipfile":           "Python Pipenv",
		"pyproject.toml":    "Python (PEP 517)",
		"Cargo.toml":        "Rust Cargo",
		"pom.xml":           "Java Maven",
		"build.gradle":      "Java Gradle",
		"Gemfile":           "Ruby Bundler",
		"composer.json":     "PHP Composer",
	}
	for _, f := range files {
		base := filepath.Base(f)
		if typ, ok := depFiles[base]; ok {
			details := ""
			// Try to read first few lines for context
			full := filepath.Join(repoPath, filepath.FromSlash(f))
			lines := readFirstLines(full, 5)
			if len(lines) > 0 {
				details = strings.Join(lines, " | ")
			}
			deps = append(deps, DependencyHint{File: f, Type: typ, Details: details})
		}
	}
	return deps
}

// AWS service patterns to search for in source files.
var awsPatterns = []struct {
	Pattern string
	Service string
}{
	{"aws-sdk", "AWS SDK"},
	{"boto3", "AWS SDK (Python/boto3)"},
	{"@aws-sdk", "AWS SDK v3 (JS)"},
	{"awscli", "AWS CLI"},
	{"amazonaws.com", "AWS API endpoint"},
	{"s3.amazonaws", "S3"},
	{"s3:", "S3"},
	{"s3_bucket", "S3"},
	{"s3.Bucket", "S3"},
	{"dynamodb", "DynamoDB"},
	{"DynamoDB", "DynamoDB"},
	{"sns:", "SNS"},
	{"sns.Topic", "SNS"},
	{"sqs:", "SQS"},
	{"sqs.Queue", "SQS"},
	{"lambda", "Lambda"},
	{"Lambda", "Lambda"},
	{"ecs:", "ECS"},
	{"ECS", "ECS"},
	{"eks:", "EKS"},
	{"EKS", "EKS"},
	{"cloudwatch", "CloudWatch"},
	{"CloudWatch", "CloudWatch"},
	{"cloudformation", "CloudFormation"},
	{"CloudFormation", "CloudFormation"},
	{"aws_", "AWS (Terraform provider)"},
	{"provider \"aws\"", "AWS (Terraform provider)"},
}

func detectAWSHints(files []string, repoPath string) []AWSHint {
	var hints []AWSHint
	// Only scan source-like files
	scanExts := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true, ".java": true,
		".cs": true, ".rb": true, ".php": true, ".tf": true, ".yml": true,
		".yaml": true, ".json": true, ".toml": true, ".cfg": true, ".ini": true,
		".sh": true, ".bash": true, ".md": true, ".txt": true,
	}
	seen := map[string]bool{} // deduplicate service+file

	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if !scanExts[ext] {
			continue
		}
		full := filepath.Join(repoPath, filepath.FromSlash(f))
		fh, err := os.Open(full)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(fh)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			for _, p := range awsPatterns {
				if strings.Contains(line, p.Pattern) {
					key := p.Service + ":" + f
					if !seen[key] {
						seen[key] = true
						hints = append(hints, AWSHint{
							Service:  p.Service,
							File:     f,
							Line:     lineNum,
							Evidence: truncate(strings.TrimSpace(line), 120),
						})
					}
				}
			}
		}
		fh.Close()
	}
	// Sort for determinism
	sort.Slice(hints, func(i, j int) bool {
		if hints[i].Service != hints[j].Service {
			return hints[i].Service < hints[j].Service
		}
		if hints[i].File != hints[j].File {
			return hints[i].File < hints[j].File
		}
		return hints[i].Line < hints[j].Line
	})
	return hints
}

// containsString checks if a file contains a specific string (first 200 lines).
func containsString(path, s string) bool {
	fh, err := os.Open(path)
	if err != nil {
		return false
	}
	defer fh.Close()
	scanner := bufio.NewScanner(fh)
	count := 0
	for scanner.Scan() {
		count++
		if count > 200 {
			break
		}
		if strings.Contains(scanner.Text(), s) {
			return true
		}
	}
	return false
}

func readFirstLines(path string, n int) []string {
	fh, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer fh.Close()
	var lines []string
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() && len(lines) < n {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
