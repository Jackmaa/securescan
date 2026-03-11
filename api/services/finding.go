package services

import (
	"context" // Passed through from handlers for DB operations and cancellation.
	"fmt"     // Build dynamic SQL and wrap errors with context.
	"strings" // Assemble SQL fragments safely from known filter/sort options.

	"securescan/models" // Finding persistence and API response shapes.

	"github.com/google/uuid"          // Scan IDs are UUIDs.
	"github.com/jackc/pgx/v5/pgxpool" // PostgreSQL connection pool.
)

// FindingFilter captures the server-side query options for listing findings.
//
// This structure exists so that:
// - the handler can convert query params into a typed object, and
// - the service can build a single SQL query from those options.
type FindingFilter struct {
	ScanID   uuid.UUID
	Severity string
	Owasp    string
	Tool     string
	Sort     string
	Page     int
	Limit    int
}

// FindingResult is the paginated response returned by List.
type FindingResult struct {
	Findings []models.Finding `json:"findings"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	Limit    int              `json:"limit"`
}

// FindingService provides read/write operations on findings.
//
// Findings are the normalized output of all scanners (Semgrep, TruffleHog, npm audit, etc.).
// This service focuses on queryability (filtering/sorting/pagination) and efficient inserts.
type FindingService struct {
	DB *pgxpool.Pool
}

// NewFindingService constructs the service.
func NewFindingService(db *pgxpool.Pool) *FindingService {
	return &FindingService{DB: db}
}

// List returns findings for a scan, with optional filters and pagination.
//
// Why we build SQL dynamically:
// - Filters are optional; assembling the WHERE clause avoids complicated “OR param is NULL” patterns.
// - We still use positional parameters for values, so user input is not interpolated into SQL.
//
// Sorting:
// - Severity sorting uses a CASE expression so the order matches human expectations.
func (s *FindingService) List(ctx context.Context, f FindingFilter) (*FindingResult, error) {
	// Build dynamic WHERE clause from filters
	where := []string{"scan_id = $1"}
	args := []any{f.ScanID}
	argIdx := 2

	if f.Severity != "" {
		where = append(where, fmt.Sprintf("severity = $%d", argIdx))
		args = append(args, f.Severity)
		argIdx++
	}
	if f.Owasp != "" {
		where = append(where, fmt.Sprintf("owasp_category = $%d", argIdx))
		args = append(args, f.Owasp)
		argIdx++
	}
	if f.Tool != "" {
		where = append(where, fmt.Sprintf("tool_name = $%d", argIdx))
		args = append(args, f.Tool)
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Severity sort uses a custom ordering so critical > high > medium > low > info
	orderBy := "created_at DESC"
	switch f.Sort {
	case "severity":
		orderBy = "CASE severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 ELSE 5 END"
	case "file":
		orderBy = "file_path ASC, line_start ASC"
	case "owasp":
		orderBy = "owasp_category ASC NULLS LAST"
	case "tool":
		orderBy = "tool_name ASC"
	}

	// Count total
	var total int
	err := s.DB.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM findings WHERE %s", whereClause),
		args...,
	).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Fetch page
	offset := (f.Page - 1) * f.Limit
	query := fmt.Sprintf(`
		SELECT id, scan_id, tool_name, rule_id, file_path, line_start, line_end,
		       col_start, col_end, message, severity, owasp_category, owasp_label,
		       cwe_id, raw_output, code_snippet, created_at
		FROM findings WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIdx, argIdx+1)
	args = append(args, f.Limit, offset)

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []models.Finding
	for rows.Next() {
		var f models.Finding
		err := rows.Scan(&f.ID, &f.ScanID, &f.ToolName, &f.RuleID, &f.FilePath,
			&f.LineStart, &f.LineEnd, &f.ColStart, &f.ColEnd, &f.Message,
			&f.Severity, &f.OwaspCategory, &f.OwaspLabel, &f.CweID,
			&f.RawOutput, &f.CodeSnippet, &f.CreatedAt)
		if err != nil {
			return nil, err
		}
		findings = append(findings, f)
	}

	return &FindingResult{
		Findings: findings,
		Total:    total,
		Page:     f.Page,
		Limit:    f.Limit,
	}, nil
}

// BulkInsert writes all findings from a tool run into the database.
//
// Why bulk insert in a transaction:
// - Tools can output many findings; we want all-or-nothing persistence per tool run.
// - A transaction reduces overhead and avoids partially persisted results when errors occur.
func (s *FindingService) BulkInsert(ctx context.Context, findings []models.Finding) error {
	if len(findings) == 0 {
		return nil
	}

	// Use a transaction for atomicity
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, f := range findings {
		_, err := tx.Exec(ctx, `
			INSERT INTO findings (id, scan_id, tool_name, rule_id, file_path, line_start, line_end,
			    col_start, col_end, message, severity, owasp_category, owasp_label, cwe_id,
			    raw_output, code_snippet)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		`, f.ID, f.ScanID, f.ToolName, f.RuleID, f.FilePath, f.LineStart, f.LineEnd,
			f.ColStart, f.ColEnd, f.Message, f.Severity, f.OwaspCategory, f.OwaspLabel,
			f.CweID, f.RawOutput, f.CodeSnippet)
		if err != nil {
			return fmt.Errorf("insert finding: %w", err)
		}
	}

	return tx.Commit(ctx)
}
