package main

import (
	"context"
	"fmt"

	"ContentBlueprint/internal/updater"
)

const updateProgressEventName = "app:update-progress"

type UpdateInfo = updater.Info
type UpdateProgress = updater.Progress

type updateCoordinator interface {
	Check(context.Context) (updater.Info, error)
	Download(context.Context, string, updater.ProgressFunc) (updater.Info, error)
	Launch(string) error
}

// CheckForUpdates performs a read-only check of the fixed public GitHub release.
func (app *App) CheckForUpdates() (UpdateInfo, error) {
	if app.updates == nil {
		return UpdateInfo{}, fmt.Errorf("application updater is unavailable")
	}
	return app.updates.Check(app.updateContext())
}

// DownloadUpdate downloads only the named latest version and emits bounded
// progress updates. The frontend cannot supply a URL or destination path.
func (app *App) DownloadUpdate(version string) (UpdateInfo, error) {
	if app.updates == nil {
		return UpdateInfo{}, fmt.Errorf("application updater is unavailable")
	}
	return app.updates.Download(app.updateContext(), version, func(progress updater.Progress) {
		if app.ctx != nil && app.emitEvent != nil {
			app.emitEvent(app.ctx, updateProgressEventName, progress)
		}
	})
}

// LaunchDownloadedUpdate starts only a backend-verified installer, then asks
// Wails to quit. A launch failure leaves the application running.
func (app *App) LaunchDownloadedUpdate(version string) error {
	if app.updates == nil {
		return fmt.Errorf("application updater is unavailable")
	}
	if app.ctx == nil || app.quitApplication == nil {
		return fmt.Errorf("application shutdown is unavailable")
	}
	if err := app.updates.Launch(version); err != nil {
		return err
	}
	app.quitApplication(app.ctx)
	return nil
}

func (app *App) updateContext() context.Context {
	if app.ctx != nil {
		return app.ctx
	}
	return context.Background()
}
