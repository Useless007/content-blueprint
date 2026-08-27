// Package updater checks and prepares SHA-256-verified Windows installer
// updates published by the fixed Content Blueprint GitHub repository.
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"ContentBlueprint/internal/versioninfo"
)

const (
	githubAPIBaseURL = "https://api.github.com/repos/Useless007/content-blueprint"
	installerAsset   = "content-blueprint-amd64-installer.exe"
	checksumsAsset   = "SHA256SUMS.txt"
	githubAPIVersion = "2022-11-28"
	updateDirPrefix  = "content-blueprint-update-"

	maximumReleaseResponse  int64 = 1 << 20
	maximumChecksumResponse int64 = 64 << 10
	maximumInstallerSize    int64 = 512 << 20
	maximumErrorResponse    int64 = 4 << 10
	maximumReleaseNotes           = 32 << 10

	checkTimeout    = 20 * time.Second
	downloadTimeout = 15 * time.Minute
	staleUpdateAge  = 7 * 24 * time.Hour
)

var stableSemverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// State describes the only update states exposed to the desktop frontend.
type State string

const (
	StateUpToDate        State = "up_to_date"
	StateUpdateAvailable State = "update_available"
	StateDownloading     State = "downloading"
	StateReady           State = "ready"
)

// Info is the concise update state returned through Wails.
type Info struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	State          State  `json:"state"`
	ReleaseURL     string `json:"releaseUrl,omitempty"`
	PublishedAt    string `json:"publishedAt,omitempty"`
	ReleaseNotes   string `json:"releaseNotes,omitempty"`
}

// Progress is emitted while the installer is downloaded against GitHub's
// bounded release-asset size.
type Progress struct {
	Version         string `json:"version"`
	DownloadedBytes int64  `json:"downloadedBytes"`
	TotalBytes      int64  `json:"totalBytes"`
	Percent         int    `json:"percent"`
}

// ProgressFunc observes download progress without owning update state.
type ProgressFunc func(Progress)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type launchFunc func(string) error

type dependencies struct {
	httpClient      httpDoer
	baseURL         string
	launcher        launchFunc
	temporaryRoot   string
	checkTimeout    time.Duration
	downloadTimeout time.Duration
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	State              string `json:"state"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
}

type githubRelease struct {
	TagName     string         `json:"tag_name"`
	HTMLURL     string         `json:"html_url"`
	PublishedAt string         `json:"published_at"`
	Body        string         `json:"body"`
	Draft       bool           `json:"draft"`
	Prerelease  bool           `json:"prerelease"`
	Assets      []releaseAsset `json:"assets"`
}

type release struct {
	version      semver
	versionText  string
	releaseURL   string
	publishedAt  string
	releaseNotes string
	installer    releaseFile
	checksums    releaseFile
}

type releaseFile struct {
	url    string
	size   int64
	digest *[sha256.Size]byte
}

type verifiedDownload struct {
	path   string
	digest [sha256.Size]byte
	info   Info
}

// Manager is a deep update module. The caller supplies only a version when it
// requests a mutation; URLs, paths, checksums, and verified launch state stay
// inside this module.
type Manager struct {
	httpClient      httpDoer
	baseURL         string
	baseHost        string
	launcher        launchFunc
	temporaryRoot   string
	checkTimeout    time.Duration
	downloadTimeout time.Duration
	current         semver

	downloadMu sync.Mutex
	mu         sync.RWMutex
	verified   map[string]verifiedDownload
}

// New constructs the production updater for the fixed public GitHub repository.
func New() *Manager {
	client := &http.Client{
		Timeout: downloadTimeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if request.URL.Scheme != "https" {
				return fmt.Errorf("update redirect must use HTTPS")
			}
			if !isTrustedGitHubHost(request.URL.Hostname()) {
				return fmt.Errorf("update redirect left trusted GitHub hosts")
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many update redirects")
			}
			return nil
		},
	}
	manager, err := newManager(dependencies{
		httpClient:      client,
		baseURL:         githubAPIBaseURL,
		launcher:        launchInstaller,
		checkTimeout:    checkTimeout,
		downloadTimeout: downloadTimeout,
	})
	if err != nil {
		panic("invalid built-in updater configuration: " + err.Error())
	}
	_ = manager.cleanupStaleDirectories(time.Now())
	return manager
}

func newManager(deps dependencies) (*Manager, error) {
	if deps.httpClient == nil {
		return nil, fmt.Errorf("update HTTP client is required")
	}
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(deps.baseURL), "/"))
	if err != nil || base.Scheme != "https" || base.Host == "" || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("update base URL must be an absolute HTTPS URL")
	}
	if deps.launcher == nil {
		return nil, fmt.Errorf("update launcher is required")
	}
	current, err := parseSemver(versioninfo.CurrentVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid current application version: %w", err)
	}
	if deps.checkTimeout <= 0 {
		deps.checkTimeout = checkTimeout
	}
	if deps.downloadTimeout <= 0 {
		deps.downloadTimeout = downloadTimeout
	}
	temporaryRoot := strings.TrimSpace(deps.temporaryRoot)
	if temporaryRoot == "" {
		temporaryRoot = os.TempDir()
	}
	temporaryRoot, err = filepath.Abs(temporaryRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve update temporary root: %w", err)
	}
	temporaryRoot = filepath.Clean(temporaryRoot)
	rootInfo, err := os.Stat(temporaryRoot)
	if err != nil || !rootInfo.IsDir() {
		return nil, fmt.Errorf("update temporary root is unavailable")
	}
	temporaryRoot, err = filepath.EvalSymlinks(temporaryRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve update temporary root links: %w", err)
	}
	temporaryRoot = filepath.Clean(temporaryRoot)
	return &Manager{
		httpClient:      deps.httpClient,
		baseURL:         strings.TrimRight(base.String(), "/"),
		baseHost:        strings.ToLower(base.Hostname()),
		launcher:        deps.launcher,
		temporaryRoot:   temporaryRoot,
		checkTimeout:    deps.checkTimeout,
		downloadTimeout: deps.downloadTimeout,
		current:         current,
		verified:        make(map[string]verifiedDownload),
	}, nil
}

// Check reads the latest stable GitHub release. It does not create or modify
// any file and never downloads an installer.
func (manager *Manager) Check(ctx context.Context) (Info, error) {
	ctx, cancel := context.WithTimeout(nonNilContext(ctx), manager.checkTimeout)
	defer cancel()

	release, err := manager.fetchLatestRelease(ctx)
	if err != nil {
		return Info{}, err
	}
	return manager.infoFor(release), nil
}

// Download retrieves the exact installer and checksum assets for version,
// verifies SHA-256, then records the backend-owned path as ready to launch.
func (manager *Manager) Download(ctx context.Context, version string, progress ProgressFunc) (Info, error) {
	requested, err := parseSemver(version)
	if err != nil || requested.String() != version {
		return Info{}, fmt.Errorf("requested update version must be a strict stable semantic version")
	}
	if requested.compare(manager.current) <= 0 {
		return Info{}, fmt.Errorf("requested update version must be newer than %s", versioninfo.CurrentVersion)
	}

	manager.downloadMu.Lock()
	defer manager.downloadMu.Unlock()

	manager.mu.RLock()
	ready, alreadyVerified := manager.verified[version]
	manager.mu.RUnlock()
	if alreadyVerified {
		if err := verifyFileDigest(ready.path, ready.digest); err == nil {
			return ready.info, nil
		}
		manager.mu.Lock()
		delete(manager.verified, version)
		manager.mu.Unlock()
	}

	ctx, cancel := context.WithTimeout(nonNilContext(ctx), manager.downloadTimeout)
	defer cancel()

	release, err := manager.fetchLatestRelease(ctx)
	if err != nil {
		return Info{}, err
	}
	if release.version.compare(requested) != 0 {
		return Info{}, fmt.Errorf("requested version %s does not match latest release %s", version, release.versionText)
	}

	checksumBody, err := manager.readBounded(ctx, release.checksums.url, maximumChecksumResponse, "checksum")
	if err != nil {
		return Info{}, err
	}
	if int64(len(checksumBody)) != release.checksums.size {
		return Info{}, fmt.Errorf("downloaded %s size does not match GitHub release metadata", checksumsAsset)
	}
	if release.checksums.digest != nil {
		actualChecksumDigest := sha256.Sum256(checksumBody)
		if actualChecksumDigest != *release.checksums.digest {
			return Info{}, fmt.Errorf("downloaded %s does not match its GitHub asset digest", checksumsAsset)
		}
	}
	expected, err := installerChecksum(checksumBody)
	if err != nil {
		return Info{}, err
	}
	if release.installer.digest != nil && expected != *release.installer.digest {
		return Info{}, fmt.Errorf("GitHub installer digest does not agree with %s", checksumsAsset)
	}

	directory, err := os.MkdirTemp(manager.temporaryRoot, updateDirPrefix)
	if err != nil {
		return Info{}, fmt.Errorf("create private update directory: %w", err)
	}
	keepDirectory := false
	defer func() {
		if !keepDirectory {
			_ = manager.removeOwnedUpdateDirectory(directory)
		}
	}()

	partialPath := filepath.Join(directory, installerAsset+".partial")
	finalPath := filepath.Join(directory, installerAsset)
	actual, err := manager.downloadInstaller(ctx, release.installer.url, partialPath, version, release.installer.size, progress)
	if err != nil {
		return Info{}, err
	}
	if actual != expected {
		return Info{}, fmt.Errorf("installer SHA-256 does not match %s", checksumsAsset)
	}
	if err := os.Rename(partialPath, finalPath); err != nil {
		return Info{}, fmt.Errorf("finalize verified installer: %w", err)
	}

	info := manager.infoFor(release)
	info.State = StateReady
	manager.mu.Lock()
	manager.verified[version] = verifiedDownload{path: finalPath, digest: actual, info: info}
	manager.mu.Unlock()
	keepDirectory = true
	return info, nil
}

// Launch starts only an installer downloaded and verified by this Manager.
// It does not wait for the installer and it never quits the application.
func (manager *Manager) Launch(version string) error {
	parsed, err := parseSemver(version)
	if err != nil || parsed.String() != version {
		return fmt.Errorf("launch version must be a strict stable semantic version")
	}

	manager.mu.RLock()
	verified, ok := manager.verified[version]
	manager.mu.RUnlock()
	if !ok {
		return fmt.Errorf("version %s has not been downloaded and verified in this session", version)
	}
	if filepath.Base(verified.path) != installerAsset {
		return fmt.Errorf("verified installer path is invalid")
	}
	if err := verifyFileDigest(verified.path, verified.digest); err != nil {
		manager.mu.Lock()
		delete(manager.verified, version)
		manager.mu.Unlock()
		return err
	}
	if err := manager.launcher(verified.path); err != nil {
		return fmt.Errorf("launch verified installer: %w", err)
	}
	return nil
}

func (manager *Manager) fetchLatestRelease(ctx context.Context) (release, error) {
	body, err := manager.readBounded(ctx, manager.baseURL+"/releases/latest", maximumReleaseResponse, "release metadata")
	if err != nil {
		return release{}, err
	}
	var response githubRelease
	if err := json.Unmarshal(body, &response); err != nil {
		return release{}, fmt.Errorf("decode GitHub release metadata: %w", err)
	}
	if response.Draft || response.Prerelease {
		return release{}, fmt.Errorf("latest GitHub release is not a stable published release")
	}
	tag := strings.TrimSpace(response.TagName)
	versionText := strings.TrimPrefix(tag, "v")
	if tag != versionText && tag != "v"+versionText {
		return release{}, fmt.Errorf("release tag is invalid")
	}
	version, err := parseSemver(versionText)
	if err != nil {
		return release{}, fmt.Errorf("release tag %q is not a strict stable semantic version", tag)
	}

	assets := make(map[string]releaseFile, 2)
	for _, asset := range response.Assets {
		if asset.Name != installerAsset && asset.Name != checksumsAsset {
			continue
		}
		if _, duplicate := assets[asset.Name]; duplicate {
			return release{}, fmt.Errorf("release contains duplicate %s assets", asset.Name)
		}
		if err := validateAssetURL(asset.BrowserDownloadURL, tag, asset.Name); err != nil {
			return release{}, err
		}
		if asset.State != "uploaded" {
			return release{}, fmt.Errorf("release asset %s is not fully uploaded", asset.Name)
		}
		maximumSize := maximumChecksumResponse
		if asset.Name == installerAsset {
			maximumSize = maximumInstallerSize
		}
		if asset.Size <= 0 || asset.Size > maximumSize {
			return release{}, fmt.Errorf("release asset %s has invalid size %d", asset.Name, asset.Size)
		}
		digest, err := parseGitHubAssetDigest(asset.Digest, asset.Name)
		if err != nil {
			return release{}, err
		}
		assets[asset.Name] = releaseFile{url: asset.BrowserDownloadURL, size: asset.Size, digest: digest}
	}
	for _, name := range []string{installerAsset, checksumsAsset} {
		if assets[name].url == "" {
			return release{}, fmt.Errorf("release %s is missing required asset %s", versionText, name)
		}
	}

	releaseURL := strings.TrimSpace(response.HTMLURL)
	u, parseErr := url.Parse(releaseURL)
	expectedPath := "/Useless007/content-blueprint/releases/tag/" + tag
	if parseErr != nil || u.Scheme != "https" || u.Hostname() != "github.com" || u.EscapedPath() != expectedPath || u.RawQuery != "" || u.Fragment != "" {
		return release{}, fmt.Errorf("release page URL is invalid")
	}
	publishedAt := ""
	if rawPublishedAt := strings.TrimSpace(response.PublishedAt); rawPublishedAt != "" {
		published, parseErr := time.Parse(time.RFC3339, rawPublishedAt)
		if parseErr != nil {
			return release{}, fmt.Errorf("release published_at is invalid")
		}
		publishedAt = published.UTC().Format(time.RFC3339)
	}
	return release{
		version:      version,
		versionText:  versionText,
		releaseURL:   releaseURL,
		publishedAt:  publishedAt,
		releaseNotes: sanitizeReleaseNotes(response.Body),
		installer:    assets[installerAsset],
		checksums:    assets[checksumsAsset],
	}, nil
}

func (manager *Manager) infoFor(latest release) Info {
	state := StateUpToDate
	if latest.version.compare(manager.current) > 0 {
		state = StateUpdateAvailable
	}
	return Info{
		CurrentVersion: versioninfo.CurrentVersion,
		LatestVersion:  latest.versionText,
		State:          state,
		ReleaseURL:     latest.releaseURL,
		PublishedAt:    latest.publishedAt,
		ReleaseNotes:   latest.releaseNotes,
	}
}

func (manager *Manager) readBounded(ctx context.Context, resourceURL string, maximum int64, label string) ([]byte, error) {
	response, err := manager.get(ctx, resourceURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", label, err)
	}
	defer response.Body.Close()
	if err := requireSuccess(response, label); err != nil {
		return nil, err
	}
	if response.ContentLength > maximum {
		return nil, fmt.Errorf("%s response exceeds %d bytes", label, maximum)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maximum+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	if int64(len(body)) > maximum {
		return nil, fmt.Errorf("%s response exceeds %d bytes", label, maximum)
	}
	return body, nil
}

func (manager *Manager) downloadInstaller(ctx context.Context, resourceURL, path, version string, expectedSize int64, progress ProgressFunc) ([sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	response, err := manager.get(ctx, resourceURL)
	if err != nil {
		return digest, fmt.Errorf("fetch installer: %w", err)
	}
	defer response.Body.Close()
	if err := requireSuccess(response, "installer"); err != nil {
		return digest, err
	}
	if response.ContentLength > maximumInstallerSize {
		return digest, fmt.Errorf("installer response exceeds %d bytes", maximumInstallerSize)
	}
	if response.ContentLength >= 0 && response.ContentLength != expectedSize {
		return digest, fmt.Errorf("installer Content-Length does not match GitHub release metadata")
	}
	total := expectedSize
	emitProgress(progress, version, 0, total, false)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return digest, fmt.Errorf("create private installer file: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()

	hash := sha256.New()
	reader := &io.LimitedReader{R: response.Body, N: maximumInstallerSize + 1}
	buffer := make([]byte, 128<<10)
	var downloaded int64
	for {
		count, readErr := reader.Read(buffer)
		if count > 0 {
			downloaded += int64(count)
			if downloaded > maximumInstallerSize {
				return digest, fmt.Errorf("installer response exceeds %d bytes", maximumInstallerSize)
			}
			if _, err := file.Write(buffer[:count]); err != nil {
				return digest, fmt.Errorf("write installer: %w", err)
			}
			_, _ = hash.Write(buffer[:count])
			emitProgress(progress, version, downloaded, total, false)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return digest, fmt.Errorf("read installer: %w", readErr)
		}
		if err := ctx.Err(); err != nil {
			return digest, err
		}
	}
	if err := file.Sync(); err != nil {
		return digest, fmt.Errorf("sync installer: %w", err)
	}
	if err := file.Close(); err != nil {
		return digest, fmt.Errorf("close installer: %w", err)
	}
	closed = true
	if downloaded != expectedSize {
		return digest, fmt.Errorf("downloaded installer size does not match GitHub release metadata")
	}
	copy(digest[:], hash.Sum(nil))
	emitProgress(progress, version, downloaded, total, true)
	return digest, nil
}

func (manager *Manager) get(ctx context.Context, resourceURL string) (*http.Response, error) {
	u, err := url.Parse(resourceURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || !manager.isAllowedResourceHost(u.Hostname()) {
		return nil, fmt.Errorf("update resource must use a trusted HTTPS host")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	request.Header.Set("User-Agent", "Content-Blueprint/"+versioninfo.CurrentVersion)
	response, err := manager.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Body == nil {
		return nil, fmt.Errorf("empty HTTP response")
	}
	if response.Request != nil && response.Request.URL != nil {
		finalURL := response.Request.URL
		if finalURL.Scheme != "https" || !manager.isAllowedResourceHost(finalURL.Hostname()) {
			_ = response.Body.Close()
			return nil, fmt.Errorf("update response left trusted HTTPS hosts")
		}
	}
	return response, nil
}

func (manager *Manager) isAllowedResourceHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == manager.baseHost || isTrustedGitHubHost(host)
}

func isTrustedGitHubHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "github.com" || host == "api.github.com" || host == "githubusercontent.com" || strings.HasSuffix(host, ".githubusercontent.com")
}

func requireSuccess(response *http.Response, label string) error {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(response.Body, maximumErrorResponse))
	detail := strings.TrimSpace(string(body))
	if detail != "" {
		return fmt.Errorf("fetch %s: HTTP %d: %s", label, response.StatusCode, detail)
	}
	return fmt.Errorf("fetch %s: HTTP %d", label, response.StatusCode)
}

func validateAssetURL(rawURL, tag, assetName string) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme != "https" || u.Hostname() != "github.com" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("release asset %s must use the fixed GitHub repository HTTPS URL", assetName)
	}
	expectedPath := "/Useless007/content-blueprint/releases/download/" + tag + "/" + assetName
	if u.EscapedPath() != expectedPath {
		return fmt.Errorf("release asset %s does not match release tag %s", assetName, tag)
	}
	return nil
}

func parseGitHubAssetDigest(value, assetName string) (*[sha256.Size]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if !strings.HasPrefix(value, "sha256:") {
		return nil, fmt.Errorf("release asset %s has unsupported digest", assetName)
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	if err != nil || len(decoded) != sha256.Size {
		return nil, fmt.Errorf("release asset %s has invalid SHA-256 digest", assetName)
	}
	var digest [sha256.Size]byte
	copy(digest[:], decoded)
	return &digest, nil
}

func installerChecksum(body []byte) ([sha256.Size]byte, error) {
	var result [sha256.Size]byte
	matchCount := 0
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name != installerAsset {
			continue
		}
		decoded, err := hex.DecodeString(fields[0])
		if err != nil || len(decoded) != sha256.Size {
			return result, fmt.Errorf("invalid SHA-256 for %s", installerAsset)
		}
		matchCount++
		if matchCount > 1 {
			return result, fmt.Errorf("duplicate checksum entries for %s", installerAsset)
		}
		copy(result[:], decoded)
	}
	if matchCount != 1 {
		return result, fmt.Errorf("%s does not contain an exact checksum for %s", checksumsAsset, installerAsset)
	}
	return result, nil
}

func (manager *Manager) cleanupStaleDirectories(now time.Time) error {
	entries, err := os.ReadDir(manager.temporaryRoot)
	if err != nil {
		return fmt.Errorf("scan update temporary root: %w", err)
	}
	cutoff := now.Add(-staleUpdateAge)
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), updateDirPrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = manager.removeOwnedUpdateDirectory(filepath.Join(manager.temporaryRoot, entry.Name()))
	}
	return nil
}

// removeOwnedUpdateDirectory intentionally is not recursive. It removes only
// updater-owned filenames from one direct child directory, then removes that
// directory only if it is empty.
func (manager *Manager) removeOwnedUpdateDirectory(directory string) error {
	root := filepath.Clean(manager.temporaryRoot)
	directory, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("resolve update directory: %w", err)
	}
	directory = filepath.Clean(directory)
	relative, err := filepath.Rel(root, directory)
	if err != nil || filepath.Dir(relative) != "." || !strings.HasPrefix(filepath.Base(relative), updateDirPrefix) {
		return fmt.Errorf("refuse to remove directory outside the update temporary root")
	}
	info, err := os.Lstat(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect update directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse to remove non-directory update path")
	}
	resolvedDirectory, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return fmt.Errorf("resolve update directory links: %w", err)
	}
	resolvedDirectory = filepath.Clean(resolvedDirectory)
	resolvedRelative, err := filepath.Rel(root, resolvedDirectory)
	if err != nil || filepath.Dir(resolvedRelative) != "." || !strings.HasPrefix(filepath.Base(resolvedRelative), updateDirPrefix) {
		return fmt.Errorf("refuse to remove linked directory outside the update temporary root")
	}
	directory = resolvedDirectory
	for _, name := range []string{installerAsset + ".partial", installerAsset} {
		path := filepath.Join(directory, name)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale update file: %w", err)
		}
	}
	if err := os.Remove(directory); err != nil {
		return fmt.Errorf("remove empty stale update directory: %w", err)
	}
	return nil
}

func verifyFileDigest(path string, expected [sha256.Size]byte) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open verified installer: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect verified installer: %w", err)
	}
	if info.Size() > maximumInstallerSize {
		return fmt.Errorf("verified installer exceeds %d bytes", maximumInstallerSize)
	}
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(file, maximumInstallerSize+1))
	if err != nil {
		return fmt.Errorf("hash verified installer: %w", err)
	}
	if count > maximumInstallerSize {
		return fmt.Errorf("verified installer exceeds %d bytes", maximumInstallerSize)
	}
	actual := hash.Sum(nil)
	if !equalDigest(actual, expected[:]) {
		return fmt.Errorf("verified installer changed after download")
	}
	return nil
}

func equalDigest(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var difference byte
	for index := range left {
		difference |= left[index] ^ right[index]
	}
	return difference == 0
}

func emitProgress(callback ProgressFunc, version string, downloaded, total int64, complete bool) {
	if callback == nil {
		return
	}
	percent := 0
	if total > 0 {
		percent = int(downloaded * 100 / total)
		if percent > 100 {
			percent = 100
		}
	}
	if complete {
		percent = 100
	}
	callback(Progress{Version: version, DownloadedBytes: downloaded, TotalBytes: total, Percent: percent})
}

func sanitizeReleaseNotes(value string) string {
	value = strings.ToValidUTF8(value, "")
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	var result strings.Builder
	result.Grow(min(len(value), maximumReleaseNotes))
	for _, character := range value {
		if unicode.IsControl(character) && character != '\n' && character != '\t' {
			continue
		}
		size := utf8.RuneLen(character)
		if size < 0 || result.Len()+size > maximumReleaseNotes {
			break
		}
		result.WriteRune(character)
	}
	return strings.TrimSpace(result.String())
}

func launchInstaller(path string) error {
	command := exec.Command(path)
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

type semver struct {
	major uint64
	minor uint64
	patch uint64
}

func parseSemver(value string) (semver, error) {
	if !stableSemverPattern.MatchString(value) {
		return semver{}, fmt.Errorf("invalid semantic version %q", value)
	}
	parts := strings.Split(value, ".")
	values := make([]uint64, len(parts))
	for index, part := range parts {
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return semver{}, fmt.Errorf("invalid semantic version %q", value)
		}
		values[index] = parsed
	}
	return semver{major: values[0], minor: values[1], patch: values[2]}, nil
}

func (version semver) String() string {
	return fmt.Sprintf("%d.%d.%d", version.major, version.minor, version.patch)
}

func (version semver) compare(other semver) int {
	left := []uint64{version.major, version.minor, version.patch}
	right := []uint64{other.major, other.minor, other.patch}
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}
