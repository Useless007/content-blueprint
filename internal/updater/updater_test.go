package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testAPIBaseURL = "https://api.test.invalid/repos/Useless007/content-blueprint"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestCheckReportsUpToDateWithoutWritingFiles(t *testing.T) {
	temporaryRoot := t.TempDir()
	var requests atomic.Int32
	manager := testManager(t, temporaryRoot, func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		if request.URL.String() != testAPIBaseURL+"/releases/latest" {
			t.Fatalf("unexpected URL: %s", request.URL)
		}
		if request.Header.Get("X-GitHub-Api-Version") != githubAPIVersion {
			t.Fatalf("GitHub API version header = %q", request.Header.Get("X-GitHub-Api-Version"))
		}
		return jsonResponse(t, request, releaseJSON(t, "v0.3.0", true, true)), nil
	}, func(string) error {
		t.Fatal("check must not launch an installer")
		return nil
	})

	info, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if info.State != StateUpToDate || info.CurrentVersion != "0.3.0" || info.LatestVersion != "0.3.0" {
		t.Fatalf("unexpected info: %+v", info)
	}
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
	entries, err := os.ReadDir(temporaryRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("read-only check created files: %+v", entries)
	}
}

func TestCheckReportsNewerStableRelease(t *testing.T) {
	manager := testManager(t, t.TempDir(), func(request *http.Request) (*http.Response, error) {
		return jsonResponse(t, request, releaseJSON(t, "v0.4.1", true, true)), nil
	}, func(string) error { return nil })

	info, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if info.State != StateUpdateAvailable || info.LatestVersion != "0.4.1" {
		t.Fatalf("unexpected info: %+v", info)
	}
	if info.PublishedAt != "2026-08-28T03:04:05Z" || info.ReleaseNotes != "Release notes" {
		t.Fatalf("unexpected release details: %+v", info)
	}
	if info.ReleaseURL != "https://github.com/Useless007/content-blueprint/releases/tag/v0.4.1" {
		t.Fatalf("release URL = %q", info.ReleaseURL)
	}
}

func TestCheckRejectsMalformedAndOversizedResponses(t *testing.T) {
	tests := []struct {
		name     string
		response func(*http.Request) *http.Response
		contains string
	}{
		{
			name: "malformed JSON",
			response: func(request *http.Request) *http.Response {
				return byteResponse(request, http.StatusOK, []byte(`{"tag_name":`))
			},
			contains: "decode GitHub release metadata",
		},
		{
			name: "oversized Content-Length",
			response: func(request *http.Request) *http.Response {
				response := byteResponse(request, http.StatusOK, []byte("{}"))
				response.ContentLength = maximumReleaseResponse + 1
				return response
			},
			contains: "exceeds",
		},
		{
			name: "oversized streamed body",
			response: func(request *http.Request) *http.Response {
				body := bytes.Repeat([]byte{'x'}, int(maximumReleaseResponse+1))
				response := byteResponse(request, http.StatusOK, body)
				response.ContentLength = -1
				return response
			},
			contains: "exceeds",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := testManager(t, t.TempDir(), func(request *http.Request) (*http.Response, error) {
				return test.response(request), nil
			}, func(string) error { return nil })
			_, err := manager.Check(context.Background())
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("Check() error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func TestCheckRejectsMissingRequiredAssets(t *testing.T) {
	manager := testManager(t, t.TempDir(), func(request *http.Request) (*http.Response, error) {
		return jsonResponse(t, request, releaseJSON(t, "v0.3.1", false, true)), nil
	}, func(string) error { return nil })

	_, err := manager.Check(context.Background())
	if err == nil || !strings.Contains(err.Error(), installerAsset) {
		t.Fatalf("Check() error = %v, want missing installer asset", err)
	}
}

func TestCheckRejectsIncompleteOrUnboundedAssets(t *testing.T) {
	tests := []struct {
		name            string
		installerSize   int64
		checksumSize    int64
		installerState  string
		checksumState   string
		installerDigest string
		want            string
	}{
		{name: "installer not uploaded", installerSize: 10, checksumSize: 10, installerState: "new", checksumState: "uploaded", want: "not fully uploaded"},
		{name: "checksum not uploaded", installerSize: 10, checksumSize: 10, installerState: "uploaded", checksumState: "new", want: "not fully uploaded"},
		{name: "empty installer", installerSize: 0, checksumSize: 10, installerState: "uploaded", checksumState: "uploaded", want: "invalid size"},
		{name: "oversized installer", installerSize: maximumInstallerSize + 1, checksumSize: 10, installerState: "uploaded", checksumState: "uploaded", want: "invalid size"},
		{name: "empty checksum", installerSize: 10, checksumSize: 0, installerState: "uploaded", checksumState: "uploaded", want: "invalid size"},
		{name: "oversized checksum", installerSize: 10, checksumSize: maximumChecksumResponse + 1, installerState: "uploaded", checksumState: "uploaded", want: "invalid size"},
		{name: "malformed optional digest", installerSize: 10, checksumSize: 10, installerState: "uploaded", checksumState: "uploaded", installerDigest: "sha256:not-hex", want: "invalid SHA-256 digest"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := testManager(t, t.TempDir(), func(request *http.Request) (*http.Response, error) {
				body := releaseJSONWithMetadata(t, "v0.3.1", true, true, test.installerSize, test.checksumSize, test.installerState, test.checksumState, test.installerDigest, "")
				return jsonResponse(t, request, body), nil
			}, func(string) error { return nil })
			_, err := manager.Check(context.Background())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Check() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestDownloadRequiresGitHubAssetDigestsToAgree(t *testing.T) {
	installer := []byte("installer bytes")
	installerDigest := sha256.Sum256(installer)
	checksumBody := []byte(hex.EncodeToString(installerDigest[:]) + "  " + installerAsset + "\n")
	checksumDigest := sha256.Sum256(checksumBody)

	tests := []struct {
		name            string
		installerDigest string
		checksumDigest  string
		servedChecksum  []byte
		want            string
	}{
		{
			name:            "installer digest disagrees with checksum list",
			installerDigest: "sha256:" + strings.Repeat("0", 64),
			checksumDigest:  "sha256:" + hex.EncodeToString(checksumDigest[:]),
			servedChecksum:  checksumBody,
			want:            "does not agree",
		},
		{
			name:            "checksum asset digest disagrees with bytes",
			installerDigest: "sha256:" + hex.EncodeToString(installerDigest[:]),
			checksumDigest:  "sha256:" + strings.Repeat("0", 64),
			servedChecksum:  checksumBody,
			want:            "GitHub asset digest",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := testManager(t, t.TempDir(), func(request *http.Request) (*http.Response, error) {
				switch request.URL.String() {
				case testAPIBaseURL + "/releases/latest":
					body := releaseJSONWithMetadata(t, "v0.3.1", true, true, int64(len(installer)), int64(len(test.servedChecksum)), "uploaded", "uploaded", test.installerDigest, test.checksumDigest)
					return jsonResponse(t, request, body), nil
				case assetURL("v0.3.1", checksumsAsset):
					return byteResponse(request, http.StatusOK, test.servedChecksum), nil
				default:
					return nil, errors.New("installer must not be fetched after digest disagreement")
				}
			}, func(string) error { return nil })
			_, err := manager.Download(context.Background(), "0.3.1", nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Download() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestDownloadAcceptsAgreeingGitHubAssetDigests(t *testing.T) {
	installer := []byte("installer with matching metadata digests")
	installerDigest := sha256.Sum256(installer)
	checksumBody := []byte(hex.EncodeToString(installerDigest[:]) + "  " + installerAsset + "\n")
	checksumDigest := sha256.Sum256(checksumBody)
	manager := testManager(t, t.TempDir(), func(request *http.Request) (*http.Response, error) {
		switch request.URL.String() {
		case testAPIBaseURL + "/releases/latest":
			body := releaseJSONWithMetadata(
				t, "v0.3.1", true, true, int64(len(installer)), int64(len(checksumBody)), "uploaded", "uploaded",
				"sha256:"+hex.EncodeToString(installerDigest[:]), "sha256:"+hex.EncodeToString(checksumDigest[:]),
			)
			return jsonResponse(t, request, body), nil
		case assetURL("v0.3.1", checksumsAsset):
			return byteResponse(request, http.StatusOK, checksumBody), nil
		case assetURL("v0.3.1", installerAsset):
			return byteResponse(request, http.StatusOK, installer), nil
		default:
			return nil, errors.New("unexpected URL")
		}
	}, func(string) error { return nil })

	info, err := manager.Download(context.Background(), "0.3.1", nil)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if info.State != StateReady {
		t.Fatalf("Download() state = %q", info.State)
	}
}

func TestDownloadRejectsChecksumMismatchAndCleansPartialFiles(t *testing.T) {
	root := t.TempDir()
	installer := []byte("not the installer named in the checksum")
	manager := releaseManager(t, root, "v0.3.1", installer, strings.Repeat("0", 64), func(string) error {
		t.Fatal("mismatched installer must not launch")
		return nil
	})

	_, err := manager.Download(context.Background(), "0.3.1", nil)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("Download() error = %v, want checksum mismatch", err)
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("failed download left files behind: %+v", entries)
	}
}

func TestDownloadRejectsRequestedVersionMismatch(t *testing.T) {
	var checksumRequested atomic.Bool
	manager := testManager(t, t.TempDir(), func(request *http.Request) (*http.Response, error) {
		if request.URL.String() == testAPIBaseURL+"/releases/latest" {
			return jsonResponse(t, request, releaseJSON(t, "v0.3.2", true, true)), nil
		}
		checksumRequested.Store(true)
		return nil, errors.New("unexpected asset request")
	}, func(string) error { return nil })

	_, err := manager.Download(context.Background(), "0.3.1", nil)
	if err == nil || !strings.Contains(err.Error(), "does not match latest release 0.3.2") {
		t.Fatalf("Download() error = %v, want version mismatch", err)
	}
	if checksumRequested.Load() {
		t.Fatal("version mismatch must be rejected before asset download")
	}
}

func TestLaunchRejectsArbitraryAndUnverifiedVersions(t *testing.T) {
	var launched atomic.Int32
	manager := testManager(t, t.TempDir(), func(request *http.Request) (*http.Response, error) {
		return nil, errors.New("network must not be used")
	}, func(string) error {
		launched.Add(1)
		return nil
	})

	for _, version := range []string{"../../evil.exe", "v0.3.1", "0.3.1"} {
		if err := manager.Launch(version); err == nil {
			t.Fatalf("Launch(%q) unexpectedly succeeded", version)
		}
	}
	if launched.Load() != 0 {
		t.Fatalf("launcher called %d times", launched.Load())
	}
}

func TestDownloadHonorsCancellation(t *testing.T) {
	root := t.TempDir()
	installer := []byte("installer")
	digest := sha256.Sum256(installer)
	checksumBody := []byte(hex.EncodeToString(digest[:]) + "  " + installerAsset + "\n")
	started := make(chan struct{})
	manager := testManager(t, root, func(request *http.Request) (*http.Response, error) {
		switch request.URL.String() {
		case testAPIBaseURL + "/releases/latest":
			body := releaseJSONWithMetadata(t, "v0.3.1", true, true, int64(len(installer)), int64(len(checksumBody)), "uploaded", "uploaded", "", "")
			return jsonResponse(t, request, body), nil
		case assetURL("v0.3.1", checksumsAsset):
			return byteResponse(request, http.StatusOK, checksumBody), nil
		case assetURL("v0.3.1", installerAsset):
			close(started)
			<-request.Context().Done()
			return nil, request.Context().Err()
		default:
			return nil, errors.New("unexpected URL")
		}
	}, func(string) error { return nil })

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := manager.Download(ctx, "0.3.1", nil)
		result <- err
	}()
	select {
	case <-started:
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("installer request did not start")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Download() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Download() did not return after cancellation")
	}
	if err := manager.Launch("0.3.1"); err == nil {
		t.Fatal("canceled download became launchable")
	}
}

func TestVerifiedDownloadCanLaunchOnlyItsBackendOwnedPath(t *testing.T) {
	root := t.TempDir()
	installer := []byte("verified Windows installer bytes")
	digest := sha256.Sum256(installer)
	var launchedPath string
	manager := releaseManager(t, root, "v0.3.1", installer, hex.EncodeToString(digest[:]), func(path string) error {
		launchedPath = path
		return nil
	})
	var progress []Progress

	info, err := manager.Download(context.Background(), "0.3.1", func(update Progress) {
		progress = append(progress, update)
	})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if info.State != StateReady || info.LatestVersion != "0.3.1" {
		t.Fatalf("unexpected ready info: %+v", info)
	}
	if len(progress) < 2 || progress[len(progress)-1].Percent != 100 {
		t.Fatalf("progress = %+v, want completion", progress)
	}
	if err := manager.Launch("0.3.1"); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if filepath.Base(launchedPath) != installerAsset {
		t.Fatalf("launched path = %q", launchedPath)
	}
	relative, err := filepath.Rel(manager.temporaryRoot, launchedPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatalf("launcher received path outside backend temp root: %q", launchedPath)
	}
	contents, err := os.ReadFile(launchedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents, installer) {
		t.Fatalf("launched installer contents = %q", contents)
	}
}

func TestLaunchRejectsInstallerChangedAfterVerification(t *testing.T) {
	root := t.TempDir()
	installer := []byte("original installer")
	digest := sha256.Sum256(installer)
	var launches atomic.Int32
	manager := releaseManager(t, root, "v0.3.1", installer, hex.EncodeToString(digest[:]), func(string) error {
		launches.Add(1)
		return nil
	})
	if _, err := manager.Download(context.Background(), "0.3.1", nil); err != nil {
		t.Fatal(err)
	}
	manager.mu.RLock()
	path := manager.verified["0.3.1"].path
	manager.mu.RUnlock()
	if err := os.WriteFile(path, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.Launch("0.3.1"); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("Launch() error = %v, want tamper rejection", err)
	}
	if launches.Load() != 0 {
		t.Fatal("tampered installer reached launcher")
	}
}

func TestManagerRejectsNonHTTPSBaseURL(t *testing.T) {
	_, err := newManager(dependencies{
		httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, nil })},
		baseURL:    "http://api.test.invalid/repo",
		launcher:   func(string) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("newManager() error = %v, want HTTPS rejection", err)
	}
}

func TestReleaseNotesArePlainTextSafeAndBounded(t *testing.T) {
	input := "\x00first\r\n<script>alert(1)</script>\x07\n" + strings.Repeat("ก", maximumReleaseNotes)
	result := sanitizeReleaseNotes(input)
	if strings.ContainsAny(result, "\x00\x07\r") {
		t.Fatalf("sanitized notes retain control characters: %q", result[:min(len(result), 80)])
	}
	if !strings.Contains(result, "<script>alert(1)</script>") {
		t.Fatal("release notes must remain plain text rather than silently rewriting content")
	}
	if len(result) > maximumReleaseNotes {
		t.Fatalf("sanitized notes length = %d, want <= %d", len(result), maximumReleaseNotes)
	}
	if strings.ToValidUTF8(result, "") != result {
		t.Fatal("sanitized notes are not valid UTF-8")
	}
}

func TestCleanupRemovesOnlyOldManagerOwnedFilesAndDirectories(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	oldDirectory := filepath.Join(root, updateDirPrefix+"old")
	recentDirectory := filepath.Join(root, updateDirPrefix+"recent")
	unrelatedDirectory := filepath.Join(root, "another-application-update-old")
	unexpectedDirectory := filepath.Join(root, updateDirPrefix+"contains-user-file")
	for _, directory := range []string{oldDirectory, recentDirectory, unrelatedDirectory, unexpectedDirectory} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{installerAsset, installerAsset + ".partial"} {
		if err := os.WriteFile(filepath.Join(oldDirectory, name), []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(recentDirectory, installerAsset), []byte("recent"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unexpectedDirectory, "keep-me.txt"), []byte("user data"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldTime := now.Add(-staleUpdateAge - time.Hour)
	for _, directory := range []string{oldDirectory, unrelatedDirectory, unexpectedDirectory} {
		if err := os.Chtimes(directory, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(recentDirectory, now, now); err != nil {
		t.Fatal(err)
	}

	manager := testManager(t, root, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network must not be used")
	}, func(string) error { return nil })
	if err := manager.cleanupStaleDirectories(now); err != nil {
		t.Fatalf("cleanupStaleDirectories() error = %v", err)
	}
	if _, err := os.Stat(oldDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old owned directory still exists: %v", err)
	}
	for _, path := range []string{recentDirectory, unrelatedDirectory, filepath.Join(unexpectedDirectory, "keep-me.txt")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("cleanup removed unrelated or recent path %s: %v", path, err)
		}
	}
}

func testManager(t *testing.T, root string, transport roundTripFunc, launcher launchFunc) *Manager {
	t.Helper()
	manager, err := newManager(dependencies{
		httpClient:      &http.Client{Transport: transport},
		baseURL:         testAPIBaseURL,
		launcher:        launcher,
		temporaryRoot:   root,
		checkTimeout:    2 * time.Second,
		downloadTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func releaseManager(t *testing.T, root, tag string, installer []byte, checksum string, launcher launchFunc) *Manager {
	t.Helper()
	checksumBody := []byte(checksum + "  " + installerAsset + "\n")
	return testManager(t, root, func(request *http.Request) (*http.Response, error) {
		switch request.URL.String() {
		case testAPIBaseURL + "/releases/latest":
			body := releaseJSONWithMetadata(t, tag, true, true, int64(len(installer)), int64(len(checksumBody)), "uploaded", "uploaded", "", "")
			return jsonResponse(t, request, body), nil
		case assetURL(tag, checksumsAsset):
			return byteResponse(request, http.StatusOK, checksumBody), nil
		case assetURL(tag, installerAsset):
			return byteResponse(request, http.StatusOK, installer), nil
		default:
			return nil, errors.New("unexpected URL: " + request.URL.String())
		}
	}, launcher)
}

func releaseJSON(t *testing.T, tag string, includeInstaller, includeChecksums bool) []byte {
	t.Helper()
	return releaseJSONWithMetadata(t, tag, includeInstaller, includeChecksums, 1024, 128, "uploaded", "uploaded", "", "")
}

func releaseJSONWithMetadata(t *testing.T, tag string, includeInstaller, includeChecksums bool, installerSize, checksumSize int64, installerState, checksumState, installerDigest, checksumDigest string) []byte {
	t.Helper()
	assets := make([]map[string]any, 0, 2)
	if includeInstaller {
		assets = append(assets, map[string]any{
			"name": installerAsset, "browser_download_url": assetURL(tag, installerAsset),
			"state": installerState, "size": installerSize, "digest": installerDigest,
		})
	}
	if includeChecksums {
		assets = append(assets, map[string]any{
			"name": checksumsAsset, "browser_download_url": assetURL(tag, checksumsAsset),
			"state": checksumState, "size": checksumSize, "digest": checksumDigest,
		})
	}
	value := map[string]any{
		"tag_name":     tag,
		"html_url":     "https://github.com/Useless007/content-blueprint/releases/tag/" + tag,
		"published_at": "2026-08-28T10:04:05+07:00",
		"body":         "Release notes",
		"draft":        false,
		"prerelease":   false,
		"assets":       assets,
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func assetURL(tag, name string) string {
	return "https://github.com/Useless007/content-blueprint/releases/download/" + tag + "/" + name
}

func jsonResponse(t *testing.T, request *http.Request, body []byte) *http.Response {
	t.Helper()
	response := byteResponse(request, http.StatusOK, body)
	response.Header.Set("Content-Type", "application/json")
	return response
}

func byteResponse(request *http.Request, status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       request,
	}
}
