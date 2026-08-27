package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/exporter"
	"ContentBlueprint/internal/gemini"
	"ContentBlueprint/internal/prompt"
	"ContentBlueprint/internal/quality"
	"ContentBlueprint/internal/storage"
	"ContentBlueprint/internal/updater"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Type aliases keep the Wails contract concise while allowing the backend
// services to share one source of truth for JSON field names.
type EvidenceSource = domain.EvidenceSource
type GroundingSource = domain.GroundingSource
type ContentBrief = domain.ContentBrief
type ProviderSettings = domain.ProviderSettings
type GenerationRequest = domain.GenerationRequest
type GeneratedContent = domain.GeneratedContent
type KeyTakeaway = domain.KeyTakeaway
type FAQItem = domain.FAQItem
type QualityCheck = domain.QualityCheck
type QualityReport = domain.QualityReport
type PromptPreview = domain.PromptPreview
type Usage = domain.Usage
type GenerationResult = domain.GenerationResult
type Project = domain.Project
type ProjectSummary = domain.ProjectSummary
type BootstrapData = domain.BootstrapData

type saveDialogFunc func(context.Context, wailsruntime.SaveDialogOptions) (string, error)
type quitApplicationFunc func(context.Context)
type emitEventFunc func(context.Context, string, ...interface{})

type App struct {
	ctx             context.Context
	store           *storage.Store
	facebook        *facebookCoordinator
	growth          *growthCoordinator
	updates         updateCoordinator
	httpClient      *http.Client
	initErr         error
	saveFileDialog  saveDialogFunc
	quitApplication quitApplicationFunc
	emitEvent       emitEventFunc
}

func NewApp() *App {
	dataDirectory, err := applicationDataDirectory()
	facebook, facebookErr := newFacebookCoordinator()
	if err == nil && facebookErr != nil {
		err = facebookErr
	}
	growth, growthErr := newGrowthCoordinator()
	if err == nil && growthErr != nil {
		err = growthErr
	}
	app := &App{
		httpClient:      &http.Client{Timeout: 3 * time.Minute},
		facebook:        facebook,
		growth:          growth,
		updates:         updater.New(),
		initErr:         err,
		saveFileDialog:  wailsruntime.SaveFileDialog,
		quitApplication: wailsruntime.Quit,
		emitEvent:       wailsruntime.EventsEmit,
	}
	if err == nil {
		app.store = storage.New(dataDirectory)
	}
	return app
}

func applicationDataDirectory() (string, error) {
	if override := strings.TrimSpace(os.Getenv("CONTENT_BLUEPRINT_DATA_DIR")); override != "" {
		absolute, err := filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve CONTENT_BLUEPRINT_DATA_DIR: %w", err)
		}
		return absolute, nil
	}
	configurationDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user configuration directory: %w", err)
	}
	return filepath.Join(configurationDirectory, "ContentBlueprint"), nil
}

func (app *App) startup(ctx context.Context) {
	app.ctx = ctx
}

func (app *App) Bootstrap() (BootstrapData, error) {
	if err := app.ready(); err != nil {
		return BootstrapData{}, err
	}
	settings, err := app.store.LoadSettings()
	if err != nil {
		return BootstrapData{}, err
	}
	projects, err := app.store.ListProjects()
	if err != nil {
		return BootstrapData{}, err
	}
	return BootstrapData{
		Settings:              settings,
		Projects:              projects,
		APIKeyFromEnvironment: strings.TrimSpace(os.Getenv("GEMINI_API_KEY")) != "",
	}, nil
}

func (app *App) BuildPrompt(brief ContentBrief) (PromptPreview, error) {
	return prompt.Build(brief)
}

func (app *App) GenerateContent(request GenerationRequest) (GenerationResult, error) {
	if err := app.ready(); err != nil {
		return GenerationResult{}, err
	}
	preview, err := prompt.Build(request.Brief)
	if err != nil {
		return GenerationResult{}, err
	}
	settings := domain.NormalizeSettings(request.Settings)
	if err := domain.ValidateSettings(settings); err != nil {
		return GenerationResult{}, err
	}
	apiKey := strings.TrimSpace(request.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	}
	if apiKey == "" {
		return GenerationResult{}, fmt.Errorf("Gemini API key is required; enter a session-only key or set GEMINI_API_KEY")
	}

	ctx := app.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	client := gemini.New(app.httpClient, settings.BaseURL)
	content, groundingSources, usage, modelName, err := client.Generate(ctx, apiKey, settings, preview)
	if err != nil {
		return GenerationResult{}, err
	}
	return GenerationResult{
		Content:          content,
		GroundingSources: groundingSources,
		Quality:          quality.Evaluate(request.Brief, content),
		Prompt:           preview,
		Usage:            usage,
		Model:            modelName,
	}, nil
}

func (app *App) EvaluateContent(brief ContentBrief, content GeneratedContent) QualityReport {
	return quality.Evaluate(brief, content)
}

func (app *App) SaveProject(project Project) (Project, error) {
	if err := app.ready(); err != nil {
		return Project{}, err
	}
	if project.Content == nil {
		project.Quality = nil
	} else {
		report := quality.Evaluate(project.Brief, *project.Content)
		project.Quality = &report
	}
	return app.store.SaveProject(project)
}

func (app *App) LoadProject(id string) (Project, error) {
	if err := app.ready(); err != nil {
		return Project{}, err
	}
	return app.store.LoadProject(id)
}

func (app *App) ListProjects() ([]ProjectSummary, error) {
	if err := app.ready(); err != nil {
		return nil, err
	}
	return app.store.ListProjects()
}

func (app *App) DeleteProject(id string) error {
	if err := app.ready(); err != nil {
		return err
	}
	return app.store.DeleteProject(id)
}

func (app *App) SaveSettings(settings ProviderSettings) error {
	if err := app.ready(); err != nil {
		return err
	}
	return app.store.SaveSettings(settings)
}

func (app *App) ExportProject(project Project, format string) (string, error) {
	if err := app.ready(); err != nil {
		return "", err
	}
	data, extension, err := exporter.Render(project, format)
	if err != nil {
		return "", err
	}
	if app.ctx == nil {
		return "", fmt.Errorf("export dialog is unavailable before the application starts")
	}
	if app.saveFileDialog == nil {
		return "", fmt.Errorf("export dialog is unavailable")
	}
	path, err := app.saveFileDialog(app.ctx, wailsruntime.SaveDialogOptions{
		Title:                "Export content",
		DefaultFilename:      exporter.DefaultFilename(project, format),
		Filters:              []wailsruntime.FileFilter{exportFilter(format)},
		CanCreateDirectories: true,
	})
	if err != nil {
		return "", fmt.Errorf("open export dialog: %w", err)
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	path = exporter.EnsureExtension(path, extension)
	if err := exporter.Write(path, data); err != nil {
		return "", err
	}
	return path, nil
}

func (app *App) ready() error {
	if app.initErr != nil {
		return app.initErr
	}
	if app.store == nil {
		return fmt.Errorf("local project storage is unavailable")
	}
	return nil
}

func exportFilter(format string) wailsruntime.FileFilter {
	switch exporter.NormalizeFormat(format) {
	case "html":
		return wailsruntime.FileFilter{DisplayName: "HTML document (*.html)", Pattern: "*.html"}
	case "markdown":
		return wailsruntime.FileFilter{DisplayName: "Markdown document (*.md)", Pattern: "*.md"}
	default:
		return wailsruntime.FileFilter{DisplayName: "JSON document (*.json)", Pattern: "*.json"}
	}
}
