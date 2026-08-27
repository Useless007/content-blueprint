package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ContentBlueprint/internal/domain"
)

func TestStoreProjectLifecycle(t *testing.T) {
	store := New(t.TempDir())
	project := domain.Project{
		Name: "My project",
		Brief: domain.ContentBrief{
			Keyword: "evidence SEO",
		},
		Quality:  &domain.QualityReport{Score: 82},
		Settings: domain.DefaultSettings(),
		GroundingSources: []domain.GroundingSource{
			{URL: "https://EXAMPLE.com:443/research", Title: ""},
			{URL: "https://example.com/research", Title: "Grounded research"},
			{URL: "javascript:alert(1)", Title: "Invalid"},
		},
	}

	saved, err := store.SaveProject(project)
	if err != nil {
		t.Fatalf("SaveProject() error = %v", err)
	}
	if saved.ID == "" || saved.CreatedAt == "" || saved.UpdatedAt == "" {
		t.Fatalf("SaveProject() did not populate identity/timestamps: %#v", saved)
	}
	if len(saved.GroundingSources) != 1 || saved.GroundingSources[0].URL != "https://example.com/research" || saved.GroundingSources[0].Title != "Grounded research" {
		t.Errorf("saved grounding sources = %#v", saved.GroundingSources)
	}
	if _, err := time.Parse(time.RFC3339Nano, saved.UpdatedAt); err != nil {
		t.Fatalf("updatedAt is not RFC3339: %v", err)
	}

	loaded, err := store.LoadProject(saved.ID)
	if err != nil {
		t.Fatalf("LoadProject() error = %v", err)
	}
	if loaded.Name != saved.Name || loaded.Brief.Keyword != "evidence SEO" {
		t.Errorf("LoadProject() = %#v, want saved project", loaded)
	}
	if loaded.Brief.Evidence == nil || loaded.GroundingSources == nil || loaded.Content != nil || loaded.Quality == nil || loaded.Quality.Checks == nil {
		t.Errorf("LoadProject() should normalize slices for the frontend: %#v", loaded)
	}

	loaded.Name = "Updated name"
	updated, err := store.SaveProject(loaded)
	if err != nil {
		t.Fatalf("SaveProject(update) error = %v", err)
	}
	if updated.CreatedAt != saved.CreatedAt || updated.Name != "Updated name" {
		t.Errorf("updated project = %#v", updated)
	}

	summaries, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != saved.ID || summaries[0].Score != 82 {
		t.Errorf("ListProjects() = %#v", summaries)
	}

	if err := store.DeleteProject(saved.ID); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
	if _, err := store.LoadProject(saved.ID); err == nil {
		t.Fatal("LoadProject() after delete returned nil error")
	}
}

func TestStoreSettingsDefaultsAndPersistence(t *testing.T) {
	root := t.TempDir()
	store := New(root)
	settings, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Model != domain.DefaultModel || !settings.UseGrounding {
		t.Errorf("default settings = %#v", settings)
	}

	settings.UseGrounding = false
	settings.Model = "gemini-test-model"
	if err := store.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.UseGrounding || reloaded.Model != "gemini-test-model" {
		t.Errorf("reloaded settings = %#v", reloaded)
	}

	data, err := os.ReadFile(filepath.Join(root, settingsFilename))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(data)), "apikey") {
		t.Errorf("settings file unexpectedly contains an API-key field: %s", data)
	}
}

func TestStoreRejectsTraversalIDs(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.LoadProject("../settings"); err == nil {
		t.Fatal("LoadProject() accepted a traversal id")
	}
	if err := store.DeleteProject("..\\settings"); err == nil {
		t.Fatal("DeleteProject() accepted a traversal id")
	}
}

func TestListProjectsSkipsBrokenEntriesAndKeepsHealthyProjects(t *testing.T) {
	root := t.TempDir()
	store := New(root)
	healthy, err := store.SaveProject(domain.Project{
		ID: "healthy", Name: "Healthy", Brief: domain.ContentBrief{Keyword: "good"}, Settings: domain.DefaultSettings(),
	})
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, projectsDirname)
	if err := os.WriteFile(filepath.Join(directory, "broken.json"), []byte(`{"id":`), 0o600); err != nil {
		t.Fatal(err)
	}
	oversizedPath := filepath.Join(directory, "oversized.json")
	oversized, err := os.Create(oversizedPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := oversized.Truncate(maxJSONFileSize + 1); err != nil {
		oversized.Close()
		t.Fatal(err)
	}
	if err := oversized.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(filepath.Join(directory, "alias.json"), domain.Project{ID: "different"}); err != nil {
		t.Fatal(err)
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].ID != healthy.ID {
		t.Errorf("ListProjects() = %#v, want only healthy project", projects)
	}
	for _, test := range []struct {
		id   string
		want string
	}{{"broken", "decode JSON"}, {"oversized", "exceeds"}, {"alias", "mismatched id"}} {
		if _, err := store.LoadProject(test.id); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("LoadProject(%q) error = %v, want substring %q", test.id, err, test.want)
		}
	}
}

func TestSaveProjectRejectsOversizedJSONAndPreservesExistingFile(t *testing.T) {
	root := t.TempDir()
	store := New(root)
	content := &domain.GeneratedContent{Title: "Original", MainContentHTML: "small"}
	saved, err := store.SaveProject(domain.Project{
		ID: "size-limit", Name: "Original", Content: content, Settings: domain.DefaultSettings(),
	})
	if err != nil {
		t.Fatal(err)
	}
	oversizedContent := *saved.Content
	oversizedContent.Title = "Must not replace original"
	oversizedContent.MainContentHTML = strings.Repeat("x", maxJSONFileSize)
	saved.Content = &oversizedContent
	if _, err := store.SaveProject(saved); err == nil || !strings.Contains(err.Error(), "encoded JSON exceeds") {
		t.Fatalf("SaveProject(oversized) error = %v", err)
	}

	loaded, err := store.LoadProject("size-limit")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Content == nil || loaded.Content.Title != "Original" || loaded.Content.MainContentHTML != "small" {
		t.Errorf("oversized save changed existing project: %#v", loaded.Content)
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(root, projectsDirname, ".content-blueprint-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporaryFiles) != 0 {
		t.Errorf("temporary files were not cleaned up: %#v", temporaryFiles)
	}
}
