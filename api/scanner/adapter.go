package scanner

import (
	"context"

	"securescan/models"

	"github.com/google/uuid"
)

// ToolAdapter defines the contract for each security tool integration.
// Each adapter knows how to run a specific tool and parse its output
// into normalized Finding objects.
type ToolAdapter interface {
	Name() string
	IsApplicable(languages []string) bool
	Run(ctx context.Context, repoPath string) ([]byte, error)
	Parse(scanID uuid.UUID, raw []byte) ([]models.Finding, error)
}

// Registry holds all available tool adapters. During scan orchestration,
// we iterate over this and check IsApplicable() against the project's languages.
var Registry []ToolAdapter

func init() {
	Registry = []ToolAdapter{
		&SemgrepAdapter{},
		&TruffleHogAdapter{},
		&NpmAuditAdapter{},
		&ESLintSecurityAdapter{},
	}
}
