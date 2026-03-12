package services

import (
	"context"
	"strings"

	"securescan/models"

	"github.com/google/uuid"
)

// SQLInjectionFixer generates fixes for SQL injection vulnerabilities.
//
// Detection: CWE-89 or rule IDs containing "sql-injection".
// Fix strategy: replace string concatenation with parameterized queries.
// The fix is language-aware: different placeholder syntax for different ecosystems.
type SQLInjectionFixer struct{}

func (f *SQLInjectionFixer) CanFix(finding models.Finding) bool {
	if finding.CweID != nil && *finding.CweID == "CWE-89" {
		return true
	}
	if finding.RuleID != nil && strings.Contains(strings.ToLower(*finding.RuleID), "sql-injection") {
		return true
	}
	return strings.Contains(strings.ToLower(finding.Message), "sql injection")
}

func (f *SQLInjectionFixer) Generate(_ context.Context, finding models.Finding, sourceCode string) (*models.Fix, error) {
	explanation := "SQL injection occurs when user input is concatenated directly into SQL queries. " +
		"Use parameterized queries (prepared statements) instead, which separate SQL structure from data."

	// Language-specific fix examples
	fixedCode := "// Use parameterized queries instead of string concatenation:\n"
	if strings.Contains(sourceCode, "mysql") || strings.Contains(sourceCode, "pg") || strings.Contains(sourceCode, "sql.") {
		fixedCode += `db.Query("SELECT * FROM users WHERE id = $1", userInput)`
	} else if strings.Contains(sourceCode, "sequelize") || strings.Contains(sourceCode, "knex") {
		fixedCode += `Model.findAll({ where: { id: userInput } })`
	} else {
		fixedCode += `db.query("SELECT * FROM users WHERE id = ?", [userInput])`
	}

	filePath := ""
	if finding.FilePath != nil {
		filePath = *finding.FilePath
	}

	fix := &models.Fix{
		ID:           uuid.New(),
		FindingID:    finding.ID,
		FixType:      "template",
		Description:  "Replace string-concatenated SQL with parameterized query",
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
