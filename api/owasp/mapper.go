package owasp

import (
	"strings"

	"securescan/models"
)

// OwaspMapping holds a resolved OWASP category and its human-readable label.
type OwaspMapping struct {
	Category string // "A01" through "A10"
	Label    string
}

// MapFinding applies the 3-layer OWASP mapping strategy to a single finding.
//
// Priority chain (first match wins):
//   1. Direct metadata — if the tool already set OwaspCategory, trust it.
//   2. CWE→OWASP lookup — if the finding has a CWE ID, consult the static table.
//   3. Heuristic fallback — rule ID and message keyword matching for tools
//      that provide neither OWASP nor CWE metadata.
func MapFinding(f *models.Finding) {
	// Layer 1: Tool already provided OWASP metadata (e.g., Semgrep)
	if f.OwaspCategory != nil && *f.OwaspCategory != "" {
		if f.OwaspLabel == nil {
			label := labelFor(*f.OwaspCategory)
			f.OwaspLabel = &label
		}
		return
	}

	// Layer 2: CWE→OWASP lookup table
	if f.CweID != nil && *f.CweID != "" {
		cwe := normalizeCWE(*f.CweID)
		if mapping, ok := CWEToOWASP[cwe]; ok {
			f.OwaspCategory = &mapping.Category
			f.OwaspLabel = &mapping.Label
			return
		}
	}

	// Layer 3: Heuristic fallback based on tool name, rule ID, and message content
	if mapping := heuristicMap(f); mapping != nil {
		f.OwaspCategory = &mapping.Category
		f.OwaspLabel = &mapping.Label
	}
}

// MapFindings applies OWASP mapping to all findings in a slice.
func MapFindings(findings []models.Finding) {
	for i := range findings {
		MapFinding(&findings[i])
	}
}

// normalizeCWE handles variations like "CWE-89", "cwe-89", "89"
func normalizeCWE(cwe string) string {
	cwe = strings.TrimSpace(cwe)
	cwe = strings.ToUpper(cwe)
	if !strings.HasPrefix(cwe, "CWE-") {
		cwe = "CWE-" + cwe
	}
	return cwe
}

// heuristicMap infers OWASP category from tool name, rule IDs, and message keywords.
// This is the lowest-confidence layer, but it ensures coverage for tools like
// TruffleHog and npm audit that don't emit CWE metadata.
func heuristicMap(f *models.Finding) *OwaspMapping {
	// Tool-level heuristics
	switch f.ToolName {
	case "trufflehog":
		return &OwaspMapping{Category: "A07", Label: "Identification and Authentication Failures"}
	case "npm_audit":
		return &OwaspMapping{Category: "A06", Label: "Vulnerable and Outdated Components"}
	}

	// Rule ID and message keyword heuristics
	ruleID := ""
	if f.RuleID != nil {
		ruleID = strings.ToLower(*f.RuleID)
	}
	msg := strings.ToLower(f.Message)
	combined := ruleID + " " + msg

	keywordMap := []struct {
		keywords []string
		mapping  OwaspMapping
	}{
		{[]string{"sql-injection", "sqli", "sql injection"}, OwaspMapping{"A03", "Injection"}},
		{[]string{"xss", "cross-site scripting", "cross site scripting"}, OwaspMapping{"A03", "Injection"}},
		{[]string{"command-injection", "os-command", "shell-injection"}, OwaspMapping{"A03", "Injection"}},
		{[]string{"eval", "code-injection", "code injection"}, OwaspMapping{"A03", "Injection"}},
		{[]string{"hardcoded", "hard-coded", "secret", "credential", "password", "api-key", "apikey"}, OwaspMapping{"A07", "Identification and Authentication Failures"}},
		{[]string{"crypto", "weak-hash", "md5", "sha1", "des", "insecure-cipher"}, OwaspMapping{"A02", "Cryptographic Failures"}},
		{[]string{"csrf", "cross-site request"}, OwaspMapping{"A01", "Broken Access Control"}},
		{[]string{"path-traversal", "directory-traversal", "lfi", "rfi"}, OwaspMapping{"A01", "Broken Access Control"}},
		{[]string{"ssrf", "server-side request"}, OwaspMapping{"A10", "Server-Side Request Forgery"}},
		{[]string{"deserializ", "insecure-deserial"}, OwaspMapping{"A08", "Software and Data Integrity Failures"}},
		{[]string{"vulnerable-dep", "outdated", "known-vulnerability"}, OwaspMapping{"A06", "Vulnerable and Outdated Components"}},
		{[]string{"misconfigur", "debug-mode", "default-password"}, OwaspMapping{"A05", "Security Misconfiguration"}},
	}

	for _, entry := range keywordMap {
		for _, kw := range entry.keywords {
			if strings.Contains(combined, kw) {
				return &entry.mapping
			}
		}
	}

	return nil
}

var categoryLabels = map[string]string{
	"A01": "Broken Access Control",
	"A02": "Cryptographic Failures",
	"A03": "Injection",
	"A04": "Insecure Design",
	"A05": "Security Misconfiguration",
	"A06": "Vulnerable and Outdated Components",
	"A07": "Identification and Authentication Failures",
	"A08": "Software and Data Integrity Failures",
	"A09": "Security Logging and Monitoring Failures",
	"A10": "Server-Side Request Forgery",
}

func labelFor(category string) string {
	if label, ok := categoryLabels[category]; ok {
		return label
	}
	return category
}
