package services

import (
	"context"
	"fmt"
	"strings"

	"securescan/config"
	"securescan/models"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
)

// AIFixGenerator uses Claude to produce context-aware fix suggestions.
//
// Unlike template generators, the AI fixer:
//   - Analyzes the actual vulnerable code + surrounding context
//   - Produces tailored fixes rather than generic patterns
//   - Includes pedagogical explanations (why the code is vulnerable, how the fix works)
//
// The AI fixer is invoked on-demand per finding (via the "Get AI Fix" button), not
// automatically during the scan pipeline, to control API costs.
type AIFixGenerator struct {
	APIKey string
}

func NewAIFixGenerator(cfg *config.Config) *AIFixGenerator {
	return &AIFixGenerator{APIKey: cfg.AnthropicKey}
}

func (g *AIFixGenerator) CanFix(finding models.Finding) bool {
	// AI can attempt a fix for any finding that has source code context
	return g.APIKey != "" && finding.FilePath != nil
}

func (g *AIFixGenerator) Generate(ctx context.Context, finding models.Finding, sourceCode string) (*models.Fix, error) {
	if g.APIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not configured")
	}
	if sourceCode == "" {
		return nil, fmt.Errorf("no source code context available")
	}

	client := anthropic.NewClient(option.WithAPIKey(g.APIKey))

	owaspInfo := ""
	if finding.OwaspCategory != nil {
		owaspInfo = fmt.Sprintf(" (OWASP %s)", *finding.OwaspCategory)
	}
	cweInfo := ""
	if finding.CweID != nil {
		cweInfo = fmt.Sprintf(" (%s)", *finding.CweID)
	}

	prompt := fmt.Sprintf(`You are a security engineer reviewing vulnerable code. A security scanner found the following issue:

**Tool:** %s
**Rule:** %s
**Severity:** %s%s%s
**Message:** %s
**File:** %s (line %s)

**Vulnerable code context:**
%s

Please provide:
1. A FIXED version of the vulnerable code (just the code, minimal changes)
2. A brief explanation of WHY the original code is vulnerable and HOW the fix addresses it

Format your response exactly as:
FIXED_CODE:
<the fixed code>

EXPLANATION:
<your explanation>`,
		finding.ToolName,
		deref(finding.RuleID),
		finding.Severity, owaspInfo, cweInfo,
		finding.Message,
		deref(finding.FilePath),
		lineRange(finding.LineStart, finding.LineEnd),
		sourceCode,
	)

	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-sonnet-4-5-20250514",
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude API call failed: %w", err)
	}

	// Extract text from the response
	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText += block.Text
		}
	}

	fixedCode, explanation := parseAIResponse(responseText)

	filePath := ""
	if finding.FilePath != nil {
		filePath = *finding.FilePath
	}

	fix := &models.Fix{
		ID:           uuid.New(),
		FindingID:    finding.ID,
		FixType:      "ai",
		Description:  "AI-generated fix for: " + finding.Message,
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

// parseAIResponse extracts the fixed code and explanation from the structured response.
func parseAIResponse(response string) (fixedCode, explanation string) {
	parts := strings.SplitN(response, "EXPLANATION:", 2)
	if len(parts) == 2 {
		explanation = strings.TrimSpace(parts[1])
	}

	codePart := parts[0]
	codePart = strings.TrimPrefix(codePart, "FIXED_CODE:")
	codePart = strings.TrimSpace(codePart)
	// Strip markdown code fences if present
	codePart = strings.TrimPrefix(codePart, "```")
	if idx := strings.Index(codePart, "\n"); idx > 0 && idx < 20 {
		// Remove language identifier like "```javascript\n"
		codePart = codePart[idx+1:]
	}
	codePart = strings.TrimSuffix(codePart, "```")
	fixedCode = strings.TrimSpace(codePart)

	return fixedCode, explanation
}

func deref(s *string) string {
	if s == nil {
		return "N/A"
	}
	return *s
}

func lineRange(start, end *int) string {
	if start == nil {
		return "unknown"
	}
	if end == nil || *end == *start {
		return fmt.Sprintf("%d", *start)
	}
	return fmt.Sprintf("%d-%d", *start, *end)
}
