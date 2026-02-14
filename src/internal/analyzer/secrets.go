package analyzer

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SecretFinding records a potential secret or credential found in a file.
type SecretFinding struct {
	File        string
	Line        int
	Type        string // e.g. "AWS Access Key", "GitHub Token", "Password"
	Severity    string // "HIGH", "MEDIUM", "LOW"
	Evidence    string // redacted value showing prefix/suffix only
	Remediation string
}

// secretPattern defines a regex pattern for detecting secrets.
type secretPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Severity    string
	Remediation string
}

var secretPatterns = []secretPattern{
	{
		Name:        "AWS Access Key ID",
		Pattern:     regexp.MustCompile(`(?i)(AKIA[0-9A-Z]{16})`),
		Severity:    "HIGH",
		Remediation: "Remove AWS access key. Use Azure Managed Identity instead, or store in Key Vault.",
	},
	{
		Name:        "AWS Secret Access Key",
		Pattern:     regexp.MustCompile(`(?i)(aws_secret_access_key|aws_secret_key)\s*[=:]\s*["']?([A-Za-z0-9/+=]{40})["']?`),
		Severity:    "HIGH",
		Remediation: "Remove AWS secret key. Migrate to Azure Managed Identity or Key Vault.",
	},
	{
		Name:        "GitHub Token",
		Pattern:     regexp.MustCompile(`(ghp_[A-Za-z0-9]{36}|gho_[A-Za-z0-9]{36}|ghu_[A-Za-z0-9]{36}|ghs_[A-Za-z0-9]{36}|ghr_[A-Za-z0-9]{36})`),
		Severity:    "HIGH",
		Remediation: "Rotate GitHub token immediately. Use GitHub App or OIDC for CI/CD.",
	},
	{
		Name:        "Generic API Key assignment",
		Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[=:]\s*["']([^"'\s]{8,})["']`),
		Severity:    "MEDIUM",
		Remediation: "Move API key to Azure Key Vault. Reference via Managed Identity.",
	},
	{
		Name:        "Password assignment",
		Pattern:     regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*["']([^"'\s]{4,})["']`),
		Severity:    "HIGH",
		Remediation: "Remove hardcoded password. Use Managed Identity or Key Vault.",
	},
	{
		Name:        "Connection string with password",
		Pattern:     regexp.MustCompile(`(?i)(connection[_-]?string|connstr|database_url)\s*[=:]\s*["']([^"']*[Pp]assword=[^"'\s]+[^"']*)["']`),
		Severity:    "HIGH",
		Remediation: "Use Managed Identity for database access. If not possible, store connection string in Key Vault.",
	},
	{
		Name:        "SECRET= assignment",
		Pattern:     regexp.MustCompile(`(?i)(secret|secret[_-]?key)\s*[=:]\s*["']([^"'\s]{8,})["']`),
		Severity:    "HIGH",
		Remediation: "Remove hardcoded secret. Store in Azure Key Vault.",
	},
	{
		Name:        "Private key marker",
		Pattern:     regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
		Severity:    "HIGH",
		Remediation: "Remove private key from repository. Store in Key Vault or use Managed Identity certificates.",
	},
	{
		Name:        "Bearer token",
		Pattern:     regexp.MustCompile(`(?i)(bearer)\s+[A-Za-z0-9\-._~+/]+=*`),
		Severity:    "MEDIUM",
		Remediation: "Remove hardcoded bearer token. Use dynamic token acquisition via Managed Identity.",
	},
	{
		Name:        "Azure Storage Key",
		Pattern:     regexp.MustCompile(`(?i)(AccountKey|storage[_-]?key)\s*[=:]\s*["']?([A-Za-z0-9+/]{86}==)["']?`),
		Severity:    "HIGH",
		Remediation: "Use Managed Identity with RBAC for Azure Storage access.",
	},
}

// knownSecretFiles are filenames that commonly contain secrets.
var knownSecretFiles = map[string]bool{
	".env":            true,
	".env.local":      true,
	".env.production": true,
	"secrets.yaml":    true,
	"secrets.yml":     true,
	"secrets.json":    true,
	"credentials":     true,
	".npmrc":          true,
	".pypirc":         true,
}

// DetectSecrets scans repository files for potential secret patterns.
// Values are redacted in the output (prefix + "REDACTED" + suffix).
func DetectSecrets(files []string, repoPath string) []SecretFinding {
	var findings []SecretFinding

	// Binary-like extensions to skip
	skipExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
		".svg": true, ".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
		".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".exe": true,
		".dll": true, ".so": true, ".dylib": true, ".bin": true, ".pdf": true,
		".lock": true,
	}

	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if skipExts[ext] {
			continue
		}

		base := strings.ToLower(filepath.Base(f))

		// Flag known secret files
		if knownSecretFiles[base] {
			findings = append(findings, SecretFinding{
				File:        f,
				Line:        0,
				Type:        "Known secret file",
				Severity:    "MEDIUM",
				Evidence:    base + " should not be committed",
				Remediation: "Add to .gitignore. Move secrets to Azure Key Vault.",
			})
		}

		// Scan file content for patterns
		full := filepath.Join(repoPath, filepath.FromSlash(f))
		lineFindings := scanFileForSecrets(full, f)
		findings = append(findings, lineFindings...)
	}

	// Sort for determinism
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		return findings[i].Line < findings[j].Line
	})

	return findings
}

// scanFileForSecrets scans a single file for secret patterns.
func scanFileForSecrets(fullPath, relPath string) []SecretFinding {
	var findings []SecretFinding

	fh, err := os.Open(fullPath)
	if err != nil {
		return nil
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip comment-only lines that are examples/documentation
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "<!--") {
			// Still check — comments can contain real secrets
			// but reduce severity
		}

		for _, sp := range secretPatterns {
			matches := sp.Pattern.FindStringSubmatch(line)
			if len(matches) > 0 {
				evidence := redactMatch(matches[0])
				findings = append(findings, SecretFinding{
					File:        relPath,
					Line:        lineNum,
					Type:        sp.Name,
					Severity:    sp.Severity,
					Evidence:    evidence,
					Remediation: sp.Remediation,
				})
				break // one finding per line per file to avoid duplicates
			}
		}
	}

	return findings
}

// redactMatch redacts a matched secret value, showing only prefix and suffix.
func redactMatch(match string) string {
	if len(match) <= 8 {
		return "REDACTED"
	}
	prefix := match[:4]
	suffix := match[len(match)-4:]
	return prefix + "...REDACTED..." + suffix
}

// RedactValue redacts a secret value for safe output.
func RedactValue(value string) string {
	return redactMatch(value)
}
