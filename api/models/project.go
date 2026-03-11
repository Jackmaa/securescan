package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	SourceType string    `json:"source_type"`
	SourceURL  string    `json:"source_url,omitempty"`
	LocalPath  string    `json:"local_path"`
	Languages  []string  `json:"languages"`
	Frameworks []string  `json:"frameworks"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateProjectRequest struct {
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	SourceURL  string `json:"source_url,omitempty"`
}
