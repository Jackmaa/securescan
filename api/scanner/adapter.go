package scanner

import (
	"context" // Adapters accept context for cancellation/timeouts during tool execution.

	"securescan/models" // Findings are normalized into shared model types.

	"github.com/google/uuid" // Scan IDs are UUIDs and are assigned to each produced Finding.
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

// init registers all built-in adapters.
//
// Why a registry (instead of hardcoding in the scan pipeline):
// - Keeps the scan orchestrator decoupled from specific tools.
// - Makes it easy to add/remove tools in one place.
// - Enables future configuration (e.g., enable/disable per org) by filtering this list.
func init() {
	Registry = []ToolAdapter{
		&SemgrepAdapter{},
		&TruffleHogAdapter{},
		&NpmAuditAdapter{},
		&ESLintSecurityAdapter{},
	}
}
