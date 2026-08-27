package salesops

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const MaxLeads = 5_000

var leadIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)

type Lead struct {
	ID                        string `json:"id"`
	Label                     string `json:"label"`
	Source                    string `json:"source,omitempty"`
	Offer                     string `json:"offer,omitempty"`
	Stage                     string `json:"stage"`
	Needs                     string `json:"needs,omitempty"`
	Objections                string `json:"objections,omitempty"`
	Assignee                  string `json:"assignee,omitempty"`
	NextFollowUp              string `json:"nextFollowUp,omitempty"`
	HandoffNote               string `json:"handoffNote,omitempty"`
	CampaignID                string `json:"campaignId,omitempty"`
	UTM                       string `json:"utm,omitempty"`
	SaleAmountSatang          int64  `json:"saleAmountSatang"`
	CommissionRateBps         int64  `json:"commissionRateBps"`
	EstimatedCommissionSatang int64  `json:"estimatedCommissionSatang"`
	CommissionConfirmed       bool   `json:"commissionConfirmed"`
	ConfirmedCommissionSatang int64  `json:"confirmedCommissionSatang"`
	ConfirmedBy               string `json:"confirmedBy,omitempty"`
	ConfirmedAt               string `json:"confirmedAt,omitempty"`
	CreatedAt                 string `json:"createdAt"`
	UpdatedAt                 string `json:"updatedAt"`
}

type Store struct {
	directory string
	mu        sync.Mutex
	now       func() time.Time
}

func NewStore(directory string) (*Store, error) {
	absolute, err := filepath.Abs(strings.TrimSpace(directory))
	if err != nil || strings.TrimSpace(directory) == "" {
		return nil, fmt.Errorf("resolve sales operations directory")
	}
	return &Store{directory: absolute, now: time.Now}, nil
}

func NormalizeLead(input Lead) Lead {
	input.ID = strings.TrimSpace(input.ID)
	input.Label = strings.TrimSpace(input.Label)
	input.Source = strings.TrimSpace(input.Source)
	input.Offer = strings.TrimSpace(input.Offer)
	input.Stage = strings.TrimSpace(input.Stage)
	input.Needs = strings.TrimSpace(input.Needs)
	input.Objections = strings.TrimSpace(input.Objections)
	input.Assignee = strings.TrimSpace(input.Assignee)
	input.NextFollowUp = strings.TrimSpace(input.NextFollowUp)
	input.HandoffNote = strings.TrimSpace(input.HandoffNote)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.UTM = strings.TrimSpace(input.UTM)
	input.ConfirmedBy = strings.TrimSpace(input.ConfirmedBy)
	input.ConfirmedAt = strings.TrimSpace(input.ConfirmedAt)
	return input
}

func ValidateLead(input Lead) error {
	lead := NormalizeLead(input)
	if !leadIDPattern.MatchString(lead.ID) {
		return fmt.Errorf("lead id is invalid")
	}
	if lead.Label == "" || len([]rune(lead.Label)) > 500 {
		return fmt.Errorf("lead label is required and must not exceed 500 characters")
	}
	if lead.Stage != "new" && lead.Stage != "qualified" && lead.Stage != "owner_review" && lead.Stage != "won" && lead.Stage != "lost" {
		return fmt.Errorf("lead stage is invalid")
	}
	for name, value := range map[string]string{
		"source": lead.Source, "offer": lead.Offer, "needs": lead.Needs, "objections": lead.Objections,
		"assignee": lead.Assignee, "handoffNote": lead.HandoffNote, "campaignId": lead.CampaignID, "utm": lead.UTM,
	} {
		if len([]rune(value)) > 4_000 {
			return fmt.Errorf("%s exceeds 4000 characters", name)
		}
	}
	if lead.NextFollowUp != "" {
		if _, err := time.Parse("2006-01-02", lead.NextFollowUp); err != nil {
			return fmt.Errorf("nextFollowUp must use YYYY-MM-DD")
		}
	}
	if lead.SaleAmountSatang < 0 || lead.CommissionRateBps < 0 || lead.CommissionRateBps > 10_000 || lead.ConfirmedCommissionSatang < 0 {
		return fmt.Errorf("money and commission values are out of range")
	}
	estimated, err := EstimateCommission(lead.SaleAmountSatang, lead.CommissionRateBps)
	if err != nil || lead.EstimatedCommissionSatang != estimated {
		return fmt.Errorf("estimatedCommissionSatang does not match integer money inputs")
	}
	if lead.CommissionConfirmed {
		if lead.ConfirmedBy == "" || len([]rune(lead.ConfirmedBy)) > 500 {
			return fmt.Errorf("confirmedBy is required for commission confirmation")
		}
		if lead.ConfirmedAt == "" {
			return fmt.Errorf("confirmedAt is required for commission confirmation")
		}
		if _, err := time.Parse(time.RFC3339, lead.ConfirmedAt); err != nil {
			return fmt.Errorf("confirmedAt must use RFC3339")
		}
	} else if lead.ConfirmedCommissionSatang != 0 || lead.ConfirmedBy != "" || lead.ConfirmedAt != "" {
		return fmt.Errorf("unconfirmed commission must not include confirmation fields")
	}
	return nil
}

func EstimateCommission(saleAmountSatang, commissionRateBps int64) (int64, error) {
	if saleAmountSatang < 0 || commissionRateBps < 0 || commissionRateBps > 10_000 {
		return 0, fmt.Errorf("money or commission rate is invalid")
	}
	if commissionRateBps != 0 && saleAmountSatang > math.MaxInt64/commissionRateBps {
		return 0, fmt.Errorf("commission calculation overflows")
	}
	return saleAmountSatang * commissionRateBps / 10_000, nil
}

func (store *Store) Save(input Lead) (Lead, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	leads, err := store.load()
	if err != nil {
		return Lead{}, err
	}
	lead := NormalizeLead(input)
	if lead.ID == "" {
		lead.ID, err = randomID("lead")
		if err != nil {
			return Lead{}, err
		}
	}
	estimated, err := EstimateCommission(lead.SaleAmountSatang, lead.CommissionRateBps)
	if err != nil {
		return Lead{}, err
	}
	lead.EstimatedCommissionSatang = estimated
	now := store.now().UTC()
	if lead.CommissionConfirmed && lead.ConfirmedAt == "" {
		lead.ConfirmedAt = now.Format(time.RFC3339)
	}
	if err := ValidateLead(lead); err != nil {
		return Lead{}, err
	}
	found := false
	for index := range leads {
		if leads[index].ID == lead.ID {
			lead.CreatedAt = leads[index].CreatedAt
			found = true
			leads[index] = lead
			break
		}
	}
	if !found {
		if len(leads) >= MaxLeads {
			return Lead{}, fmt.Errorf("lead record limit reached")
		}
		lead.CreatedAt = now.Format(time.RFC3339)
		leads = append(leads, lead)
	}
	lead.UpdatedAt = now.Format(time.RFC3339)
	for index := range leads {
		if leads[index].ID == lead.ID {
			leads[index] = lead
		}
	}
	if err := store.write(leads); err != nil {
		return Lead{}, err
	}
	return lead, nil
}

func (store *Store) List() ([]Lead, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	leads, err := store.load()
	if err != nil {
		return nil, err
	}
	sort.Slice(leads, func(i, j int) bool { return leads[i].UpdatedAt > leads[j].UpdatedAt })
	return leads, nil
}

func (store *Store) Delete(id string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	id = strings.TrimSpace(id)
	if !leadIDPattern.MatchString(id) {
		return fmt.Errorf("lead id is invalid")
	}
	leads, err := store.load()
	if err != nil {
		return err
	}
	result := make([]Lead, 0, len(leads))
	found := false
	for _, lead := range leads {
		if lead.ID == id {
			found = true
			continue
		}
		result = append(result, lead)
	}
	if !found {
		return fmt.Errorf("lead not found")
	}
	return store.write(result)
}

func (store *Store) load() ([]Lead, error) {
	path := filepath.Join(store.directory, "growth-leads.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return []Lead{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 8_000_000 {
		return nil, fmt.Errorf("lead storage exceeds safe limit")
	}
	var leads []Lead
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&leads); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("lead storage contains trailing data")
	}
	if len(leads) > MaxLeads {
		return nil, fmt.Errorf("lead storage exceeds record limit")
	}
	for index := range leads {
		if err := ValidateLead(leads[index]); err != nil {
			return nil, fmt.Errorf("stored lead %d is invalid: %w", index, err)
		}
	}
	return leads, nil
}

func (store *Store) write(leads []Lead) error {
	if err := os.MkdirAll(store.directory, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(leads, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(store.directory, ".growth-leads-*.tmp")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
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
	destination := filepath.Join(store.directory, "growth-leads.json")
	if err := os.Rename(name, destination); err != nil {
		if removeErr := os.Remove(destination); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return err
		}
		return os.Rename(name, destination)
	}
	return nil
}

func randomID(prefix string) (string, error) {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(value), nil
}
