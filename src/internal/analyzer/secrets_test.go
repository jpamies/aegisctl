package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSecrets_AWSKey(t *testing.T) {
	// Create a temp dir with a file containing a fake AWS key
	dir := t.TempDir()
	content := `# Config file
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
some other line
`
	if err := os.WriteFile(filepath.Join(dir, "config.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	files := []string{"config.txt"}
	findings := DetectSecrets(files, dir)

	if len(findings) == 0 {
		t.Fatal("expected at least one secret finding for AWS key")
	}

	found := false
	for _, f := range findings {
		if f.Type == "AWS Access Key ID" {
			found = true
			if f.Line != 2 {
				t.Errorf("expected line 2, got %d", f.Line)
			}
			if f.Severity != "HIGH" {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
			// Verify the value is redacted
			if f.Evidence == "AKIAIOSFODNN7EXAMPLE" {
				t.Error("evidence should be redacted, got raw value")
			}
		}
	}
	if !found {
		t.Error("AWS Access Key ID finding not found")
	}
}

func TestDetectSecrets_GitHubToken(t *testing.T) {
	dir := t.TempDir()
	content := `TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij
`
	if err := os.WriteFile(filepath.Join(dir, "env.sh"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	files := []string{"env.sh"}
	findings := DetectSecrets(files, dir)

	if len(findings) == 0 {
		t.Fatal("expected at least one secret finding for GitHub token")
	}

	found := false
	for _, f := range findings {
		if f.Type == "GitHub Token" {
			found = true
			if f.Severity != "HIGH" {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("GitHub Token finding not found")
	}
}

func TestDetectSecrets_Password(t *testing.T) {
	dir := t.TempDir()
	content := `database:
  password: "SuperSecret123!"
  host: localhost
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	files := []string{"config.yaml"}
	findings := DetectSecrets(files, dir)

	if len(findings) == 0 {
		t.Fatal("expected at least one secret finding for password")
	}

	found := false
	for _, f := range findings {
		if f.Type == "Password assignment" {
			found = true
			if f.Line != 2 {
				t.Errorf("expected line 2, got %d", f.Line)
			}
		}
	}
	if !found {
		t.Error("Password finding not found")
	}
}

func TestDetectSecrets_KnownSecretFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FOO=bar\n"), 0644); err != nil {
		t.Fatal(err)
	}

	files := []string{".env"}
	findings := DetectSecrets(files, dir)

	found := false
	for _, f := range findings {
		if f.Type == "Known secret file" {
			found = true
		}
	}
	if !found {
		t.Error("expected .env to be flagged as known secret file")
	}
}

func TestDetectSecrets_NoSecrets(t *testing.T) {
	dir := t.TempDir()
	content := `package main

func main() {
	fmt.Println("hello world")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	files := []string{"main.go"}
	findings := DetectSecrets(files, dir)

	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestDetectSecrets_PrivateKey(t *testing.T) {
	dir := t.TempDir()
	content := `-----BEGIN RSA PRIVATE KEY-----
MIIBogIBAAJBALRTnQ9kM4EXAMPLE
-----END RSA PRIVATE KEY-----
`
	if err := os.WriteFile(filepath.Join(dir, "key.pem"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	files := []string{"key.pem"}
	findings := DetectSecrets(files, dir)

	found := false
	for _, f := range findings {
		if f.Type == "Private key marker" {
			found = true
			if f.Severity != "HIGH" {
				t.Errorf("expected HIGH severity, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("Private key finding not found")
	}
}

func TestRedactValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"short", "REDACTED"},
		{"AKIAIOSFODNN7EXAMPLE", "AKIA...REDACTED...MPLE"},
		{"ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij", "ghp_...REDACTED...ghij"},
	}

	for _, tt := range tests {
		got := RedactValue(tt.input)
		if got != tt.want {
			t.Errorf("RedactValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
