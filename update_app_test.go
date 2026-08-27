package main

import (
	"context"
	"errors"
	"testing"

	"ContentBlueprint/internal/updater"
)

type fakeUpdateCoordinator struct {
	checkInfo       updater.Info
	downloadInfo    updater.Info
	downloadVersion string
	launchVersion   string
	launchErr       error
}

func (fake *fakeUpdateCoordinator) Check(context.Context) (updater.Info, error) {
	return fake.checkInfo, nil
}

func (fake *fakeUpdateCoordinator) Download(_ context.Context, version string, progress updater.ProgressFunc) (updater.Info, error) {
	fake.downloadVersion = version
	progress(updater.Progress{Version: version, DownloadedBytes: 10, TotalBytes: 10, Percent: 100})
	return fake.downloadInfo, nil
}

func (fake *fakeUpdateCoordinator) Launch(version string) error {
	fake.launchVersion = version
	return fake.launchErr
}

func TestUpdateFacadeExposesOnlyVersionBasedMutations(t *testing.T) {
	fake := &fakeUpdateCoordinator{
		checkInfo:    updater.Info{CurrentVersion: "0.3.0", LatestVersion: "0.3.1", State: updater.StateUpdateAvailable},
		downloadInfo: updater.Info{CurrentVersion: "0.3.0", LatestVersion: "0.3.1", State: updater.StateReady},
	}
	quitCalled := false
	eventName := ""
	var eventPayload interface{}
	app := &App{
		ctx:             context.Background(),
		updates:         fake,
		quitApplication: func(context.Context) { quitCalled = true },
		emitEvent: func(_ context.Context, name string, payload ...interface{}) {
			eventName = name
			if len(payload) == 1 {
				eventPayload = payload[0]
			}
		},
	}

	checked, err := app.CheckForUpdates()
	if err != nil || checked.State != updater.StateUpdateAvailable {
		t.Fatalf("CheckForUpdates() = %+v, %v", checked, err)
	}
	downloaded, err := app.DownloadUpdate("0.3.1")
	if err != nil || downloaded.State != updater.StateReady || fake.downloadVersion != "0.3.1" {
		t.Fatalf("DownloadUpdate() = %+v, %v; version = %q", downloaded, err, fake.downloadVersion)
	}
	progress, ok := eventPayload.(updater.Progress)
	if eventName != updateProgressEventName || !ok || progress.Percent != 100 {
		t.Fatalf("progress event = %q, %#v", eventName, eventPayload)
	}
	if err := app.LaunchDownloadedUpdate("0.3.1"); err != nil || fake.launchVersion != "0.3.1" || !quitCalled {
		t.Fatalf("LaunchDownloadedUpdate() error = %v; version = %q; quit = %t", err, fake.launchVersion, quitCalled)
	}
}

func TestUpdateFacadeKeepsAppOpenWhenLaunchFails(t *testing.T) {
	launchErr := errors.New("launch failed")
	fake := &fakeUpdateCoordinator{launchErr: launchErr}
	quitCalled := false
	app := &App{
		ctx:             context.Background(),
		updates:         fake,
		quitApplication: func(context.Context) { quitCalled = true },
	}
	if err := app.LaunchDownloadedUpdate("0.3.1"); !errors.Is(err, launchErr) {
		t.Fatalf("LaunchDownloadedUpdate() error = %v, want %v", err, launchErr)
	}
	if quitCalled {
		t.Fatal("app quit after installer launch failure")
	}
}

func TestUpdateFacadeRejectsUnavailableCoordinator(t *testing.T) {
	app := &App{}
	if _, err := app.CheckForUpdates(); err == nil {
		t.Fatal("CheckForUpdates() unexpectedly succeeded")
	}
	if _, err := app.DownloadUpdate("0.3.1"); err == nil {
		t.Fatal("DownloadUpdate() unexpectedly succeeded")
	}
	if err := app.LaunchDownloadedUpdate("0.3.1"); err == nil {
		t.Fatal("LaunchDownloadedUpdate() unexpectedly succeeded")
	}
}

var _ updateCoordinator = (*fakeUpdateCoordinator)(nil)
