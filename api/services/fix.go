package services

import (
	"context" // Passed through from handlers for DB operations.
	"fmt"     // Used to signal not-found conditions cleanly.

	"securescan/models" // Fix persistence model.

	"github.com/google/uuid"          // Fix IDs and scan IDs are UUIDs.
	"github.com/jackc/pgx/v5/pgxpool" // PostgreSQL connection pool.
)

// FixService provides read/write operations on fixes.
//
// Fixes are remediation suggestions generated from findings. This service is focused on:
// - listing fixes for review in the UI
// - updating accept/reject status
// - inserting newly generated fixes
type FixService struct {
	DB *pgxpool.Pool
}

// NewFixService constructs the service.
func NewFixService(db *pgxpool.Pool) *FixService {
	return &FixService{DB: db}
}

// ListByScan returns all fixes for a scan, ordered by creation time.
//
// Ordering is stable so UIs can render consistent lists (and so bulk actions map
// naturally to the displayed order).
func (s *FixService) ListByScan(ctx context.Context, scanID uuid.UUID) ([]models.Fix, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, finding_id, scan_id, fix_type, description, explanation,
		       original_code, fixed_code, file_path, line_start, line_end, status, created_at
		FROM fixes WHERE scan_id = $1
		ORDER BY created_at ASC
	`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fixes []models.Fix
	for rows.Next() {
		var f models.Fix
		err := rows.Scan(&f.ID, &f.FindingID, &f.ScanID, &f.FixType, &f.Description,
			&f.Explanation, &f.OriginalCode, &f.FixedCode, &f.FilePath,
			&f.LineStart, &f.LineEnd, &f.Status, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		fixes = append(fixes, f)
	}

	return fixes, nil
}

// UpdateStatus changes a fix status (e.g., "accepted" / "rejected").
//
// We check RowsAffected to distinguish "not found" from "no-op update".
func (s *FixService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.DB.Exec(ctx, `UPDATE fixes SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("fix not found")
	}
	return nil
}

// Insert persists a new fix record.
//
// Fix creation is separated from scan execution so that:
// - different fix generators can be added later (LLM-based, rule-based, etc.)
// - fixes can be generated after-the-fact without re-running scanners
func (s *FixService) Insert(ctx context.Context, fix *models.Fix) error {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO fixes (id, finding_id, scan_id, fix_type, description, explanation,
		    original_code, fixed_code, file_path, line_start, line_end)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, fix.ID, fix.FindingID, fix.ScanID, fix.FixType, fix.Description, fix.Explanation,
		fix.OriginalCode, fix.FixedCode, fix.FilePath, fix.LineStart, fix.LineEnd)
	return err
}
