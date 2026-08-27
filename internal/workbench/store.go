package workbench

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ContentBlueprint/internal/domain"
)

var ErrNotFound = errors.New("growth workbench state not found")

type BriefSnapshot struct {
	Version       int         `json:"version"`
	BriefRevision string      `json:"briefRevision"`
	Brief         GrowthBrief `json:"brief"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}

type PackSnapshot struct {
	Version           int        `json:"version"`
	BriefRevision     string     `json:"briefRevision"`
	PlaybookID        string     `json:"playbookId"`
	EvidenceSourceIDs []string   `json:"evidenceSourceIds"`
	Pack              GrowthPack `json:"pack"`
	GeneratedBy       string     `json:"generatedBy"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	ReviewStatus      string     `json:"reviewStatus"`
	ReviewerNote      string     `json:"reviewerNote,omitempty"`
	ReviewUpdatedAt   *time.Time `json:"reviewUpdatedAt,omitempty"`
}

type Store struct {
	directory string
	now       func() time.Time
	mu        sync.Mutex
}

func NewStore(directory string) (*Store, error) {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		if override := strings.TrimSpace(os.Getenv("CONTENT_BLUEPRINT_DATA_DIR")); override != "" {
			directory = filepath.Join(override, "GrowthWorkbench")
		} else {
			configurationDirectory, err := os.UserConfigDir()
			if err != nil {
				return nil, fmt.Errorf("locate user configuration directory: %w", err)
			}
			directory = filepath.Join(configurationDirectory, "ContentBlueprint", "GrowthWorkbench")
		}
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve growth workbench directory: %w", err)
	}
	return &Store{directory: absolute, now: time.Now}, nil
}

func (store *Store) Directory() string { return store.directory }

func (store *Store) SaveBrief(input GrowthBrief) (BriefSnapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := ValidateBrief(input); err != nil {
		return BriefSnapshot{}, err
	}
	brief := NormalizeBrief(input)
	revision, err := BriefRevision(brief)
	if err != nil {
		return BriefSnapshot{}, err
	}
	snapshot := BriefSnapshot{Version: StateVersion, BriefRevision: revision, Brief: brief, UpdatedAt: store.now().UTC()}
	if err := store.writeJSON("growth-brief.json", snapshot, MaxBriefJSONBytes+32_000); err != nil {
		return BriefSnapshot{}, err
	}
	return snapshot, nil
}

func (store *Store) LoadBrief() (BriefSnapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.loadBrief()
}

func (store *Store) loadBrief() (BriefSnapshot, error) {
	var snapshot BriefSnapshot
	if err := store.readJSON("growth-brief.json", &snapshot, MaxBriefJSONBytes+32_000); err != nil {
		return BriefSnapshot{}, err
	}
	if snapshot.Version != StateVersion {
		return BriefSnapshot{}, fmt.Errorf("unsupported brief state version %d", snapshot.Version)
	}
	revision, err := BriefRevision(snapshot.Brief)
	if err != nil {
		return BriefSnapshot{}, fmt.Errorf("stored brief is invalid: %w", err)
	}
	if revision != snapshot.BriefRevision {
		return BriefSnapshot{}, fmt.Errorf("stored brief revision does not match its contents")
	}
	snapshot.Brief = NormalizeBrief(snapshot.Brief)
	return snapshot, nil
}

func (store *Store) SavePack(briefRevision string, input GrowthPack, generatedBy string) (PackSnapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	brief, err := store.loadBrief()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return PackSnapshot{}, fmt.Errorf("no Growth Brief is available; sync a brief first")
		}
		return PackSnapshot{}, err
	}
	if strings.TrimSpace(briefRevision) == "" || strings.TrimSpace(briefRevision) != brief.BriefRevision {
		return PackSnapshot{}, fmt.Errorf("brief revision is stale; generate again from the current brief")
	}
	pack := NormalizePack(input)
	if err := ValidatePack(brief.Brief.PlaybookID, brief.Brief.Evidence, pack); err != nil {
		return PackSnapshot{}, err
	}
	generatedBy = strings.TrimSpace(generatedBy)
	if generatedBy == "" || len([]rune(generatedBy)) > 120 {
		return PackSnapshot{}, fmt.Errorf("generatedBy is required and must not exceed 120 characters")
	}
	snapshot := PackSnapshot{
		Version: StateVersion, BriefRevision: brief.BriefRevision, PlaybookID: brief.Brief.PlaybookID,
		EvidenceSourceIDs: evidenceIDs(brief.Brief.Evidence), Pack: pack, GeneratedBy: generatedBy,
		UpdatedAt: store.now().UTC(), ReviewStatus: "needs_review",
	}
	if err := store.writeJSON("growth-pack.json", snapshot, MaxPackJSONBytes+64_000); err != nil {
		return PackSnapshot{}, err
	}
	return snapshot, nil
}

func (store *Store) LoadPack() (PackSnapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.loadPack()
}

func (store *Store) loadPack() (PackSnapshot, error) {
	var snapshot PackSnapshot
	if err := store.readJSON("growth-pack.json", &snapshot, MaxPackJSONBytes+64_000); err != nil {
		return PackSnapshot{}, err
	}
	if snapshot.Version != StateVersion || snapshot.BriefRevision == "" {
		return PackSnapshot{}, fmt.Errorf("stored Growth Pack metadata is invalid")
	}
	evidence := make([]domain.EvidenceSource, len(snapshot.EvidenceSourceIDs))
	for index, sourceID := range snapshot.EvidenceSourceIDs {
		evidence[index].ID = sourceID
	}
	if err := ValidatePack(snapshot.PlaybookID, evidence, snapshot.Pack); err != nil {
		return PackSnapshot{}, fmt.Errorf("stored Growth Pack is invalid: %w", err)
	}
	if !validReviewStatus(snapshot.ReviewStatus) {
		return PackSnapshot{}, fmt.Errorf("stored review status is invalid")
	}
	snapshot.Pack = NormalizePack(snapshot.Pack)
	return snapshot, nil
}

func evidenceIDs(evidence []domain.EvidenceSource) []string {
	result := make([]string, len(evidence))
	for index, source := range evidence {
		result[index] = source.ID
	}
	return result
}

func (store *Store) ReviewPack(briefRevision, status, note string) (PackSnapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	status = strings.TrimSpace(status)
	note = strings.TrimSpace(note)
	if !validReviewStatus(status) {
		return PackSnapshot{}, fmt.Errorf("reviewStatus must be needs_review, approved, or rejected")
	}
	if len([]rune(note)) > 4_000 {
		return PackSnapshot{}, fmt.Errorf("reviewerNote exceeds 4000 characters")
	}
	brief, err := store.loadBrief()
	if err != nil {
		return PackSnapshot{}, err
	}
	snapshot, err := store.loadPack()
	if err != nil {
		return PackSnapshot{}, err
	}
	if strings.TrimSpace(briefRevision) != brief.BriefRevision || snapshot.BriefRevision != brief.BriefRevision {
		return PackSnapshot{}, fmt.Errorf("brief revision is stale; review the current generated pack")
	}
	if (status == "approved" || status == "rejected") && note == "" {
		return PackSnapshot{}, fmt.Errorf("reviewerNote is required for approval or rejection")
	}
	now := store.now().UTC()
	snapshot.ReviewStatus = status
	snapshot.ReviewerNote = note
	snapshot.ReviewUpdatedAt = &now
	if err := store.writeJSON("growth-pack.json", snapshot, MaxPackJSONBytes+64_000); err != nil {
		return PackSnapshot{}, err
	}
	return snapshot, nil
}

func validReviewStatus(value string) bool {
	return value == "needs_review" || value == "approved" || value == "rejected"
}

func (store *Store) writeJSON(name string, value any, maximum int) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	if len(encoded) > maximum {
		return fmt.Errorf("%s exceeds the storage limit", name)
	}
	if err := os.MkdirAll(store.directory, 0o700); err != nil {
		return fmt.Errorf("create growth workbench directory: %w", err)
	}
	temporary, err := os.CreateTemp(store.directory, ".growth-workbench-*.tmp")
	if err != nil {
		return err
	}
	nameTemporary := temporary.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(nameTemporary)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(encoded, '\n')); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	destination := filepath.Join(store.directory, name)
	if err := os.Rename(nameTemporary, destination); err != nil {
		if removeErr := os.Remove(destination); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return err
		}
		if err := os.Rename(nameTemporary, destination); err != nil {
			return err
		}
	}
	remove = false
	return nil
}

func (store *Store) readJSON(name string, destination any, maximum int) error {
	encoded, err := os.ReadFile(filepath.Join(store.directory, name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}
	if len(encoded) > maximum {
		return fmt.Errorf("%s exceeds the storage limit", name)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode %s: unexpected trailing data", name)
	}
	return nil
}
