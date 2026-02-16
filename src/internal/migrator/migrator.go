// Package migrator generates AWS-to-Azure migration assessment reports.
package migrator

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/jpamies/aegisctl/internal/analyzer"
	"github.com/jpamies/aegisctl/internal/output"
)

// ServiceMapping defines an AWS-to-Azure service mapping.
type ServiceMapping struct {
	AWSService      string
	AzureEquivalent string
	Confidence      string
	Notes           string
}

// MigrationData holds the template data for migration reports.
type MigrationData struct {
	RepoPath string
	AWSHints []analyzer.AWSHint
	Mappings []ServiceMapping
}

// awsToAzureMap is the known mapping of AWS services to Azure equivalents.
var awsToAzureMap = map[string]ServiceMapping{
	"S3": {
		AWSService:      "S3",
		AzureEquivalent: "Azure Blob Storage",
		Confidence:      "High",
		Notes:           "API differs; SDK change required.",
	},
	"Lambda": {
		AWSService:      "Lambda",
		AzureEquivalent: "Azure Functions (Consumption plan)",
		Confidence:      "High",
		Notes:           "Handler signature differs. Triggers map to bindings.",
	},
	"DynamoDB": {
		AWSService:      "DynamoDB",
		AzureEquivalent: "Azure Cosmos DB (NoSQL API)",
		Confidence:      "Medium",
		Notes:           "Data model differs. Partition key strategy review needed.",
	},
	"SNS": {
		AWSService:      "SNS",
		AzureEquivalent: "Azure Service Bus Topics / Event Grid",
		Confidence:      "Medium",
		Notes:           "Fan-out patterns differ.",
	},
	"SQS": {
		AWSService:      "SQS",
		AzureEquivalent: "Azure Service Bus Queues / Queue Storage",
		Confidence:      "High",
		Notes:           "Standard queuing. Consider Service Bus for advanced features.",
	},
	"ECS": {
		AWSService:      "ECS",
		AzureEquivalent: "Azure Container Apps / AKS",
		Confidence:      "Medium",
		Notes:           "Container Apps preferred for simplicity. AKS for Kubernetes compatibility.",
	},
	"EKS": {
		AWSService:      "EKS",
		AzureEquivalent: "Azure Kubernetes Service (AKS)",
		Confidence:      "High",
		Notes:           "Direct Kubernetes equivalent.",
	},
	"CloudWatch": {
		AWSService:      "CloudWatch",
		AzureEquivalent: "Azure Monitor + Application Insights",
		Confidence:      "High",
		Notes:           "Metrics and logs map well. Dashboard recreation needed.",
	},
	"CloudFormation": {
		AWSService:      "CloudFormation",
		AzureEquivalent: "Bicep",
		Confidence:      "High",
		Notes:           "IaC rewrite required. Bicep is the recommended Azure IaC.",
	},
	"AWS SDK": {
		AWSService:      "AWS SDK",
		AzureEquivalent: "Azure SDK",
		Confidence:      "High",
		Notes:           "SDK replacement across the codebase.",
	},
	"AWS SDK (Python/boto3)": {
		AWSService:      "AWS SDK (Python/boto3)",
		AzureEquivalent: "Azure SDK for Python",
		Confidence:      "High",
		Notes:           "Replace boto3 with azure-* packages.",
	},
	"AWS SDK v3 (JS)": {
		AWSService:      "AWS SDK v3 (JS)",
		AzureEquivalent: "Azure SDK for JavaScript",
		Confidence:      "High",
		Notes:           "Replace @aws-sdk/* with @azure/* packages.",
	},
	"AWS CLI": {
		AWSService:      "AWS CLI",
		AzureEquivalent: "Azure CLI",
		Confidence:      "High",
		Notes:           "Command mapping needed in scripts.",
	},
	"AWS API endpoint": {
		AWSService:      "AWS API endpoints",
		AzureEquivalent: "Azure REST API / SDK",
		Confidence:      "Medium",
		Notes:           "Endpoint URLs and auth mechanism changes required.",
	},
	"AWS (Terraform provider)": {
		AWSService:      "Terraform AWS Provider",
		AzureEquivalent: "Bicep (preferred) or Terraform AzureRM",
		Confidence:      "High",
		Notes:           "Migrate to Bicep. Terraform AzureRM is an alternative.",
	},
}

// MigrateAWS generates an AWS migration assessment report.
func MigrateAWS(repoPath, outputDir string) error {
	findings, err := analyzer.Analyze(repoPath)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	data := buildMigrationData(findings)

	funcMap := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}

	t, err := template.New("migration").Funcs(funcMap).Parse(output.AWSMigrationTmpl)
	if err != nil {
		return fmt.Errorf("parsing migration template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing migration template: %w", err)
	}

	return output.WriteFile(filepath.Join(outputDir, "docs", "AWS_MIGRATION.md"), buf.String())
}

func buildMigrationData(f *analyzer.Findings) MigrationData {
	data := MigrationData{
		RepoPath: f.RepoPath,
		AWSHints: f.AWSHints,
	}

	// Build unique service list for mapping
	seen := map[string]bool{}
	var services []string
	for _, h := range f.AWSHints {
		if !seen[h.Service] {
			seen[h.Service] = true
			services = append(services, h.Service)
		}
	}
	sort.Strings(services)

	for _, svc := range services {
		if m, ok := awsToAzureMap[svc]; ok {
			data.Mappings = append(data.Mappings, m)
		} else {
			data.Mappings = append(data.Mappings, ServiceMapping{
				AWSService:      svc,
				AzureEquivalent: "Needs manual assessment",
				Confidence:      "Low",
				Notes:           "No automated mapping available. Manual review required.",
			})
		}
	}

	return data
}
