package facebookcompanion

import (
	"errors"
	"testing"
	"time"
)

func TestStoreBriefAndPackRoundTrip(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fixedTime := time.Date(2026, 8, 28, 1, 2, 3, 0, time.UTC)
	store.now = func() time.Time { return fixedTime }

	briefSnapshot, err := store.SaveBrief(validBrief())
	if err != nil {
		t.Fatal(err)
	}
	packSnapshot, err := store.SavePack(
		briefSnapshot.BriefRevision,
		validPack(),
		[]GroundingSource{{Title: "หลักสูตร", URL: "https://example.com/course"}},
		"Codex",
	)
	if err != nil {
		t.Fatal(err)
	}
	if packSnapshot.UpdatedAt != fixedTime || packSnapshot.GeneratedBy != "Codex" {
		t.Fatalf("unexpected saved snapshot: %#v", packSnapshot)
	}

	loaded, err := store.LoadPack()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.BriefRevision != briefSnapshot.BriefRevision || loaded.Pack.LongPost != validPack().LongPost {
		t.Fatalf("unexpected loaded pack: %#v", loaded)
	}
}

func TestStoreRejectsStalePack(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	brief, err := store.SaveBrief(validBrief())
	if err != nil {
		t.Fatal(err)
	}
	changed := validBrief()
	changed.Topic = "หัวข้อใหม่"
	if _, err := store.SaveBrief(changed); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SavePack(brief.BriefRevision, validPack(), nil, "Claude"); err == nil {
		t.Fatal("expected stale revision to be rejected")
	}
}

func TestStoreReportsMissingState(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadBrief(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
