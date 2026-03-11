package services

import (
	"context"
	"fmt"

	"securescan/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FixService struct {
	DB *pgxpool.Pool
}

func NewFixService(db *pgxpool.Pool) *FixService {
	return &FixService{DB: db}
}

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

func (s *FixService) Insert(ctx context.Context, fix *models.Fix) error {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO fixes (id, finding_id, scan_id, fix_type, description, explanation,
		    original_code, fixed_code, file_path, line_start, line_end)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, fix.ID, fix.FindingID, fix.ScanID, fix.FixType, fix.Description, fix.Explanation,
		fix.OriginalCode, fix.FixedCode, fix.FilePath, fix.LineStart, fix.LineEnd)
	return err
}
