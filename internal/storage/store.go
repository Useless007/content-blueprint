package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"ContentBlueprint/internal/domain"
)

const (
	settingsFilename = "settings.json"
	projectsDirname  = "projects"
	maxJSONFileSize  = 32 << 20
)

var validProjectID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

type Store struct {
	root  string
	now   func() time.Time
	mutex sync.RWMutex
}

func New(root string) *Store {
	return &Store{root: filepath.Clean(root), now: time.Now}
}

func (store *Store) Root() string {
	return store.root
}

func (store *Store) LoadSettings() (domain.ProviderSettings, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	settings := domain.DefaultSettings()
	err := readJSON(filepath.Join(store.root, settingsFilename), &settings)
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return domain.ProviderSettings{}, fmt.Errorf("load provider settings: %w", err)
	}
	settings = domain.NormalizeSettings(settings)
	if err := domain.ValidateSettings(settings); err != nil {
		return domain.ProviderSettings{}, fmt.Errorf("saved provider settings are invalid: %w", err)
	}
	return settings, nil
}

func (store *Store) SaveSettings(settings domain.ProviderSettings) error {
	settings = domain.NormalizeSettings(settings)
	if err := domain.ValidateSettings(settings); err != nil {
		return err
	}

	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := writeJSONAtomic(filepath.Join(store.root, settingsFilename), settings); err != nil {
		return fmt.Errorf("save provider settings: %w", err)
	}
	return nil
}

func (store *Store) SaveProject(project domain.Project) (domain.Project, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if project.ID == "" {
		id, err := newProjectID()
		if err != nil {
			return domain.Project{}, fmt.Errorf("create project id: %w", err)
		}
		project.ID = id
	}
	if err := validateID(project.ID); err != nil {
		return domain.Project{}, err
	}
	project.Name = strings.TrimSpace(project.Name)
	if project.Name == "" {
		contentTitle := ""
		if project.Content != nil {
			contentTitle = project.Content.Title
		}
		project.Name = firstNonEmpty(project.Brief.Keyword, contentTitle, "Untitled project")
	}
	project.Settings = domain.NormalizeSettings(project.Settings)
	if err := domain.ValidateSettings(project.Settings); err != nil {
		return domain.Project{}, fmt.Errorf("invalid project settings: %w", err)
	}
	normalizeSlices(&project)

	now := store.now().UTC().Format(time.RFC3339Nano)
	if project.CreatedAt == "" {
		project.CreatedAt = now
	} else if _, err := time.Parse(time.RFC3339Nano, project.CreatedAt); err != nil {
		return domain.Project{}, fmt.Errorf("createdAt must be an RFC3339 timestamp")
	}
	project.UpdatedAt = now

	path := store.projectPath(project.ID)
	if err := writeJSONAtomic(path, project); err != nil {
		return domain.Project{}, fmt.Errorf("save project %q: %w", project.ID, err)
	}
	return project, nil
}

func (store *Store) LoadProject(id string) (domain.Project, error) {
	if err := validateID(id); err != nil {
		return domain.Project{}, err
	}
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	var project domain.Project
	if err := readJSON(store.projectPath(id), &project); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.Project{}, fmt.Errorf("project %q was not found", id)
		}
		return domain.Project{}, fmt.Errorf("load project %q: %w", id, err)
	}
	if project.ID != id {
		return domain.Project{}, fmt.Errorf("project file %q contains mismatched id %q", id, project.ID)
	}
	normalizeSlices(&project)
	project.Settings = domain.NormalizeSettings(project.Settings)
	return project, nil
}

func (store *Store) ListProjects() ([]domain.ProjectSummary, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	directory := filepath.Join(store.root, projectsDirname)
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return []domain.ProjectSummary{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list project directory: %w", err)
	}

	summaries := make([]domain.ProjectSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		fileID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if validateID(fileID) != nil {
			continue
		}
		var project domain.Project
		path := filepath.Join(directory, entry.Name())
		if err := readJSON(path, &project); err != nil {
			continue
		}
		if validateID(project.ID) != nil || project.ID != fileID {
			continue
		}
		score := 0
		if project.Quality != nil {
			score = project.Quality.Score
		}
		summaries = append(summaries, domain.ProjectSummary{
			ID:        project.ID,
			Name:      project.Name,
			Keyword:   project.Brief.Keyword,
			Score:     score,
			UpdatedAt: project.UpdatedAt,
		})
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		left, leftErr := time.Parse(time.RFC3339Nano, summaries[i].UpdatedAt)
		right, rightErr := time.Parse(time.RFC3339Nano, summaries[j].UpdatedAt)
		if leftErr == nil && rightErr == nil && !left.Equal(right) {
			return left.After(right)
		}
		if summaries[i].UpdatedAt != summaries[j].UpdatedAt {
			return summaries[i].UpdatedAt > summaries[j].UpdatedAt
		}
		return summaries[i].ID < summaries[j].ID
	})
	return summaries, nil
}

func (store *Store) DeleteProject(id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()

	err := os.Remove(store.projectPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("project %q was not found", id)
	}
	if err != nil {
		return fmt.Errorf("delete project %q: %w", id, err)
	}
	return nil
}

func (store *Store) projectPath(id string) string {
	return filepath.Join(store.root, projectsDirname, id+".json")
}

func validateID(id string) error {
	if !validProjectID.MatchString(id) {
		return fmt.Errorf("invalid project id %q", id)
	}
	return nil
}

func newProjectID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func normalizeSlices(project *domain.Project) {
	if project.Brief.Evidence == nil {
		project.Brief.Evidence = []domain.EvidenceSource{}
	}
	project.GroundingSources = domain.NormalizeGroundingSources(project.GroundingSources)
	if project.Content != nil {
		if project.Content.KeyTakeaways == nil {
			project.Content.KeyTakeaways = []domain.KeyTakeaway{}
		}
		if project.Content.FAQData == nil {
			project.Content.FAQData = []domain.FAQItem{}
		}
	}
	if project.Quality != nil && project.Quality.Checks == nil {
		project.Quality.Checks = []domain.QualityCheck{}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func readJSON(path string, target any) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > maxJSONFileSize {
		return fmt.Errorf("file exceeds %d MiB limit", maxJSONFileSize>>20)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	data = append(data, '\n')
	if len(data) > maxJSONFileSize {
		return fmt.Errorf("encoded JSON exceeds %d MiB limit", maxJSONFileSize>>20)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".content-blueprint-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return nil
}
