package services

import (
	"context"
	"strings"

	"securescan/models"

	"github.com/google/uuid"
)

// XSSFixer generates fixes for Cross-Site Scripting vulnerabilities.
//
// Detection: CWE-79 or rule IDs containing "xss".
// Fix strategy: wrap output with appropriate escaping/sanitization for the language.
type XSSFixer struct{}

func (f *XSSFixer) CanFix(finding models.Finding) bool {
	if finding.CweID != nil && *finding.CweID == "CWE-79" {
		return true
	}
	if finding.RuleID != nil && strings.Contains(strings.ToLower(*finding.RuleID), "xss") {
		return true
	}
	msg := strings.ToLower(finding.Message)
	return strings.Contains(msg, "cross-site scripting") || strings.Contains(msg, "xss")
}

func (f *XSSFixer) Generate(_ context.Context, finding models.Finding, sourceCode string) (*models.Fix, error) {
	explanation := "Cross-Site Scripting (XSS) allows attackers to inject malicious scripts " +
		"into web pages viewed by other users. Always escape or sanitize user-controlled " +
		"data before rendering it in HTML."

	// The fix recommends the safe DOM API or sanitization library for the detected language.
	fixedCode := "// Use safe DOM APIs or sanitization:\n" +
		"import DOMPurify from 'dompurify';\n" +
		"// For plain text: element.textContent = userInput;\n" +
		"// For HTML that must render: DOMPurify.sanitize(userInput)"

	if strings.Contains(sourceCode, "htmlspecialchars") || strings.Contains(sourceCode, "<?php") {
		fixedCode = "echo htmlspecialchars($userInput, ENT_QUOTES, 'UTF-8');"
	}

	filePath := ""
	if finding.FilePath != nil {
		filePath = *finding.FilePath
	}

	fix := &models.Fix{
		ID:           uuid.New(),
		FindingID:    finding.ID,
		FixType:      "template",
		Description:  "Escape or sanitize user input before rendering in HTML",
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
