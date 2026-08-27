package cliprovider

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"ContentBlueprint/internal/facebookcompanion"
)

type fakeRunner struct {
	paths map[string]string
	run   func(context.Context, Command) (ProcessResult, error)
}

func (runner *fakeRunner) LookPath(file string) (string, error) {
	if path, exists := runner.paths[file]; exists {
		return path, nil
	}
	return "", os.ErrNotExist
}

func (runner *fakeRunner) Run(ctx context.Context, command Command) (ProcessResult, error) {
	if runner.run == nil {
		return ProcessResult{}, nil
	}
	return runner.run(ctx, command)
}

func TestGenerateCodexUsesArgvStdinSchemaAndStrictOutput(t *testing.T) {
	packJSON := mustJSON(t, validPack())
	var got Command
	runner := &fakeRunner{run: func(_ context.Context, command Command) (ProcessResult, error) {
		got = command
		outputPath := flagValue(t, command.Args, "--output-last-message")
		schemaPath := flagValue(t, command.Args, "--output-schema")
		schema, err := os.ReadFile(schemaPath)
		if err != nil {
			t.Fatalf("read schema: %v", err)
		}
		if string(schema) != ContentPackSchema() {
			t.Fatal("Codex did not receive the strict Content Pack schema")
		}
		if err := os.WriteFile(outputPath, packJSON, 0o600); err != nil {
			t.Fatalf("write mock output: %v", err)
		}
		return ProcessResult{}, nil
	}}

	service := NewWithRunner(runner)
	pack, err := service.Generate(context.Background(), validBrief(), Options{
		Provider:   ProviderCodex,
		Executable: "codex-test.exe",
		Model:      "gpt-5.6-terra",
		Timeout:    time.Second,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !reflect.DeepEqual(pack, validPack()) {
		t.Fatalf("unexpected pack: %#v", pack)
	}
	wantPrefix := []string{"--ask-for-approval", "never", "exec", "--strict-config", "--ephemeral", "--ignore-user-config", "--ignore-rules"}
	if len(got.Args) < len(wantPrefix) || !reflect.DeepEqual(got.Args[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("unexpected Codex argv prefix: %#v", got.Args)
	}
	if got.Args[len(got.Args)-1] != "-" {
		t.Fatalf("Codex prompt must be read from stdin: %#v", got.Args)
	}
	for _, feature := range codexDisabledFeatures {
		assertArgPair(t, got.Args, "--disable", feature)
	}
	for _, setting := range []string{`web_search="disabled"`, `shell_environment_policy.inherit="none"`, "include_environment_context=false"} {
		assertArgPair(t, got.Args, "--config", setting)
	}
	assertArgPair(t, got.Args, "--ask-for-approval", "never")
	if flagValue(t, got.Args, "--sandbox") != "read-only" {
		t.Fatal("Codex worker sandbox is not read-only")
	}
	if strings.Contains(strings.Join(got.Args, " "), validBrief().Topic) {
		t.Fatal("brief content leaked into process arguments")
	}
	if !strings.Contains(string(got.Stdin), validBrief().Topic) {
		t.Fatal("brief was not passed through stdin")
	}
	if !strings.Contains(string(got.Stdin), "untrusted reference data") {
		t.Fatal("prompt injection boundary is missing")
	}
	if got.Env == nil {
		t.Fatal("Codex worker inherited the complete parent environment")
	}
	for _, entry := range got.Env {
		if strings.HasPrefix(strings.ToUpper(entry), "OPENAI_API_KEY=") {
			t.Fatal("Codex worker received an ambient API key")
		}
	}
}

func TestGenerateClaudeUsesStructuredOutputAndNoTools(t *testing.T) {
	envelope := map[string]any{
		"type":              "result",
		"subtype":           "success",
		"is_error":          false,
		"structured_output": validPack(),
		"result":            "ignored metadata",
	}
	var got Command
	runner := &fakeRunner{run: func(_ context.Context, command Command) (ProcessResult, error) {
		got = command
		return ProcessResult{Stdout: mustJSON(t, envelope)}, nil
	}}
	service := NewWithRunner(runner)
	pack, err := service.Generate(context.Background(), validBrief(), Options{
		Provider:   ProviderClaude,
		Executable: "claude.exe",
		Model:      "sonnet",
		Timeout:    time.Second,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !reflect.DeepEqual(pack, validPack()) {
		t.Fatalf("unexpected pack: %#v", pack)
	}
	for _, pair := range [][2]string{
		{"--output-format", "json"},
		{"--json-schema", ContentPackSchema()},
		{"--tools", ""},
		{"--permission-mode", "dontAsk"},
		{"--model", "sonnet"},
	} {
		if flagValue(t, got.Args, pair[0]) != pair[1] {
			t.Fatalf("%s was not passed safely as a separate argv value", pair[0])
		}
	}
	if got.Env == nil {
		t.Fatal("Claude worker inherited the complete parent environment")
	}
	for _, entry := range got.Env {
		upper := strings.ToUpper(entry)
		if strings.HasPrefix(upper, "ANTHROPIC_API_KEY=") || strings.HasPrefix(upper, "CLAUDE_CODE_OAUTH_TOKEN=") {
			t.Fatal("Claude worker received an ambient API credential")
		}
	}
	assertHasArg(t, got.Args, "-p")
	assertHasArg(t, got.Args, "--safe-mode")
	assertHasArg(t, got.Args, "--no-session-persistence")
	assertHasArg(t, got.Args, "--no-chrome")
	if strings.Contains(strings.Join(got.Args, " "), validBrief().Topic) {
		t.Fatal("brief content leaked into process arguments")
	}
}

func TestGenerateRejectsUnknownOutputFields(t *testing.T) {
	data := strings.TrimSuffix(string(mustJSON(t, validPack())), "}") + `,"unexpected":"value"}`
	runner := &fakeRunner{run: func(_ context.Context, command Command) (ProcessResult, error) {
		if err := os.WriteFile(flagValue(t, command.Args, "--output-last-message"), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		return ProcessResult{}, nil
	}}
	_, err := NewWithRunner(runner).Generate(context.Background(), validBrief(), Options{
		Provider: ProviderCodex, Executable: "codex-test.exe", Timeout: time.Second,
	})
	if !IsCode(err, CodeInvalidReply) {
		t.Fatalf("want invalid_reply, got %v", err)
	}
}

func TestGenerateRejectsDomainInvalidStructuredOutput(t *testing.T) {
	pack := validPack()
	pack.Hooks = []string{"same", "same", "different"}
	envelope := map[string]any{"type": "result", "subtype": "success", "structured_output": pack}
	runner := &fakeRunner{run: func(_ context.Context, _ Command) (ProcessResult, error) {
		return ProcessResult{Stdout: mustJSON(t, envelope)}, nil
	}}
	_, err := NewWithRunner(runner).Generate(context.Background(), validBrief(), Options{
		Provider: ProviderClaude, Executable: "claude.exe", Timeout: time.Second,
	})
	if !IsCode(err, CodeInvalidReply) {
		t.Fatalf("want invalid_reply, got %v", err)
	}
}

func TestGenerateCancellationAndTimeout(t *testing.T) {
	runner := &fakeRunner{run: func(ctx context.Context, _ Command) (ProcessResult, error) {
		<-ctx.Done()
		return ProcessResult{}, ctx.Err()
	}}
	service := NewWithRunner(runner)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.Generate(cancelCtx, validBrief(), Options{
		Provider: ProviderClaude, Executable: "claude.exe", Timeout: time.Second,
	})
	if !IsCode(err, CodeCancelled) {
		t.Fatalf("want cancelled, got %v", err)
	}

	_, err = service.Generate(context.Background(), validBrief(), Options{
		Provider: ProviderClaude, Executable: "claude.exe", Timeout: time.Millisecond,
	})
	if !IsCode(err, CodeInvalidInput) {
		t.Fatalf("sub-second timeout should be rejected, got %v", err)
	}
}

func TestGenerateRedactsProcessDetails(t *testing.T) {
	runner := &fakeRunner{run: func(_ context.Context, _ Command) (ProcessResult, error) {
		return ProcessResult{Stderr: []byte("token=super-secret C:\\private\\brief.json")}, errors.New("topic: secret campaign")
	}}
	_, err := NewWithRunner(runner).Generate(context.Background(), validBrief(), Options{
		Provider: ProviderClaude, Executable: "claude.exe", Timeout: time.Second,
	})
	if !IsCode(err, CodeProcess) {
		t.Fatalf("want process_failed, got %v", err)
	}
	public := err.Error()
	for _, secret := range []string{"super-secret", "private", "secret campaign"} {
		if strings.Contains(public, secret) {
			t.Fatalf("public error leaked %q: %s", secret, public)
		}
	}
}

func TestStatusUsesResolvedExecutableAndVersionOnly(t *testing.T) {
	calls := 0
	runner := &fakeRunner{
		paths: map[string]string{"codex": "/safe/codex"},
		run: func(_ context.Context, command Command) (ProcessResult, error) {
			calls++
			switch calls {
			case 1:
				if !reflect.DeepEqual(command.Args, []string{"--version"}) {
					t.Fatalf("unexpected version args: %#v", command.Args)
				}
				return ProcessResult{Stdout: []byte("codex-cli 1.2.3\n")}, nil
			case 2:
				if !reflect.DeepEqual(command.Args, []string{"login", "status"}) {
					t.Fatalf("unexpected auth args: %#v", command.Args)
				}
				return ProcessResult{}, nil
			default:
				t.Fatalf("unexpected extra status call")
				return ProcessResult{}, nil
			}
		},
	}
	status := NewWithRunner(runner).Status(context.Background(), ProviderCodex, "")
	if !status.Available || !status.AuthenticationChecked || !status.Authenticated || status.Path != "/safe/codex" || status.Version != "codex-cli 1.2.3" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestClaudeStatusParsesAuthenticationWithoutExposingAccountData(t *testing.T) {
	calls := 0
	runner := &fakeRunner{run: func(_ context.Context, command Command) (ProcessResult, error) {
		calls++
		if calls == 1 {
			return ProcessResult{Stdout: []byte("2.1.247 (Claude Code)")}, nil
		}
		if !reflect.DeepEqual(command.Args[len(command.Args)-3:], []string{"auth", "status", "--json"}) {
			t.Fatalf("unexpected Claude auth args: %#v", command.Args)
		}
		return ProcessResult{Stdout: []byte(`{"loggedIn":true,"authMethod":"claude.ai","email":"private@example.com"}`)}, nil
	}}
	status := NewWithRunner(runner).Status(context.Background(), ProviderClaude, "claude.exe")
	if !status.Available || !status.Authenticated || status.Message != "" {
		t.Fatalf("unexpected status: %#v", status)
	}
	encoded := string(mustJSON(t, status))
	if strings.Contains(encoded, "private@example.com") || strings.Contains(encoded, "claude.ai") {
		t.Fatalf("status exposed account details: %s", encoded)
	}
}

func TestStatusKeepsCLIAvailableWhenAuthenticationIsMissing(t *testing.T) {
	calls := 0
	runner := &fakeRunner{run: func(_ context.Context, _ Command) (ProcessResult, error) {
		calls++
		if calls == 1 {
			return ProcessResult{Stdout: []byte("Claude Code")}, nil
		}
		return ProcessResult{Stdout: []byte(`{"loggedIn":false}`)}, nil
	}}
	status := NewWithRunner(runner).Status(context.Background(), ProviderClaude, "claude.exe")
	if !status.Available || !status.AuthenticationChecked || status.Authenticated || status.Message == "" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestResolveWindowsClaudeCmdUsesNodeWithoutShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows npm shim behavior")
	}
	root := t.TempDir()
	shim := filepath.Join(root, "claude.cmd")
	entry := filepath.Join(root, "node_modules", "@anthropic-ai", "claude-code", "cli.js")
	if err := os.MkdirAll(filepath.Dir(entry), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shim, []byte("not executed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, []byte("// entry"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{paths: map[string]string{"node.exe": `C:\\Program Files\\nodejs\\node.exe`}}
	got, err := resolveInvocation(runner, ProviderClaude, shim)
	if err != nil {
		t.Fatalf("resolveInvocation: %v", err)
	}
	if got.path != runner.paths["node.exe"] || !reflect.DeepEqual(got.prefixArgs, []string{entry}) {
		t.Fatalf("unexpected invocation: %#v", got)
	}
}

func TestResolveWindowsClaudeCmdPrefersPackagedNativeBinary(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows npm shim behavior")
	}
	root := t.TempDir()
	shim := filepath.Join(root, "claude.cmd")
	native := filepath.Join(root, "node_modules", "@anthropic-ai", "claude-code", "bin", "claude.exe")
	if err := os.MkdirAll(filepath.Dir(native), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shim, []byte("not executed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(native, []byte("native placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := resolveInvocation(&fakeRunner{}, ProviderClaude, shim)
	if err != nil {
		t.Fatalf("resolveInvocation: %v", err)
	}
	if got.path != native || len(got.prefixArgs) != 0 {
		t.Fatalf("unexpected invocation: %#v", got)
	}
}

func TestInvalidBriefAndModelDoNotStartProcess(t *testing.T) {
	called := false
	runner := &fakeRunner{run: func(_ context.Context, _ Command) (ProcessResult, error) {
		called = true
		return ProcessResult{}, nil
	}}
	service := NewWithRunner(runner)
	brief := validBrief()
	brief.Topic = ""
	_, err := service.Generate(context.Background(), brief, Options{Provider: ProviderCodex, Executable: "codex.exe"})
	if !IsCode(err, CodeInvalidInput) || called {
		t.Fatalf("invalid brief should stop before process: err=%v called=%v", err, called)
	}
	_, err = service.Generate(context.Background(), validBrief(), Options{
		Provider: ProviderCodex, Executable: "codex.exe", Model: "--dangerous",
	})
	if !IsCode(err, CodeInvalidInput) || called {
		t.Fatalf("invalid model should stop before process: err=%v called=%v", err, called)
	}
}

func TestLimitedBufferRetainsOnlyConfiguredBytes(t *testing.T) {
	buffer := newLimitedBuffer(4)
	written, err := buffer.Write([]byte("123456"))
	if err != nil || written != 6 {
		t.Fatalf("Write returned (%d, %v)", written, err)
	}
	if got := buffer.String(); got != "1234" || !buffer.overflow {
		t.Fatalf("unexpected limited buffer state: %q overflow=%v", got, buffer.overflow)
	}
	written, err = buffer.Write([]byte("789"))
	if err != nil || written != 3 || buffer.String() != "1234" {
		t.Fatalf("overflow write returned (%d, %v), content %q", written, err, buffer.String())
	}
}

func TestReadLimitedFileRejectsOversize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(path, []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readLimitedFile(path, 4); err == nil {
		t.Fatal("expected oversize file error")
	}
}

func TestTeamWorkflowRunsIsolatedWorkersInOrderAndMarksRunsForStaleEvents(t *testing.T) {
	var commands []Command
	currentRun := 0
	runner := &fakeRunner{run: func(_ context.Context, command Command) (ProcessResult, error) {
		commands = append(commands, command)
		input := string(command.Stdin)
		switch {
		case strings.Contains(input, "WORKER_ROLE: strategist"):
			currentRun++
			return claudeEnvelope(t, validStrategy("strategy-run-"+string(rune('0'+currentRun)))), nil
		case strings.Contains(input, "WORKER_ROLE: copywriter"):
			pack := validPack()
			pack.LongPost = "draft-run-" + string(rune('0'+currentRun))
			return claudeEnvelope(t, pack), nil
		case strings.Contains(input, "WORKER_ROLE: reviewer"):
			pack := validPack()
			pack.LongPost = "final-run-" + string(rune('0'+currentRun))
			return claudeEnvelope(t, pack), nil
		default:
			t.Fatalf("unknown worker prompt: %q", input)
			return ProcessResult{}, nil
		}
	}}
	service := NewWithRunner(runner)
	runIDs := []string{"run-one", "run-two"}
	service.runID = func() (string, error) {
		id := runIDs[0]
		runIDs = runIDs[1:]
		return id, nil
	}
	var progress []ProgressEvent
	options := Options{
		Provider: ProviderClaude, Executable: "claude.exe", Workflow: WorkflowTeam, Timeout: 2 * time.Second,
		Progress: func(event ProgressEvent) { progress = append(progress, event) },
	}
	first, err := service.Generate(context.Background(), validBrief(), options)
	if err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	second, err := service.Generate(context.Background(), validBrief(), options)
	if err != nil {
		t.Fatalf("second Generate: %v", err)
	}
	if first.LongPost != "final-run-1" || second.LongPost != "final-run-2" {
		t.Fatalf("reviewer output was not final: first=%q second=%q", first.LongPost, second.LongPost)
	}
	if len(commands) != 6 {
		t.Fatalf("want six isolated workers, got %d", len(commands))
	}
	wantRoles := []string{"strategist", "copywriter", "reviewer", "strategist", "copywriter", "reviewer"}
	for index, role := range wantRoles {
		if !strings.Contains(string(commands[index].Stdin), "WORKER_ROLE: "+role) {
			t.Fatalf("worker %d is not %s", index, role)
		}
		wantDir := []string{"01-strategist", "02-copywriter", "03-reviewer"}[index%3]
		if filepath.Base(commands[index].Dir) != wantDir {
			t.Fatalf("worker %d not isolated in %s: %q", index, wantDir, commands[index].Dir)
		}
		assertHasArg(t, commands[index].Args, "--safe-mode")
		if flagValue(t, commands[index].Args, "--tools") != "" {
			t.Fatalf("worker %d tools were not disabled", index)
		}
	}
	if strings.Contains(string(commands[0].Stdin), "draft-run-") || strings.Contains(string(commands[0].Stdin), "strategy-run-") {
		t.Fatal("strategist received downstream context")
	}
	if !strings.Contains(string(commands[1].Stdin), "strategy-run-1") || strings.Contains(string(commands[1].Stdin), "draft-run-") {
		t.Fatal("copywriter did not receive only brief and current strategy")
	}
	if !strings.Contains(string(commands[2].Stdin), "strategy-run-1") || !strings.Contains(string(commands[2].Stdin), "draft-run-1") {
		t.Fatal("reviewer did not receive current strategy and draft")
	}
	if strings.Contains(string(commands[4].Stdin), "strategy-run-1") || !strings.Contains(string(commands[4].Stdin), "strategy-run-2") {
		t.Fatal("second run reused stale strategy context")
	}
	assertTeamProgress(t, progress[:6], "run-one")
	assertTeamProgress(t, progress[6:], "run-two")
}

func TestTeamWorkflowCancellationStopsAtActiveStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	runner := &fakeRunner{run: func(runCtx context.Context, command Command) (ProcessResult, error) {
		calls++
		if calls == 1 {
			return claudeEnvelope(t, validStrategy("cancel-strategy")), nil
		}
		if calls != 2 || !strings.Contains(string(command.Stdin), "WORKER_ROLE: copywriter") {
			t.Fatalf("unexpected worker call %d", calls)
		}
		cancel()
		<-runCtx.Done()
		return ProcessResult{Stderr: []byte("C:\\private\\prompt.txt token=secret")}, errors.New("secret worker failure")
	}}
	var progress []ProgressEvent
	_, err := NewWithRunner(runner).Generate(ctx, validBrief(), Options{
		Provider: ProviderClaude, Executable: "claude.exe", Workflow: WorkflowTeam, Timeout: 2 * time.Second,
		Progress: func(event ProgressEvent) { progress = append(progress, event) },
	})
	if !IsCode(err, CodeCancelled) {
		t.Fatalf("want cancelled, got %v", err)
	}
	var typed *Error
	if !errors.As(err, &typed) || typed.Stage != StageCopywriter {
		t.Fatalf("want copywriter stage, got %#v", typed)
	}
	if calls != 2 {
		t.Fatalf("reviewer ran after cancellation; calls=%d", calls)
	}
	for _, leaked := range []string{"private", "secret", validBrief().Topic} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("stage error leaked %q: %v", leaked, err)
		}
	}
	want := []struct {
		stage  Stage
		status StageStatus
	}{
		{StageStrategist, StageStarted},
		{StageStrategist, StageCompleted},
		{StageCopywriter, StageStarted},
		{StageCopywriter, StageFailed},
	}
	if len(progress) != len(want) {
		t.Fatalf("unexpected progress: %#v", progress)
	}
	for index := range want {
		if progress[index].Stage != want[index].stage || progress[index].Status != want[index].status {
			t.Fatalf("progress[%d]=%#v", index, progress[index])
		}
	}
}

func TestTeamWorkflowRejectsInvalidStrategyBeforeCopywriter(t *testing.T) {
	invalid := map[string]any{
		"audienceInsight": "insight", "positioning": "positioning", "primaryPromise": "promise",
		"angles": []any{
			map[string]any{"name": "a", "hookApproach": "a", "rationale": "a"},
			map[string]any{"name": "b", "hookApproach": "b", "rationale": "b"},
			map[string]any{"name": "c", "hookApproach": "c", "rationale": "c"},
		},
		"narrativeFlow": []string{"one", "two", "three"}, "evidenceUse": []any{}, "complianceRisks": []any{},
		"unexpected": "must be rejected",
	}
	calls := 0
	runner := &fakeRunner{run: func(_ context.Context, _ Command) (ProcessResult, error) {
		calls++
		return claudeEnvelope(t, invalid), nil
	}}
	_, err := NewWithRunner(runner).Generate(context.Background(), validBrief(), Options{
		Provider: ProviderClaude, Executable: "claude.exe", Workflow: WorkflowTeam, Timeout: time.Second,
	})
	if !IsCode(err, CodeInvalidReply) {
		t.Fatalf("want invalid_reply, got %v", err)
	}
	var typed *Error
	if !errors.As(err, &typed) || typed.Stage != StageStrategist || calls != 1 {
		t.Fatalf("invalid strategy advanced workflow: error=%#v calls=%d", typed, calls)
	}
}

func TestTeamWorkflowUsesStrictSchemasWithCodexWithoutEnablingSearch(t *testing.T) {
	calls := 0
	runner := &fakeRunner{run: func(_ context.Context, command Command) (ProcessResult, error) {
		calls++
		schema, err := os.ReadFile(flagValue(t, command.Args, "--output-schema"))
		if err != nil {
			t.Fatal(err)
		}
		var output any
		if calls == 1 {
			if string(schema) != TeamStrategySchema() {
				t.Fatal("strategist did not receive Team Strategy schema")
			}
			output = validStrategy("codex")
		} else {
			if string(schema) != ContentPackSchema() {
				t.Fatalf("worker %d did not receive Content Pack schema", calls)
			}
			pack := validPack()
			if calls == 3 {
				pack.ShortPost = "reviewed by Codex"
			}
			output = pack
		}
		for _, forbidden := range []string{"--search", "danger-full-access", "--dangerously-bypass-approvals-and-sandbox"} {
			assertNoArg(t, command.Args, forbidden)
		}
		if flagValue(t, command.Args, "--sandbox") != "read-only" {
			t.Fatal("Codex worker sandbox is not read-only")
		}
		assertHasArg(t, command.Args, "--ephemeral")
		assertHasArg(t, command.Args, "--ignore-user-config")
		assertHasArg(t, command.Args, "--ignore-rules")
		for _, feature := range codexDisabledFeatures {
			assertArgPair(t, command.Args, "--disable", feature)
		}
		if err := os.WriteFile(flagValue(t, command.Args, "--output-last-message"), mustJSON(t, output), 0o600); err != nil {
			t.Fatal(err)
		}
		return ProcessResult{}, nil
	}}
	pack, err := NewWithRunner(runner).Generate(context.Background(), validBrief(), Options{
		Provider: ProviderCodex, Executable: "codex.exe", Workflow: WorkflowTeam, Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if calls != 3 || pack.ShortPost != "reviewed by Codex" {
		t.Fatalf("unexpected Codex team result: calls=%d pack=%#v", calls, pack)
	}
}

func TestWorkflowStagesAndDefaultCompatibility(t *testing.T) {
	single, err := WorkflowStages("")
	if err != nil || !reflect.DeepEqual(single, []Stage{StageGenerate}) {
		t.Fatalf("default stages: %#v, %v", single, err)
	}
	team, err := WorkflowStages(WorkflowTeam)
	if err != nil || !reflect.DeepEqual(team, []Stage{StageStrategist, StageCopywriter, StageReviewer}) {
		t.Fatalf("team stages: %#v, %v", team, err)
	}
	if _, err := WorkflowStages("unknown"); err == nil {
		t.Fatal("unknown workflow should fail")
	}
}

func claudeEnvelope(t *testing.T, structured any) ProcessResult {
	t.Helper()
	return ProcessResult{Stdout: mustJSON(t, map[string]any{
		"type": "result", "subtype": "success", "is_error": false, "structured_output": structured,
	})}
}

func validStrategy(marker string) TeamStrategy {
	return TeamStrategy{
		AudienceInsight: "เจ้าของร้านต้องการแผนที่นำไปใช้ได้ " + marker,
		Positioning:     "ความรู้ที่ชัดเจนและตรวจสอบได้",
		PrimaryPromise:  "ช่วยจัดระบบความคิดโดยไม่รับประกันผลลัพธ์",
		Angles: []StrategyAngle{
			{Name: "ปัญหา", HookApproach: "เริ่มจากงานเดา", Rationale: "ตรงกับ pain point"},
			{Name: "หลักคิด", HookApproach: "เริ่มจากลูกค้าที่ใช่", Rationale: "ให้กรอบตัดสินใจ"},
			{Name: "ลงมือ", HookApproach: "เริ่มจากหนึ่งข้อความ", Rationale: "ทำตามได้ง่าย"},
		},
		NarrativeFlow:   []string{"ระบุปัญหา", "อธิบายหลักคิด", "เสนอขั้นต่อไป"},
		EvidenceUse:     []EvidenceUse{{SourceID: "course-outline", AllowedClaim: "หลักสูตรมี 8 บท"}},
		ComplianceRisks: []string{"อย่ารับประกันยอดขาย"},
	}
}

func assertTeamProgress(t *testing.T, events []ProgressEvent, runID string) {
	t.Helper()
	wantStages := []Stage{StageStrategist, StageStrategist, StageCopywriter, StageCopywriter, StageReviewer, StageReviewer}
	wantStatuses := []StageStatus{StageStarted, StageCompleted, StageStarted, StageCompleted, StageStarted, StageCompleted}
	if len(events) != len(wantStages) {
		t.Fatalf("run %s progress length=%d: %#v", runID, len(events), events)
	}
	for index := range events {
		if events[index].RunID != runID || events[index].Workflow != WorkflowTeam || events[index].Stage != wantStages[index] || events[index].Status != wantStatuses[index] || events[index].Index != index/2+1 || events[index].Total != 3 {
			t.Fatalf("run %s progress[%d]=%#v", runID, index, events[index])
		}
	}
}

func flagValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	for index := 0; index < len(args)-1; index++ {
		if args[index] == flag {
			return args[index+1]
		}
	}
	t.Fatalf("flag %s not found in %#v", flag, args)
	return ""
}

func assertHasArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, arg := range args {
		if arg == want {
			return
		}
	}
	t.Fatalf("argument %q not found in %#v", want, args)
}

func assertArgPair(t *testing.T, args []string, flag, value string) {
	t.Helper()
	for index := 0; index < len(args)-1; index++ {
		if args[index] == flag && args[index+1] == value {
			return
		}
	}
	t.Fatalf("argument pair %q %q not found in %#v", flag, value, args)
}

func assertNoArg(t *testing.T, args []string, forbidden string) {
	t.Helper()
	for _, arg := range args {
		if arg == forbidden {
			t.Fatalf("forbidden argument %q found in %#v", forbidden, args)
		}
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func validBrief() facebookcompanion.Brief {
	return facebookcompanion.Brief{
		Topic:      "คอร์สการตลาดสำหรับเจ้าของร้าน",
		Audience:   "เจ้าของร้านออนไลน์รายย่อย",
		Objective:  "ขอข้อมูลหลักสูตร",
		Offer:      "รับเอกสารแนะนำฟรี",
		BrandVoice: "อาจารย์ที่ชัดเจนและเป็นมิตร",
		Language:   "Thai",
		Evidence: []facebookcompanion.EvidenceSource{{
			ID: "course-outline", Title: "เอกสารหลักสูตร", Notes: "มีบทเรียน 8 บท",
		}},
	}
}

func validPack() facebookcompanion.ContentPack {
	return facebookcompanion.ContentPack{
		Hooks:      []string{"ยอดขายไม่ใช่จุดเริ่มต้น", "เริ่มจากลูกค้าที่ใช่", "แผนที่ชัดช่วยลดงานเดา"},
		LongPost:   "โพสต์ฉบับยาวที่มีสาระและตรวจสอบได้",
		ShortPost:  "โพสต์สั้นที่ชัดเจน",
		ReelScript: "เปิดเรื่อง อธิบาย และชวนขอข้อมูล",
		CarouselSlides: []facebookcompanion.CarouselSlide{
			{Headline: "ปัญหา", Body: "เริ่มจากปัญหาที่พบจริง"},
			{Headline: "หลักคิด", Body: "กำหนดกลุ่มเป้าหมายให้ชัด"},
			{Headline: "ขั้นต่อไป", Body: "ทดลองกับข้อความหนึ่งชุด"},
		},
		CTA:          "ส่งข้อความเพื่อขอเอกสารแนะนำ",
		FirstComment: "รายละเอียดเพิ่มเติมอยู่ในเอกสารแนะนำ",
		ReplyBank: []facebookcompanion.Reply{
			{Intent: "ถามราคา", Reply: "ขอทราบเป้าหมายก่อนเพื่อแนะนำตัวเลือกที่เหมาะสม"},
			{Intent: "ถามเวลา", Reply: "หลักสูตรแบ่งเป็น 8 บท เรียนตามเวลาที่สะดวก"},
			{Intent: "ขอรายละเอียด", Reply: "ยินดีส่งเอกสารหลักสูตรให้ตรวจสอบก่อน"},
		},
		ComplianceNotes: []string{},
	}
}
