package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"securescan/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectService struct {
	DB        *pgxpool.Pool
	Workspace string
}

func NewProjectService(db *pgxpool.Pool, workspace string) *ProjectService {
	return &ProjectService{DB: db, Workspace: workspace}
}

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
func cloneRepo(ctx context.Context, url, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// detectStack inspects the cloned repo for language/framework indicators.
// Checks for well-known manifest files and counts file extensions as a fallback.
func detectStack(repoPath string) (languages, frameworks []string) {
	langSet := map[string]bool{}
	fwSet := map[string]bool{}

	// Manifest-based detection: each file strongly implies a language + framework
	manifests := map[string][2]string{
		"package.json":    {"JavaScript", ""},
		"tsconfig.json":   {"TypeScript", ""},
		"go.mod":          {"Go", ""},
		"requirements.txt": {"Python", ""},
		"Pipfile":         {"Python", ""},
		"composer.json":   {"PHP", ""},
		"Gemfile":         {"Ruby", ""},
		"Cargo.toml":      {"Rust", ""},
		"pom.xml":         {"Java", "Maven"},
		"build.gradle":    {"Java", "Gradle"},
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
