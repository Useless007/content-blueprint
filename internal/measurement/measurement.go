package measurement

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const MaxExperiments = 2_000

var recordIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)

type VariantMetrics struct {
	Impressions   int64 `json:"impressions"`
	Clicks        int64 `json:"clicks"`
	Leads         int64 `json:"leads"`
	Sales         int64 `json:"sales"`
	RevenueSatang int64 `json:"revenueSatang"`
}

type Experiment struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Hypothesis    string         `json:"hypothesis"`
	Variable      string         `json:"variable"`
	VariantA      string         `json:"variantA"`
	VariantB      string         `json:"variantB"`
	StartDate     string         `json:"startDate,omitempty"`
	EndDate       string         `json:"endDate,omitempty"`
	Audience      string         `json:"audience,omitempty"`
	Channel       string         `json:"channel,omitempty"`
	PrimaryMetric string         `json:"primaryMetric"`
	Guardrail     string         `json:"guardrail,omitempty"`
	Comparable    bool           `json:"comparable"`
	MetricsA      VariantMetrics `json:"metricsA"`
	MetricsB      VariantMetrics `json:"metricsB"`
	Learning      string         `json:"learning,omitempty"`
	Decision      string         `json:"decision,omitempty"`
	ApprovedBy    string         `json:"approvedBy,omitempty"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
}

type DerivedRates struct {
	CTR       float64 `json:"ctr"`
	LeadRate  float64 `json:"leadRate"`
	CloseRate float64 `json:"closeRate"`
}

type ExperimentView struct {
	Experiment    Experiment   `json:"experiment"`
	RatesA        DerivedRates `json:"ratesA"`
	RatesB        DerivedRates `json:"ratesB"`
	AnalysisLabel string       `json:"analysisLabel"`
	Winner        string       `json:"winner"`
}

type Store struct {
	directory string
	mu        sync.Mutex
	now       func() time.Time
}

func NewStore(directory string) (*Store, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, fmt.Errorf("measurement directory is required")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	return &Store{directory: absolute, now: time.Now}, nil
}

func NormalizeExperiment(input Experiment) Experiment {
	input.ID = strings.TrimSpace(input.ID)
	input.Title = strings.TrimSpace(input.Title)
	input.Hypothesis = strings.TrimSpace(input.Hypothesis)
	input.Variable = strings.TrimSpace(input.Variable)
	input.VariantA = strings.TrimSpace(input.VariantA)
	input.VariantB = strings.TrimSpace(input.VariantB)
	input.StartDate = strings.TrimSpace(input.StartDate)
	input.EndDate = strings.TrimSpace(input.EndDate)
	input.Audience = strings.TrimSpace(input.Audience)
	input.Channel = strings.TrimSpace(input.Channel)
	input.PrimaryMetric = strings.TrimSpace(input.PrimaryMetric)
	input.Guardrail = strings.TrimSpace(input.Guardrail)
	input.Learning = strings.TrimSpace(input.Learning)
	input.Decision = strings.TrimSpace(input.Decision)
	input.ApprovedBy = strings.TrimSpace(input.ApprovedBy)
	return input
}

func ValidateExperiment(input Experiment) error {
	experiment := NormalizeExperiment(input)
	if !recordIDPattern.MatchString(experiment.ID) {
		return fmt.Errorf("experiment id is invalid")
	}
	for name, value := range map[string]string{
		"title": experiment.Title, "hypothesis": experiment.Hypothesis, "variable": experiment.Variable,
		"variantA": experiment.VariantA, "variantB": experiment.VariantB, "primaryMetric": experiment.PrimaryMetric,
	} {
		if value == "" || len([]rune(value)) > 4_000 {
			return fmt.Errorf("%s is required and must not exceed 4000 characters", name)
		}
	}
	for name, value := range map[string]string{
		"audience": experiment.Audience, "channel": experiment.Channel, "guardrail": experiment.Guardrail,
		"learning": experiment.Learning, "decision": experiment.Decision, "approvedBy": experiment.ApprovedBy,
	} {
		if len([]rune(value)) > 8_000 {
			return fmt.Errorf("%s exceeds 8000 characters", name)
		}
	}
	for name, value := range map[string]string{"startDate": experiment.StartDate, "endDate": experiment.EndDate} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return fmt.Errorf("%s must use YYYY-MM-DD", name)
			}
		}
	}
	if experiment.StartDate != "" && experiment.EndDate != "" && experiment.EndDate < experiment.StartDate {
		return fmt.Errorf("endDate must not precede startDate")
	}
	if err := validateMetrics("metricsA", experiment.MetricsA); err != nil {
		return err
	}
	if err := validateMetrics("metricsB", experiment.MetricsB); err != nil {
		return err
	}
	return nil
}

func validateMetrics(name string, metrics VariantMetrics) error {
	if metrics.Impressions < 0 || metrics.Clicks < 0 || metrics.Leads < 0 || metrics.Sales < 0 || metrics.RevenueSatang < 0 {
		return fmt.Errorf("%s counts and revenue must be non-negative", name)
	}
	if metrics.Clicks > metrics.Impressions || metrics.Leads > metrics.Clicks || metrics.Sales > metrics.Leads {
		return fmt.Errorf("%s funnel counts are inconsistent", name)
	}
	return nil
}

func Derived(metrics VariantMetrics) DerivedRates {
	return DerivedRates{
		CTR:       safeRate(metrics.Clicks, metrics.Impressions),
		LeadRate:  safeRate(metrics.Leads, metrics.Clicks),
		CloseRate: safeRate(metrics.Sales, metrics.Leads),
	}
}

func safeRate(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func View(experiment Experiment) ExperimentView {
	label := "directional"
	if experiment.Comparable {
		label = "comparable"
	}
	return ExperimentView{Experiment: experiment, RatesA: Derived(experiment.MetricsA), RatesB: Derived(experiment.MetricsB), AnalysisLabel: label, Winner: ""}
}

func (store *Store) Save(input Experiment) (ExperimentView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	records, err := store.load()
	if err != nil {
		return ExperimentView{}, err
	}
	experiment := NormalizeExperiment(input)
	if experiment.ID == "" {
		experiment.ID, err = randomID("exp")
		if err != nil {
			return ExperimentView{}, err
		}
	}
	if err := ValidateExperiment(experiment); err != nil {
		return ExperimentView{}, err
	}
	now := store.now().UTC().Format(time.RFC3339)
	found := false
	for index := range records {
		if records[index].ID == experiment.ID {
			experiment.CreatedAt = records[index].CreatedAt
			records[index] = experiment
			found = true
			break
		}
	}
	if !found {
		if len(records) >= MaxExperiments {
			return ExperimentView{}, fmt.Errorf("experiment record limit reached")
		}
		experiment.CreatedAt = now
		records = append(records, experiment)
	}
	experiment.UpdatedAt = now
	for index := range records {
		if records[index].ID == experiment.ID {
			records[index] = experiment
		}
	}
	if err := store.write(records); err != nil {
		return ExperimentView{}, err
	}
	return View(experiment), nil
}

func (store *Store) List() ([]ExperimentView, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	records, err := store.load()
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool { return records[i].UpdatedAt > records[j].UpdatedAt })
	views := make([]ExperimentView, len(records))
	for index, record := range records {
		views[index] = View(record)
	}
	return views, nil
}

func (store *Store) Delete(id string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	id = strings.TrimSpace(id)
	if !recordIDPattern.MatchString(id) {
		return fmt.Errorf("experiment id is invalid")
	}
	records, err := store.load()
	if err != nil {
		return err
	}
	result := make([]Experiment, 0, len(records))
	found := false
	for _, record := range records {
		if record.ID == id {
			found = true
			continue
		}
		result = append(result, record)
	}
	if !found {
		return fmt.Errorf("experiment not found")
	}
	return store.write(result)
}

func (store *Store) load() ([]Experiment, error) {
	data, err := os.ReadFile(filepath.Join(store.directory, "growth-experiments.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return []Experiment{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 12_000_000 {
		return nil, fmt.Errorf("experiment storage exceeds safe limit")
	}
	var records []Experiment
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&records); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("experiment storage contains trailing data")
	}
	if len(records) > MaxExperiments {
		return nil, fmt.Errorf("experiment storage exceeds record limit")
	}
	for index := range records {
		if err := ValidateExperiment(records[index]); err != nil {
			return nil, fmt.Errorf("stored experiment %d is invalid: %w", index, err)
		}
	}
	return records, nil
}

func (store *Store) write(records []Experiment) error {
	if err := os.MkdirAll(store.directory, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(store.directory, ".growth-experiments-*.tmp")
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
	destination := filepath.Join(store.directory, "growth-experiments.json")
	if err := os.Rename(name, destination); err != nil {
		if removeErr := os.Remove(destination); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return err
		}
		return os.Rename(name, destination)
	}
	return nil
}

type UTMRequest struct {
	DestinationURL string `json:"destinationUrl"`
	Source         string `json:"source"`
	Medium         string `json:"medium"`
	Campaign       string `json:"campaign"`
	Content        string `json:"content,omitempty"`
	Term           string `json:"term,omitempty"`
}

type UTMResult struct {
	URL        string `json:"url"`
	CampaignID string `json:"campaignId"`
}

func BuildUTM(input UTMRequest) (UTMResult, error) {
	if len(input.DestinationURL) > 4_096 || strings.Contains(input.DestinationURL, "\\") {
		return UTMResult{}, fmt.Errorf("destination URL is invalid")
	}
	parsed, err := url.Parse(strings.TrimSpace(input.DestinationURL))
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return UTMResult{}, fmt.Errorf("destination URL must be absolute HTTP(S) without credentials")
	}
	values := map[string]string{
		"utm_source": normalizeUTM(input.Source), "utm_medium": normalizeUTM(input.Medium),
		"utm_campaign": normalizeUTM(input.Campaign), "utm_content": normalizeUTM(input.Content), "utm_term": normalizeUTM(input.Term),
	}
	if values["utm_source"] == "" || values["utm_medium"] == "" || values["utm_campaign"] == "" {
		return UTMResult{}, fmt.Errorf("source, medium, and campaign are required")
	}
	query := parsed.Query()
	for key, value := range values {
		query.Del(key)
		if value != "" {
			query.Set(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	campaignID := values["utm_source"] + "-" + values["utm_medium"] + "-" + values["utm_campaign"]
	if len(campaignID) > 180 {
		digest := sha256.Sum256([]byte(campaignID))
		campaignID = campaignID[:160] + "-" + hex.EncodeToString(digest[:8])
	}
	return UTMResult{URL: parsed.String(), CampaignID: campaignID}, nil
}

func normalizeUTM(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	separator := false
	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || unicode.IsMark(character) {
			builder.WriteRune(character)
			separator = false
		} else if builder.Len() > 0 && !separator {
			builder.WriteByte('-')
			separator = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func randomID(prefix string) (string, error) {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(value), nil
}
