package salesops

import "testing"

func TestLeadMoneyMathAndConfirmationBoundary(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	lead, err := store.Save(Lead{Label: "Manual lead card", Stage: "won", SaleAmountSatang: 123_45, CommissionRateBps: 1_250})
	if err != nil {
		t.Fatal(err)
	}
	if lead.EstimatedCommissionSatang != 1_543 {
		t.Fatalf("estimated commission = %d, want 1543", lead.EstimatedCommissionSatang)
	}
	lead.CommissionConfirmed = true
	lead.ConfirmedCommissionSatang = 1_500
	if _, err := store.Save(lead); err == nil {
		t.Fatal("commission confirmation without confirmedBy was accepted")
	}
	lead.ConfirmedBy = "Page owner"
	confirmed, err := store.Save(lead)
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.ConfirmedAt == "" || confirmed.ConfirmedCommissionSatang != 1_500 {
		t.Fatalf("confirmed lead = %#v", confirmed)
	}
	listed, err := store.List()
	if err != nil || len(listed) != 1 || listed[0].ID != confirmed.ID {
		t.Fatalf("List = %#v, %v", listed, err)
	}
	if err := store.Delete(confirmed.ID); err != nil {
		t.Fatal(err)
	}
	listed, err = store.List()
	if err != nil || len(listed) != 0 {
		t.Fatalf("List after delete = %#v, %v", listed, err)
	}
}

func TestLeadRejectsInvalidStageAndOverflow(t *testing.T) {
	if _, err := EstimateCommission(9_223_372_036_854_775_000, 10_000); err == nil {
		t.Fatal("overflowing commission was accepted")
	}
	lead := Lead{ID: "lead_1", Label: "Test", Stage: "contacted", EstimatedCommissionSatang: 0}
	if err := ValidateLead(lead); err == nil {
		t.Fatal("unknown lead stage was accepted")
	}
}
