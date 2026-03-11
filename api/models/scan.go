package models

import (
	"time" // Scan timing fields are represented as time.Time pointers for optionality.

	"github.com/google/uuid" // Scan and project IDs are UUIDs in API and DB.
)

// Scan represents one execution of the scanning pipeline for a project.
//
// Fields like Score/Grade/ErrorMsg are pointers to distinguish "not computed yet"
// from a legitimate zero value.
type Scan struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Status      string     `json:"status"`
	Score       *int       `json:"score,omitempty"`
	Grade       *string    `json:"grade,omitempty"`
	ToolCount   int        `json:"tool_count"`
	ToolsDone   int        `json:"tools_done"`
	ErrorMsg    *string    `json:"error_msg,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
