package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"securescan/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SSEEvent represents a Server-Sent Event to push to connected clients.
type SSEEvent struct {
	Type string // event name: status, tool_start, tool_complete, etc.
	Data string // JSON payload
}

// ScanStats holds aggregated numbers for the dashboard.
type ScanStats struct {
	TotalFindings int            `json:"total_findings"`
	BySeverity    map[string]int `json:"by_severity"`
	ByOwasp       map[string]int `json:"by_owasp"`
	ByTool        map[string]int `json:"by_tool"`
	Score         *int           `json:"score"`
	Grade         *string        `json:"grade"`
}

type ScanService struct {
	DB *pgxpool.Pool

	// SSE subscriber management: each scan ID maps to a set of channels.
	// Multiple browser tabs can subscribe to the same scan.
	mu          sync.RWMutex
	subscribers map[uuid.UUID][]chan SSEEvent
}

func NewScanService(db *pgxpool.Pool) *ScanService {
	return &ScanService{
		DB:          db,
		subscribers: make(map[uuid.UUID][]chan SSEEvent),
	}
}

func (s *ScanService) CreateAndRun(ctx context.Context, project *models.Project) (*models.Scan, error) {
	scanID := uuid.New()
	now := time.Now()

	_, err := s.DB.Exec(ctx, `
		INSERT INTO scans (id, project_id, status, started_at)
		VALUES ($1, $2, 'pending', $3)
	`, scanID, project.ID, now)
	if err != nil {
		return nil, fmt.Errorf("insert scan: %w", err)
	}

	scan := &models.Scan{
		ID:        scanID,
		ProjectID: project.ID,
		Status:    "pending",
		StartedAt: &now,
	}

	// Kick off the scan pipeline in a goroutine so the HTTP response returns immediately.
	// The client tracks progress via SSE on GET /scans/:id/progress.
	go s.runPipeline(scan, project)

	return scan, nil
}

func (s *ScanService) GetByID(ctx context.Context, id uuid.UUID) (*models.Scan, error) {
	scan := &models.Scan{}
	err := s.DB.QueryRow(ctx, `
		SELECT id, project_id, status, score, grade, tool_count, tools_done,
		       error_msg, started_at, completed_at, created_at
		FROM scans WHERE id = $1
	`, id).Scan(&scan.ID, &scan.ProjectID, &scan.Status, &scan.Score, &scan.Grade,
		&scan.ToolCount, &scan.ToolsDone, &scan.ErrorMsg, &scan.StartedAt,
		&scan.CompletedAt, &scan.CreatedAt)
	if err != nil {
		return nil, err
	}
	return scan, nil
}

func (s *ScanService) GetStats(ctx context.Context, scanID uuid.UUID) (*ScanStats, error) {
	stats := &ScanStats{
		BySeverity: make(map[string]int),
		ByOwasp:    make(map[string]int),
		ByTool:     make(map[string]int),
	}

	// Get scan score/grade
	scan, err := s.GetByID(ctx, scanID)
	if err != nil {
		return nil, err
	}
	stats.Score = scan.Score
	stats.Grade = scan.Grade

	// Aggregate findings
	rows, err := s.DB.Query(ctx, `
		SELECT severity, owasp_category, tool_name, COUNT(*)
		FROM findings WHERE scan_id = $1
		GROUP BY GROUPING SETS ((severity), (owasp_category), (tool_name))
	`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var severity, owasp, tool *string
		var count int
		if err := rows.Scan(&severity, &owasp, &tool, &count); err != nil {
			return nil, err
		}
		if severity != nil {
			stats.BySeverity[*severity] = count
		}
		if owasp != nil {
			stats.ByOwasp[*owasp] = count
		}
		if tool != nil {
			stats.ByTool[*tool] = count
		}
	}

	for _, v := range stats.BySeverity {
		stats.TotalFindings += v
	}

	return stats, nil
}

// Subscribe returns a channel that will receive SSE events for the given scan.
func (s *ScanService) Subscribe(scanID uuid.UUID) chan SSEEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan SSEEvent, 64)
	s.subscribers[scanID] = append(s.subscribers[scanID], ch)
	return ch
}

// Unsubscribe removes a channel from the subscriber list and closes it.
func (s *ScanService) Unsubscribe(scanID uuid.UUID, ch chan SSEEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscribers[scanID]
	for i, sub := range subs {
		if sub == ch {
			s.subscribers[scanID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

// Broadcast sends an SSE event to all subscribers of a scan.
func (s *ScanService) Broadcast(scanID uuid.UUID, event SSEEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.subscribers[scanID] {
		select {
		case ch <- event:
		default:
			// Drop event if subscriber is too slow
		}
	}
}

// runPipeline is the main scan orchestration. Runs in its own goroutine.
// Placeholder — the real implementation comes in Phase 2 with tool adapters.
func (s *ScanService) runPipeline(scan *models.Scan, project *models.Project) {
	ctx := context.Background()

	s.updateStatus(ctx, scan.ID, "scanning")
	s.Broadcast(scan.ID, SSEEvent{Type: "status", Data: `{"status":"scanning","message":"Scan started..."}`})

	// TODO: Phase 2 — tool adapter orchestration goes here

	now := time.Now()
	s.DB.Exec(ctx, `
		UPDATE scans SET status = 'completed', completed_at = $1
		WHERE id = $2
	`, now, scan.ID)

	s.Broadcast(scan.ID, SSEEvent{Type: "complete", Data: fmt.Sprintf(`{"scan_id":"%s"}`, scan.ID)})

	// Close all subscriber channels after a brief delay so the complete event gets delivered
	s.mu.Lock()
	for _, ch := range s.subscribers[scan.ID] {
		close(ch)
	}
	delete(s.subscribers, scan.ID)
	s.mu.Unlock()
}

func (s *ScanService) updateStatus(ctx context.Context, scanID uuid.UUID, status string) {
	s.DB.Exec(ctx, `UPDATE scans SET status = $1 WHERE id = $2`, status, scanID)
}
