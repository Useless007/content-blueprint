package facebookcompanion

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
	"time"
)

const dataDirectoryEnvironment = "CONTENT_BLUEPRINT_DATA_DIR"

var ErrNotFound = errors.New("facebook companion state not found")

type Store struct {
	directory string
	now       func() time.Time
}

func NewStore(directory string) (*Store, error) {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		directory = strings.TrimSpace(os.Getenv(dataDirectoryEnvironment))
	}
	if directory == "" {
		configDirectory, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("locate user configuration directory: %w", err)
		}
		directory = filepath.Join(configDirectory, "ContentBlueprint", "FacebookCompanion")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve companion data directory: %w", err)
	}
	return &Store{directory: absolute, now: time.Now}, nil
}

func (store *Store) Directory() string {
	return store.directory
}

func (store *Store) SaveBrief(input Brief) (BriefSnapshot, error) {
	brief := NormalizeBrief(input)
	revision, err := BriefRevision(brief)
	if err != nil {
		return BriefSnapshot{}, err
	}
	snapshot := BriefSnapshot{
		Version:       StateVersion,
		BriefRevision: revision,
		Brief:         brief,
		UpdatedAt:     store.now().UTC(),
	}
	if err := store.writeJSON("facebook-brief.json", snapshot, MaxBriefJSONBytes+16_384); err != nil {
		return BriefSnapshot{}, err
	}
	return snapshot, nil
}

func (store *Store) LoadBrief() (BriefSnapshot, error) {
	var snapshot BriefSnapshot
	if err := store.readJSON("facebook-brief.json", &snapshot, MaxBriefJSONBytes+16_384); err != nil {
		return BriefSnapshot{}, err
	}
	if snapshot.Version != StateVersion {
		return BriefSnapshot{}, fmt.Errorf("unsupported brief state version %d", snapshot.Version)
	}
	revision, err := BriefRevision(snapshot.Brief)
	if err != nil {
		return BriefSnapshot{}, fmt.Errorf("stored brief is invalid: %w", err)
	}
	if snapshot.BriefRevision != revision {
		return BriefSnapshot{}, fmt.Errorf("stored brief revision does not match its contents")
	}
	snapshot.Brief = NormalizeBrief(snapshot.Brief)
	return snapshot, nil
}

func (store *Store) SavePack(
	briefRevision string,
	input ContentPack,
	groundingSources []GroundingSource,
	generatedBy string,
) (PackSnapshot, error) {
	briefRevision = strings.TrimSpace(briefRevision)
	brief, err := store.LoadBrief()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return PackSnapshot{}, fmt.Errorf("no Facebook brief is available; sync a brief from Content Blueprint first")
		}
		return PackSnapshot{}, err
	}
	if briefRevision == "" || briefRevision != brief.BriefRevision {
		return PackSnapshot{}, fmt.Errorf("brief revision is stale; read the current brief and generate again")
	}
	pack := NormalizeContentPack(input)
	if err := ValidateContentPack(pack); err != nil {
		return PackSnapshot{}, err
	}
	normalizedSources, err := NormalizeGroundingSources(groundingSources)
	if err != nil {
		return PackSnapshot{}, err
	}
	generatedBy = strings.TrimSpace(generatedBy)
	if len([]rune(generatedBy)) > 120 {
		return PackSnapshot{}, fmt.Errorf("generatedBy exceeds 120 characters")
	}
	snapshot := PackSnapshot{
		Version:          StateVersion,
		BriefRevision:    briefRevision,
		Pack:             pack,
		GroundingSources: normalizedSources,
		GeneratedBy:      generatedBy,
		UpdatedAt:        store.now().UTC(),
	}
	if err := store.writeJSON("facebook-pack.json", snapshot, MaxContentPackBytes+64_000); err != nil {
		return PackSnapshot{}, err
	}
	return snapshot, nil
}

func (store *Store) LoadPack() (PackSnapshot, error) {
	var snapshot PackSnapshot
	if err := store.readJSON("facebook-pack.json", &snapshot, MaxContentPackBytes+64_000); err != nil {
		return PackSnapshot{}, err
	}
	if snapshot.Version != StateVersion {
		return PackSnapshot{}, fmt.Errorf("unsupported content pack state version %d", snapshot.Version)
	}
	if snapshot.BriefRevision == "" {
		return PackSnapshot{}, fmt.Errorf("stored content pack has no brief revision")
	}
	if err := ValidateContentPack(snapshot.Pack); err != nil {
		return PackSnapshot{}, fmt.Errorf("stored content pack is invalid: %w", err)
	}
	sources, err := NormalizeGroundingSources(snapshot.GroundingSources)
	if err != nil {
		return PackSnapshot{}, fmt.Errorf("stored grounding sources are invalid: %w", err)
	}
	snapshot.Pack = NormalizeContentPack(snapshot.Pack)
	snapshot.GroundingSources = sources
	return snapshot, nil
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
		return fmt.Errorf("create companion data directory: %w", err)
	}
	temporary, err := os.CreateTemp(store.directory, ".content-blueprint-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary companion state: %w", err)
	}
	temporaryName := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryName)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("secure temporary companion state: %w", err)
	}
	if _, err := temporary.Write(append(encoded, '\n')); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary companion state: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("flush temporary companion state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary companion state: %w", err)
	}
	destination := filepath.Join(store.directory, name)
	if err := replaceFile(temporaryName, destination); err != nil {
		return fmt.Errorf("replace %s: %w", name, err)
	}
	removeTemporary = false
	return nil
}

func (store *Store) readJSON(name string, destination any, maximum int) error {
	path := filepath.Join(store.directory, name)
	encoded, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("read %s: %w", name, err)
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
		return fmt.Errorf("decode %s: unexpected trailing value", name)
	}
	return nil
}

// os.Rename replaces existing files atomically on Unix. Windows can reject an
// existing destination in some environments, so fall back to a narrow remove
// of this package's exact state-file path before the second rename attempt.
func replaceFile(source, destination string) error {
	if err := os.Rename(source, destination); err == nil {
		return nil
	} else {
		if removeErr := os.Remove(destination); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return err
		}
		return os.Rename(source, destination)
	}
}
