package services

import (
	"context"
	"strings"

	"securescan/models"

	"github.com/google/uuid"
)

// PlaintextPasswordFixer generates fixes for plaintext password storage.
//
// Detection: CWE-256/259 or rule IDs containing "plaintext-password".
// Fix strategy: add bcrypt/argon2 hashing with language-appropriate code.
type PlaintextPasswordFixer struct{}

func (f *PlaintextPasswordFixer) CanFix(finding models.Finding) bool {
	if finding.CweID != nil {
		cwe := *finding.CweID
		if cwe == "CWE-256" || cwe == "CWE-259" {
			return true
		}
	}
	if finding.RuleID != nil {
		rule := strings.ToLower(*finding.RuleID)
		if strings.Contains(rule, "plaintext-password") || strings.Contains(rule, "plaintext_password") {
			return true
		}
	}
	msg := strings.ToLower(finding.Message)
	return strings.Contains(msg, "plaintext password") || strings.Contains(msg, "password in plain")
}

func (f *PlaintextPasswordFixer) Generate(_ context.Context, finding models.Finding, sourceCode string) (*models.Fix, error) {
	explanation := "Storing passwords in plaintext allows anyone with database access to read them directly. " +
		"Always hash passwords using a slow, salted algorithm like bcrypt or argon2 before storage. " +
		"These algorithms are designed to be computationally expensive, making brute-force attacks impractical."

	fixedCode := "// Hash passwords before storing:\n"
	if strings.Contains(sourceCode, "bcrypt") || strings.Contains(sourceCode, "require(") || strings.Contains(sourceCode, "import ") {
		fixedCode += "import bcrypt from 'bcryptjs';\n\n" +
			"const saltRounds = 12;\n" +
			"const hashedPassword = await bcrypt.hash(plainPassword, saltRounds);\n" +
			"// Store hashedPassword in DB\n\n" +
			"// To verify:\n" +
			"const isValid = await bcrypt.compare(inputPassword, hashedPassword);"
	} else if strings.Contains(sourceCode, "golang.org") || strings.Contains(sourceCode, "package ") {
		fixedCode += "import \"golang.org/x/crypto/bcrypt\"\n\n" +
			"hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)\n" +
			"// Store hashed in DB\n\n" +
			"// To verify:\n" +
			"err = bcrypt.CompareHashAndPassword(hashed, []byte(inputPassword))"
	} else if strings.Contains(sourceCode, "<?php") || strings.Contains(sourceCode, "password_hash") {
		fixedCode += "$hashedPassword = password_hash($plainPassword, PASSWORD_BCRYPT);\n" +
			"// Store $hashedPassword in DB\n\n" +
			"// To verify:\n" +
			"$isValid = password_verify($inputPassword, $hashedPassword);"
	} else {
		fixedCode += "// Use bcrypt (recommended) or argon2:\n" +
			"// Node.js: bcryptjs or argon2 packages\n" +
			"// Python: bcrypt or passlib packages\n" +
			"// Go: golang.org/x/crypto/bcrypt\n" +
			"// PHP: password_hash() built-in"
	}

	filePath := ""
	if finding.FilePath != nil {
		filePath = *finding.FilePath
	}

	fix := &models.Fix{
		ID:           uuid.New(),
		FindingID:    finding.ID,
		FixType:      "template",
		Description:  "Hash passwords with bcrypt instead of storing in plaintext",
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
