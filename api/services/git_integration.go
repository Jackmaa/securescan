package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"securescan/config"
	"securescan/models"

	"github.com/google/uuid"
)

// GitIntegrationService handles creating branches, applying fixes, committing, and pushing.
//
// The workflow:
//   1. Gather all accepted fixes for a scan
//   2. Create a new branch (fix/securescan-YYYY-MM-DD)
//   3. Apply fixes in REVERSE line order to preserve line numbers
//   4. Commit with a structured message listing what was fixed
//   5. Push via git CLI using the configured PAT
//   6. Create a PR via GitHub API (optional)
type GitIntegrationService struct {
	DB          interface{ Exec(ctx context.Context, sql string, args ...any) (interface{ RowsAffected() int64 }, error) }
	FixSvc      *FixService
	GithubToken string
}

func NewGitIntegrationService(fixSvc *FixService, cfg *config.Config) *GitIntegrationService {
	return &GitIntegrationService{
		FixSvc:      fixSvc,
		GithubToken: cfg.GithubToken,
	}
}

// ApplyFixes creates a branch, applies accepted fixes, commits, and pushes.
// Returns the branch name and any error.
func (s *GitIntegrationService) ApplyFixes(ctx context.Context, scanID uuid.UUID, repoPath string) (string, error) {
	// Gather accepted fixes
	fixes, err := s.FixSvc.ListByScan(ctx, scanID)
	if err != nil {
		return "", fmt.Errorf("list fixes: %w", err)
	}

	var accepted []models.Fix
	for _, fix := range fixes {
		if fix.Status == "accepted" {
			accepted = append(accepted, fix)
		}
	}

	if len(accepted) == 0 {
		return "", fmt.Errorf("no accepted fixes to apply")
	}

	// Create branch
	branchName := fmt.Sprintf("fix/securescan-%s", time.Now().Format("2006-01-02"))
	if err := gitCmd(ctx, repoPath, "checkout", "-b", branchName); err != nil {
		return "", fmt.Errorf("create branch: %w", err)
	}

	// Sort fixes by file, then by line number DESCENDING so earlier fixes don't
	// shift line numbers for later ones in the same file.
	sort.Slice(accepted, func(i, j int) bool {
		if accepted[i].FilePath != accepted[j].FilePath {
			return accepted[i].FilePath < accepted[j].FilePath
		}
		li, lj := 0, 0
		if accepted[i].LineStart != nil {
			li = *accepted[i].LineStart
		}
		if accepted[j].LineStart != nil {
			lj = *accepted[j].LineStart
		}
		return li > lj // descending
	})

	// Apply each fix
	applied := 0
	for _, fix := range accepted {
		if fix.FixedCode == nil || fix.OriginalCode == nil {
			continue
		}

		filePath := fix.FilePath
		if !strings.HasPrefix(filePath, "/") {
			filePath = repoPath + "/" + filePath
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("skip fix %s: can't read %s: %v", fix.ID, filePath, err)
			continue
		}

		content := string(data)
		original := *fix.OriginalCode
		fixed := *fix.FixedCode

		// Simple string replacement — works for exact matches
		if strings.Contains(content, original) {
			content = strings.Replace(content, original, fixed, 1)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				log.Printf("skip fix %s: can't write %s: %v", fix.ID, filePath, err)
				continue
			}
			applied++
		}
	}

	if applied == 0 {
		// Revert branch creation if nothing was applied
		gitCmd(ctx, repoPath, "checkout", "-")
		gitCmd(ctx, repoPath, "branch", "-D", branchName)
		return "", fmt.Errorf("no fixes could be applied (code may have changed)")
	}

	// Stage and commit
	if err := gitCmd(ctx, repoPath, "add", "-A"); err != nil {
		return "", fmt.Errorf("git add: %w", err)
	}

	commitMsg := fmt.Sprintf("fix: apply %d SecureScan security fixes\n\nFixes applied:\n", applied)
	for _, fix := range accepted {
		if fix.FixedCode != nil {
			commitMsg += fmt.Sprintf("- %s (%s)\n", fix.Description, fix.FilePath)
		}
	}

	if err := gitCmd(ctx, repoPath, "commit", "-m", commitMsg); err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}

	// Push (uses PAT via credential helper or URL embedding)
	if s.GithubToken != "" {
		if err := gitCmd(ctx, repoPath, "push", "-u", "origin", branchName); err != nil {
			return branchName, fmt.Errorf("git push: %w (branch created locally as %s)", err, branchName)
		}
	}

	// Update fix statuses to 'applied'
	for _, fix := range accepted {
		s.FixSvc.UpdateStatus(ctx, fix.ID, "applied")
	}

	return branchName, nil
}

func gitCmd(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
