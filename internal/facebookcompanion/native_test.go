package facebookcompanion

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"testing"

	"ContentBlueprint/internal/domain"
	"ContentBlueprint/internal/workbench"
)

func TestNativeHostSaveBriefAndFetchPack(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	briefPayload, err := json.Marshal(NativeRequest{Action: "saveBrief", Brief: pointerTo(validBrief())})
	if err != nil {
		t.Fatal(err)
	}
	input := frameForTest(briefPayload)
	var output bytes.Buffer
	if err := RunNativeHost(context.Background(), bytes.NewReader(input), &output, store); err != nil {
		t.Fatal(err)
	}
	response := decodeNativeResponseForTest(t, output.Bytes())
	if !response.OK || response.BriefRevision == "" {
		t.Fatalf("unexpected save response: %#v", response)
	}
	if _, err := store.SavePack(response.BriefRevision, validPack(), nil, "Claude"); err != nil {
		t.Fatal(err)
	}

	fetchPayload, _ := json.Marshal(NativeRequest{Action: "getLatestPack", BriefRevision: response.BriefRevision})
	output.Reset()
	if err := RunNativeHost(context.Background(), bytes.NewReader(frameForTest(fetchPayload)), &output, store); err != nil {
		t.Fatal(err)
	}
	fetch := decodeNativeResponseForTest(t, output.Bytes())
	if !fetch.OK || !fetch.Found || fetch.Stale || fetch.Snapshot == nil {
		t.Fatalf("unexpected fetch response: %#v", fetch)
	}

	changedBrief := validBrief()
	changedBrief.Topic += " updated"
	if _, err := store.SaveBrief(changedBrief); err != nil {
		t.Fatal(err)
	}
	fetchPayload, _ = json.Marshal(NativeRequest{Action: "getLatestPack"})
	output.Reset()
	if err := RunNativeHost(context.Background(), bytes.NewReader(frameForTest(fetchPayload)), &output, store); err != nil {
		t.Fatal(err)
	}
	fetch = decodeNativeResponseForTest(t, output.Bytes())
	if !fetch.OK || !fetch.Found || !fetch.Stale {
		t.Fatalf("expected latest pack to be stale after the stored brief changed: %#v", fetch)
	}
}

func TestNativeHostRejectsUnknownFields(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	response := HandleNativeRequest([]byte(`{"action":"health","secret":"must not pass"}`), store)
	if response.OK || response.ErrorCode != "INVALID_MESSAGE" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestNativeHostFetchesTypedGrowthPackAndMarksStale(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CONTENT_BLUEPRINT_DATA_DIR", root)
	facebookStore, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	growthStore, err := workbench.NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	brief := validNativeGrowthBrief()
	briefSnapshot, err := growthStore.SaveBrief(brief)
	if err != nil {
		t.Fatal(err)
	}
	packSnapshot, err := growthStore.SavePack(briefSnapshot.BriefRevision, validNativeGrowthPack(), "Codex CLI")
	if err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(NativeRequest{Action: "getLatestGrowthPack", BriefRevision: briefSnapshot.BriefRevision})
	response := HandleNativeRequest(payload, facebookStore)
	if !response.OK || !response.Found || response.Stale || response.GrowthSnapshot == nil {
		t.Fatalf("Growth response = %#v", response)
	}
	if response.Snapshot != nil || response.GrowthSnapshot.BriefRevision != packSnapshot.BriefRevision || response.GrowthSnapshot.ReviewStatus != "needs_review" {
		t.Fatalf("Growth response mixed legacy state or lost metadata: %#v", response)
	}

	changed := brief
	changed.Inputs = map[string]string{"offer": "Workshop", "audience": "Marketing teachers", "problems": "Unclear positioning"}
	if _, err := growthStore.SaveBrief(changed); err != nil {
		t.Fatal(err)
	}
	response = HandleNativeRequest([]byte(`{"action":"getLatestGrowthPack"}`), facebookStore)
	if !response.OK || !response.Found || !response.Stale || response.GrowthSnapshot == nil {
		t.Fatalf("stale Growth response = %#v", response)
	}
}

func TestValidateNativeOrigin(t *testing.T) {
	if err := ValidateNativeOrigin(AllowedExtensionOrigin); err != nil {
		t.Fatal(err)
	}
	if err := ValidateNativeOrigin("chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/"); err == nil {
		t.Fatal("expected foreign origin to be rejected")
	}
}

func frameForTest(payload []byte) []byte {
	framed := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(framed[:4], uint32(len(payload)))
	copy(framed[4:], payload)
	return framed
}

func decodeNativeResponseForTest(t *testing.T, framed []byte) NativeResponse {
	t.Helper()
	if len(framed) < 4 {
		t.Fatalf("response frame is too short: %d", len(framed))
	}
	length := binary.LittleEndian.Uint32(framed[:4])
	if int(length) != len(framed)-4 {
		t.Fatalf("response length %d != payload %d", length, len(framed)-4)
	}
	var response NativeResponse
	if err := json.Unmarshal(framed[4:], &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func pointerTo[T any](value T) *T {
	return &value
}

func validNativeGrowthBrief() workbench.GrowthBrief {
	return workbench.GrowthBrief{
		PlaybookID: "offer-audience", Language: "Thai", Inputs: map[string]string{
			"offer": "Workshop", "audience": "Shop owners", "problems": "Unclear positioning",
		},
		Evidence: []domain.EvidenceSource{{ID: "outline", Title: "Course outline", URL: "https://example.com/outline", Notes: "Eight lessons."}},
	}
}

func validNativeGrowthPack() workbench.GrowthPack {
	return workbench.GrowthPack{
		Title: "Offer plan", Summary: "A bounded plan.",
		Blocks: []workbench.GrowthBlock{{
			ID: "plan", Title: "Plan", Purpose: "Explain the offer", Kind: workbench.BlockProse,
			Body: "Use the supplied course outline.", Items: []workbench.BlockItem{}, Columns: []string{}, Rows: [][]string{}, Code: "",
			EvidenceBasis: workbench.BasisSuppliedEvidence, SourceIDs: []string{"outline"},
		}},
		OpenQuestions: []string{}, RiskFlags: []string{"Verify dates."},
		ReviewChecks: []workbench.ReviewCheck{{Status: "review", Label: "Owner review", Reason: "Check facts before use."}},
	}
}
