package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/storage"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func TestBootstrapUsesLocalDefaultsAndReportsEnvironmentKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "environment-only-secret")
	app := &App{
		store:      storage.New(t.TempDir()),
		httpClient: http.DefaultClient,
	}
	data, err := app.Bootstrap()
	if err != nil {
		t.Fatal(err)
	}
	if data.Settings.Model != domain.DefaultModel || !data.Settings.UseGrounding {
		t.Errorf("Bootstrap() settings = %#v", data.Settings)
	}
	if !data.APIKeyFromEnvironment {
		t.Error("Bootstrap() did not report GEMINI_API_KEY availability")
	}
	if data.Projects == nil || len(data.Projects) != 0 {
		t.Errorf("Bootstrap() projects = %#v, want non-nil empty slice", data.Projects)
	}
}

func TestExportProjectUsesSaveDialog(t *testing.T) {
	directory := t.TempDir()
	selected := filepath.Join(directory, "article")
	dialogCalled := false
	app := &App{
		ctx:        context.Background(),
		store:      storage.New(t.TempDir()),
		httpClient: http.DefaultClient,
		saveFileDialog: func(_ context.Context, options wailsruntime.SaveDialogOptions) (string, error) {
			dialogCalled = true
			if options.DefaultFilename != "my-article.md" || len(options.Filters) != 1 {
				t.Errorf("dialog options = %#v", options)
			}
			return selected, nil
		},
	}
	project := Project{
		Name: "My article",
		Content: &GeneratedContent{
			Title: "My article", Slug: "my-article", MainContentHTML: "<h2>Body</h2><p>Text.</p>",
		},
	}
	path, err := app.ExportProject(project, "markdown")
	if err != nil {
		t.Fatal(err)
	}
	if !dialogCalled || path != selected+".md" {
		t.Errorf("ExportProject() path = %q, dialogCalled = %v", path, dialogCalled)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("ExportProject() wrote an empty file")
	}
}
