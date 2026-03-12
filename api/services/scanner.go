package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"securescan/models"
	"securescan/owasp"
	"securescan/scanner"

	"github.com/google/uuid"
)

// ToolResult carries a single tool's output through the fan-in channel.
type ToolResult struct {
	ToolName string
	Findings []models.Finding
	Err      error
}

// RunScanPipeline orchestrates the full scanning flow for a project.
//
// Execution model:
//   1. Determine applicable tools based on project languages.
//   2. Run each tool in its own goroutine (fan-out).
//   3. Collect results via a channel (fan-in). Tool failures are non-fatal.
//   4. Map all findings to OWASP 2025 categories.
//   5. Compute score/grade.
//   6. Persist everything to the database.
//
// SSE events are broadcast at each stage so the frontend can render live progress.
func RunScanPipeline(ctx context.Context, scanSvc *ScanService, findingSvc *FindingService,
	scan *models.Scan, project *models.Project) {

	scanID := scan.ID
	repoPath := project.LocalPath

	// --- Stage 1: Determine applicable tools ---
	var applicable []scanner.ToolAdapter
	for _, adapter := range scanner.Registry {
		if adapter.IsApplicable(project.Languages) {
			applicable = append(applicable, adapter)
		}
	}

	toolCount := len(applicable)
	scanSvc.DB.Exec(ctx, `UPDATE scans SET tool_count = $1, status = 'scanning' WHERE id = $2`,
		toolCount, scanID)
	scanSvc.Broadcast(scanID, SSEEvent{
		Type: "status",
		Data: fmt.Sprintf(`{"status":"scanning","message":"Running %d tools..."}`, toolCount),
	})

	// --- Stage 2: Fan-out tool execution ---
	results := make(chan ToolResult, toolCount)
	var wg sync.WaitGroup

	for i, adapter := range applicable {
		wg.Add(1)
		go func(idx int, a scanner.ToolAdapter) {
			defer wg.Done()

			toolName := a.Name()
			scanSvc.Broadcast(scanID, SSEEvent{
				Type: "tool_start",
				Data: fmt.Sprintf(`{"tool":"%s","index":%d,"total":%d}`, toolName, idx+1, toolCount),
			})

			// Run the tool with a generous timeout per tool
			toolCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			raw, err := a.Run(toolCtx, repoPath)
			if err != nil {
				log.Printf("tool %s failed: %v", toolName, err)
				scanSvc.Broadcast(scanID, SSEEvent{
					Type: "tool_error",
					Data: fmt.Sprintf(`{"tool":"%s","error":"%s"}`, toolName, escapeJSON(err.Error())),
				})
				results <- ToolResult{ToolName: toolName, Err: err}
				return
			}

			findings, err := a.Parse(scanID, raw)
			if err != nil {
				log.Printf("tool %s parse failed: %v", toolName, err)
				scanSvc.Broadcast(scanID, SSEEvent{
					Type: "tool_error",
					Data: fmt.Sprintf(`{"tool":"%s","error":"parse: %s"}`, toolName, escapeJSON(err.Error())),
				})
				results <- ToolResult{ToolName: toolName, Err: err}
				return
			}

			scanSvc.Broadcast(scanID, SSEEvent{
				Type: "tool_complete",
				Data: fmt.Sprintf(`{"tool":"%s","findings_count":%d}`, toolName, len(findings)),
			})
			results <- ToolResult{ToolName: toolName, Findings: findings}
		}(i, adapter)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// --- Stage 3: Fan-in results ---
	var allFindings []models.Finding
	toolsDone := 0
	for result := range results {
		toolsDone++
		scanSvc.DB.Exec(ctx, `UPDATE scans SET tools_done = $1 WHERE id = $2`, toolsDone, scanID)

		if result.Err == nil && len(result.Findings) > 0 {
			allFindings = append(allFindings, result.Findings...)
		}
	}

	// --- Stage 4: OWASP mapping ---
	scanSvc.updateStatus(ctx, scanID, "mapping")
	scanSvc.Broadcast(scanID, SSEEvent{
		Type: "mapping",
		Data: `{"message":"Mapping to OWASP Top 10:2025..."}`,
	})
	owasp.MapFindings(allFindings)

	// --- Stage 5: Scoring ---
	scanSvc.updateStatus(ctx, scanID, "scoring")
	scanSvc.Broadcast(scanID, SSEEvent{
		Type: "scoring",
		Data: `{"message":"Computing security score..."}`,
	})
	score, grade := ComputeScore(allFindings)

	// --- Stage 6: Persist ---
	if len(allFindings) > 0 {
		if err := findingSvc.BulkInsert(ctx, allFindings); err != nil {
			log.Printf("bulk insert findings failed: %v", err)
		}
	}

	now := time.Now()
	scanSvc.DB.Exec(ctx, `
		UPDATE scans SET status = 'completed', score = $1, grade = $2, completed_at = $3
		WHERE id = $4
	`, score, grade, now, scanID)

	scanSvc.Broadcast(scanID, SSEEvent{
		Type: "complete",
		Data: fmt.Sprintf(`{"scan_id":"%s","score":%d,"grade":"%s","findings":%d}`,
			scanID, score, grade, len(allFindings)),
	})

	// Close all subscriber channels so SSE connections terminate cleanly
	scanSvc.mu.Lock()
	for _, ch := range scanSvc.subscribers[scanID] {
		close(ch)
	}
	delete(scanSvc.subscribers, scanID)
	scanSvc.mu.Unlock()
}

// escapeJSON ensures strings embedded in hand-built JSON don't break the format.
func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	// Strip surrounding quotes since caller embeds this inside a quoted string
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}

// newUUID is a convenience wrapper for generating finding IDs in tests.
var _ = uuid.New
