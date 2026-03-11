package services

import (
	"context"       // Passed down from handlers; used for git clone cancellation/timeouts.
	"fmt"           // Wrap errors with context for easier debugging.
	"os"            // Workspace creation/cleanup and filesystem probes for stack detection.
	"os/exec"       // Execute external tools (git clone).
	"path/filepath" // OS-correct workspace and repo path assembly.
	"strings"       // Simple content/filename heuristics for stack detection.

	"securescan/models" // Project models and create request payload.

	"github.com/google/uuid"          // Project IDs are UUIDs and define workspace directory names.
	"github.com/jackc/pgx/v5/pgxpool" // PostgreSQL connection pool.
)

// ProjectService owns project creation and retrieval.
//
// Projects are the unit of scanning: a project records where code came from (git/zip),
// where it is staged locally, and a detected language/framework “stack” used to decide
// which scanning tools are applicable.
type ProjectService struct {
	DB        *pgxpool.Pool
	Workspace string
}

// NewProjectService constructs a service with DB and workspace root.
//
// Workspace is injected so the server can control where repos are staged (and so
// tests can use temporary directories).
func NewProjectService(db *pgxpool.Pool, workspace string) *ProjectService {
	return &ProjectService{DB: db, Workspace: workspace}
}

// Create persists a new project and prepares its local workspace.
//
// High-level flow:
// - Create a unique workspace directory (named by UUID).
// - If source is `git`, clone into that directory.
// - Detect languages/frameworks to inform scan tool selection.
// - Insert the project record into the database.
//
// Cleanup strategy:
// - If cloning or DB insert fails, we remove the created workspace to avoid orphaned data.
func (s *ProjectService) Create(ctx context.Context, req models.CreateProjectRequest) (*models.Project, error) {
	projectID := uuid.New()
	localPath := filepath.Join(s.Workspace, projectID.String())

	if err := os.MkdirAll(localPath, 0755); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	if req.SourceType == "git" {
		if err := cloneRepo(ctx, req.SourceURL, localPath); err != nil {
			os.RemoveAll(localPath)
			return nil, fmt.Errorf("clone repo: %w", err)
		}
	}

	languages, frameworks := detectStack(localPath)
	// Postgres columns `languages` and `frameworks` are NOT NULL (with defaults).
	// A nil slice can be encoded as SQL NULL by drivers, so normalize to empty slices.
	if languages == nil {
		languages = []string{}
	}
	if frameworks == nil {
		frameworks = []string{}
	}

	project := &models.Project{
		ID:         projectID,
		Name:       req.Name,
		SourceType: req.SourceType,
		SourceURL:  req.SourceURL,
		LocalPath:  localPath,
		Languages:  languages,
		Frameworks: frameworks,
	}

	_, err := s.DB.Exec(ctx, `
		INSERT INTO projects (id, name, source_type, source_url, local_path, languages, frameworks)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, project.ID, project.Name, project.SourceType, project.SourceURL,
		project.LocalPath, project.Languages, project.Frameworks)
	if err != nil {
		os.RemoveAll(localPath)
		return nil, fmt.Errorf("insert project: %w", err)
	}

	return project, nil
}

// GetByID fetches a project record from the database.
//
// The returned LocalPath points to the on-disk workspace created during project creation.
func (s *ProjectService) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	p := &models.Project{}
	err := s.DB.QueryRow(ctx, `
		SELECT id, name, source_type, source_url, local_path, languages, frameworks, created_at
		FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.SourceType, &p.SourceURL, &p.LocalPath,
		&p.Languages, &p.Frameworks, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// cloneRepo does a shallow clone (depth=1) to save time and disk.
//
// Shallow clones are usually enough for SAST/secret scans because they analyze the
// current tree, not history. If future tools need git history, this can be revisited.
func cloneRepo(ctx context.Context, url, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// detectStack inspects the cloned repo for language/framework indicators.
// Checks for well-known manifest files and counts file extensions as a fallback.
//
// This is intentionally heuristic:
// - We want a fast “good enough” signal to decide which scanners to run.
// - We avoid heavyweight parsing of build files (package managers differ widely).
// If accuracy becomes important, this can be improved by integrating dedicated detectors.
func detectStack(repoPath string) (languages, frameworks []string) {
	langSet := map[string]bool{}
	fwSet := map[string]bool{}

	// Manifest-based detection: each file strongly implies a language + framework
	manifests := map[string][2]string{
		"package.json":     {"JavaScript", ""},
		"tsconfig.json":    {"TypeScript", ""},
		"go.mod":           {"Go", ""},
		"requirements.txt": {"Python", ""},
		"Pipfile":          {"Python", ""},
		"composer.json":    {"PHP", ""},
		"Gemfile":          {"Ruby", ""},
		"Cargo.toml":       {"Rust", ""},
		"pom.xml":          {"Java", "Maven"},
		"build.gradle":     {"Java", "Gradle"},
	}

	for file, pair := range manifests {
		if _, err := os.Stat(filepath.Join(repoPath, file)); err == nil {
			langSet[pair[0]] = true
			if pair[1] != "" {
				fwSet[pair[1]] = true
			}
		}
	}

	// Check package.json for known frameworks
	if data, err := os.ReadFile(filepath.Join(repoPath, "package.json")); err == nil {
		content := string(data)
		frameworkHints := map[string]string{
			"react":   "React",
			"vue":     "Vue",
			"svelte":  "Svelte",
			"angular": "Angular",
			"express": "Express",
			"next":    "Next.js",
			"nuxt":    "Nuxt",
		}
		for keyword, name := range frameworkHints {
			if strings.Contains(content, `"`+keyword+`"`) {
				fwSet[name] = true
			}
		}
	}

	// Extension-based fallback for languages not covered by manifests
	extMap := map[string]string{
		".py":   "Python",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".go":   "Go",
		".java": "Java",
		".rb":   "Ruby",
		".php":  "PHP",
		".rs":   "Rust",
		".cs":   "C#",
		".cpp":  "C++",
		".c":    "C",
	}

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			// Skip hidden directories like .git
			if info != nil && info.IsDir() && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if lang, ok := extMap[ext]; ok {
			langSet[lang] = true
		}
		return nil
	})

	for l := range langSet {
		languages = append(languages, l)
	}
	for f := range fwSet {
		frameworks = append(frameworks, f)
	}
	return languages, frameworks
}
