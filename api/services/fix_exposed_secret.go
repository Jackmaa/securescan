package services

import (
	"context"
	"strings"

	"securescan/models"

	"github.com/google/uuid"
)

// ExposedSecretFixer generates fixes for hardcoded secrets and credentials.
//
// Detection: tool = trufflehog, or CWE-798, or keywords like "secret", "credential".
// Fix strategy: replace hardcoded value with environment variable reference.
type ExposedSecretFixer struct{}

func (f *ExposedSecretFixer) CanFix(finding models.Finding) bool {
	if finding.ToolName == "trufflehog" {
		return true
	}
	if finding.CweID != nil && *finding.CweID == "CWE-798" {
		return true
	}
	msg := strings.ToLower(finding.Message)
	return strings.Contains(msg, "exposed secret") || strings.Contains(msg, "hardcoded")
}

func (f *ExposedSecretFixer) Generate(_ context.Context, finding models.Finding, sourceCode string) (*models.Fix, error) {
	// Derive an env var name from the detector/rule
	envVarName := "SECRET_VALUE"
	if finding.RuleID != nil {
		envVarName = strings.ToUpper(strings.ReplaceAll(*finding.RuleID, " ", "_")) + "_KEY"
	}

	explanation := "Hardcoded secrets in source code are a critical security risk. " +
		"Anyone with repository access can extract the credentials. " +
		"Move the value to an environment variable and reference it at runtime. " +
		"Rotate the exposed secret immediately — it should be considered compromised."

	fixedCode := "// Replace the hardcoded secret with an environment variable:\n"
	if strings.Contains(sourceCode, "process.env") || strings.Contains(sourceCode, "require(") || strings.Contains(sourceCode, "import ") {
		fixedCode += "const secret = process.env." + envVarName + ";"
	} else if strings.Contains(sourceCode, "os.Getenv") || strings.Contains(sourceCode, "package ") {
		fixedCode += `secret := os.Getenv("` + envVarName + `")`
	} else if strings.Contains(sourceCode, "$_ENV") || strings.Contains(sourceCode, "<?php") {
		fixedCode += `$secret = getenv('` + envVarName + `');`
	} else {
		fixedCode += "// Read from environment variable: " + envVarName + "\n" +
			"// JavaScript: process.env." + envVarName + "\n" +
			"// Python: os.environ['" + envVarName + "']\n" +
			"// Go: os.Getenv(\"" + envVarName + "\")"
	}

	filePath := ""
	if finding.FilePath != nil {
		filePath = *finding.FilePath
	}

	fix := &models.Fix{
		ID:           uuid.New(),
		FindingID:    finding.ID,
		FixType:      "template",
		Description:  "Replace hardcoded secret with environment variable",
		Explanation:  &explanation,
		OriginalCode: &sourceCode,
		FixedCode:    &fixedCode,
		FilePath:     filePath,
		LineStart:    finding.LineStart,
		LineEnd:      finding.LineEnd,
		Status:       "pending",
	}
	return fix, nil
}
