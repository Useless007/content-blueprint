package cliprovider

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"ContentBlueprint/internal/facebookcompanion"
	"ContentBlueprint/internal/workbench"
)

type liveProcessFailure struct {
	exitCode int
	stderr   string
}

type liveRecordingRunner struct {
	OSRunner
	failures []liveProcessFailure
}

func (runner *liveRecordingRunner) Run(ctx context.Context, command Command) (ProcessResult, error) {
	result, err := runner.OSRunner.Run(ctx, command)
	if err != nil {
		runner.failures = append(runner.failures, liveProcessFailure{
			exitCode: result.ExitCode,
			stderr:   liveDiagnosticExcerpt(result.Stderr),
		})
	}
	return result, err
}

var liveEmailPattern = regexp.MustCompile(`(?i)[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}`)

func liveDiagnosticExcerpt(value []byte) string {
	text := strings.TrimSpace(string(value))
	text = liveEmailPattern.ReplaceAllString(text, "[email redacted]")
	if profile := strings.TrimSpace(os.Getenv("USERPROFILE")); profile != "" {
		text = strings.ReplaceAll(strings.ToLower(text), strings.ToLower(profile), "[user profile]")
	}
	if len(text) > 2_000 {
		text = "…" + text[len(text)-2_000:]
	}
	return text
}

// TestLiveProviderSmoke is opt-in because it consumes the signed-in CLI
// provider's subscription quota. Example:
//
//	CONTENT_BLUEPRINT_LIVE_PROVIDERS=codex,claude go test -run TestLiveProviderSmoke ./internal/cliprovider -v
//	CONTENT_BLUEPRINT_LIVE_PROVIDERS=codex CONTENT_BLUEPRINT_LIVE_WORKFLOW=team go test -run TestLiveProviderSmoke ./internal/cliprovider -v
func TestLiveProviderSmoke(t *testing.T) {
	rawProviders := strings.TrimSpace(os.Getenv("CONTENT_BLUEPRINT_LIVE_PROVIDERS"))
	if rawProviders == "" {
		t.Skip("set CONTENT_BLUEPRINT_LIVE_PROVIDERS to run signed-in CLI smoke tests")
	}
	workflow := WorkflowSingle
	if strings.EqualFold(strings.TrimSpace(os.Getenv("CONTENT_BLUEPRINT_LIVE_WORKFLOW")), "team") {
		workflow = WorkflowTeam
	}

	brief := facebookcompanion.Brief{
		Topic:          "Content Blueprint รุ่นสาธิตภายใน",
		Audience:       "ผู้ดูแลเพจที่ต้องการเตรียมร่าง Facebook ให้เจ้าของเพจตรวจ",
		Objective:      "อธิบายขั้นตอนทดลองเครื่องมือโดยไม่ชักชวนให้ซื้อ",
		BrandVoice:     "ชัดเจน สุภาพ ไม่กล่าวเกินจริง",
		Language:       "th",
		ProductDetails: "เป็นข้อมูลสาธิตสำหรับทดสอบระบบเท่านั้น ไม่มีราคา โปรโมชั่น หรือคำรับรองผลลัพธ์",
		AdditionalInstructions: "เขียนให้กระชับและระบุใน complianceNotes ว่าเป็นข้อมูลสาธิต " +
			"ห้ามอ้างว่าโพสต์นี้ถูกเผยแพร่แล้ว",
	}

	seen := make(map[Provider]bool)
	for _, name := range strings.Split(rawProviders, ",") {
		provider := Provider(strings.ToLower(strings.TrimSpace(name)))
		if !provider.Valid() {
			t.Fatalf("unsupported live provider %q", name)
		}
		if seen[provider] {
			continue
		}
		seen[provider] = true
		t.Run(string(provider), func(t *testing.T) {
			runner := &liveRecordingRunner{}
			service := NewWithRunner(runner)
			ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)
			defer cancel()
			status := service.Status(ctx, provider, "")
			if !status.Available || !status.AuthenticationChecked || !status.Authenticated {
				t.Fatalf("provider is not installed and authenticated: %s", status.Message)
			}
			pack, err := service.Generate(ctx, brief, Options{
				Provider: provider,
				Workflow: workflow,
				Timeout:  7 * time.Minute,
			})
			if err != nil {
				for index, failure := range runner.failures {
					t.Logf("failed process %d: exit=%d stderr=%q", index+1, failure.exitCode, failure.stderr)
				}
				t.Fatalf("live generation failed: %v", err)
			}
			if err := facebookcompanion.ValidateContentPack(pack); err != nil {
				t.Fatalf("provider returned an invalid pack: %v", err)
			}
		})
	}
}

// TestLiveGrowthProviderSmoke verifies the real signed-in CLI path for the
// Growth Workbench. Team mode launches three isolated CLI processes and is the
// closest smoke test to the subagent workflow shown in the app.
//
//	CONTENT_BLUEPRINT_LIVE_GROWTH_PROVIDERS=codex go test -run TestLiveGrowthProviderSmoke ./internal/cliprovider -v
//	CONTENT_BLUEPRINT_LIVE_GROWTH_PROVIDERS=codex CONTENT_BLUEPRINT_LIVE_GROWTH_WORKFLOW=team go test -run TestLiveGrowthProviderSmoke ./internal/cliprovider -v
func TestLiveGrowthProviderSmoke(t *testing.T) {
	rawProviders := strings.TrimSpace(os.Getenv("CONTENT_BLUEPRINT_LIVE_GROWTH_PROVIDERS"))
	if rawProviders == "" {
		t.Skip("set CONTENT_BLUEPRINT_LIVE_GROWTH_PROVIDERS to run signed-in Growth CLI smoke tests")
	}
	workflow := WorkflowSingle
	if strings.EqualFold(strings.TrimSpace(os.Getenv("CONTENT_BLUEPRINT_LIVE_GROWTH_WORKFLOW")), "team") {
		workflow = WorkflowTeam
	}

	brief := workbench.GrowthBrief{
		PlaybookID: "facebook-campaign",
		Language:   "th",
		BrandVoice: "ตรงประเด็น สุภาพ ไม่กล่าวเกินหลักฐาน",
		Inputs: map[string]string{
			"objective": "วางแผนคอนเทนต์สาธิตที่พาคนอ่านไปดูรายละเอียด โดยไม่อ้างผลลัพธ์หรือส่วนลด",
			"audience":  "ผู้ดูแลเพจธุรกิจขนาดเล็กที่ต้องลดงานเขียนซ้ำและยังตรวจทุกข้อความเอง",
			"offer":     "Content Blueprint รุ่นสาธิตภายใน ไม่มีราคา โปรโมชั่น หรือคำรับรองผลลัพธ์",
			"channels":  "Facebook post และ carousel",
			"cta":       "อ่านรายละเอียดและให้เจ้าของเพจตรวจร่าง",
			"competitorAdsNotes": "ตัวอย่างที่ผู้ใช้จดเอง: โฆษณาส่วนมากพูดว่าเร็ว แต่ไม่อธิบายขั้นตรวจหลักฐาน " +
				"ข้อมูลนี้ไม่ใช่ผลจากการ scrape และไม่มีรายชื่อบุคคล",
			"ownedAudienceAssets": "ผู้มีส่วนร่วมกับเพจของตนและผู้เข้าเว็บไซต์ที่มีสิทธิ์ใช้เท่านั้น " +
				"ยังไม่อนุมัติการอัปโหลดรายชื่อลูกค้า",
		},
		Evidence: nil,
	}

	seen := make(map[Provider]bool)
	for _, name := range strings.Split(rawProviders, ",") {
		provider := Provider(strings.ToLower(strings.TrimSpace(name)))
		if !provider.Valid() {
			t.Fatalf("unsupported live Growth provider %q", name)
		}
		if seen[provider] {
			continue
		}
		seen[provider] = true
		t.Run(string(provider), func(t *testing.T) {
			runner := &liveRecordingRunner{}
			service := NewWithRunner(runner)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			status := service.Status(ctx, provider, "")
			if !status.Available || !status.AuthenticationChecked || !status.Authenticated {
				t.Fatalf("provider is not installed and authenticated: %s", status.Message)
			}
			pack, err := service.GenerateGrowth(ctx, brief, Options{
				Provider: provider,
				Workflow: workflow,
				Timeout:  10 * time.Minute,
			})
			if err != nil {
				for index, failure := range runner.failures {
					t.Logf("failed Growth process %d: exit=%d stderr=%q", index+1, failure.exitCode, failure.stderr)
				}
				t.Fatalf("live Growth generation failed: %v", err)
			}
			if err := workbench.ValidatePack(brief.PlaybookID, brief.Evidence, pack); err != nil {
				t.Fatalf("provider returned an invalid Growth Pack: %v", err)
			}
			t.Logf("validated Growth Pack with %d blocks using %s workflow", len(pack.Blocks), workflow)
		})
	}
}
