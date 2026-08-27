# Content Blueprint — MVP Specification

## Product promise

A private desktop workspace that turns a content brief and evidence pack into a
structured Thai/English SEO article, validates the result, and exports clean
HTML, Markdown, or JSON. The app favors evidence and editorial quality over
word-count or “GEO hack” claims.

The Facebook workspace turns a page-admin brief into a validated nine-part
Content Pack through an already authenticated Claude Code or Codex CLI. It can
also exchange the same revision-bound contract through a local MCP server and a
Chrome/Brave Manifest V3 extension. The extension inserts plain text only into
the editor the user explicitly focused; it never scrapes Facebook or clicks
Post, Schedule, or Send.

The Growth Hub adds ten revision-bound playbooks for Facebook sales, Google SEO,
and cross-channel repurposing. Every AI result uses a finite `GrowthPack` block
contract, records whether a statement came from user input, supplied evidence,
an imported metric, or AI inference, and starts in `needs_review`. Lead cards,
commission estimates, UTM links, and experiment metrics are deterministic local
tools rather than model outputs.

Competitor research is limited to observations the user manually records from
public ads. The app may compare messages and propose a creative gap, but it must
not crawl Meta, collect followers/comments/profiles, identify another page's
customers, or turn data without permission into an advertising audience. A
public profile, Page, post, or comment is not consent to collect that person or
add them to an audience. Retargeting, Lookalike, and Advantage+ plans may use
first-party or otherwise authorized Page/Instagram/video engagement,
website/app activity, lead forms, and customer lists only with the page owner's
rights, lawful basis, notice/opt-out where applicable, and explicit approval.

AI Studio visualizes real `facebook:ai-stage` and `growth:ai-stage` Wails events
as Gather-like worker characters: Strategist, Copywriter/Producer, Reviewer,
and Browser Courier. Characters walk to their work location, sit while
`working`, and carry a folder when a completed stage hands work to the next
working stage. It does not simulate progress or expose prompts, account
metadata, output text, or local paths.

## Primary workflow

1. Create or open a local project.
2. Fill in keyword, audience, intent, objective, voice, language, and evidence.
3. Preview the exact prompt/data contract before spending API quota.
4. Generate through the Gemini Interactions API using JSON Schema.
5. Edit and preview the returned content.
6. Review deterministic quality checks and source coverage.
7. Save locally and export HTML, Markdown, or JSON.

## Growth Hub workflow

1. Choose one of the ten trusted playbooks. The frontend cannot supply its own
   system prompt or output schema.
2. Fill the playbook fields and optional Evidence Notes. Sensitive fields are
   excluded from browser draft persistence.
3. Choose Claude CLI, Codex CLI, or MCP. Direct CLI supports a single worker or
   an isolated Strategist → Producer → Reviewer team.
4. Generate against a contextual schema that narrows block kinds, evidence
   bases, and source IDs to the selected playbook and current Brief, then run
   the Go semantic validator before storing. A semantically invalid Producer or
   Reviewer output gets at most one repair call in that GrowthPack stage; a
   second failure ends the stage without saving.
5. Resolve open questions and risk flags, add a reviewer note, then approve or
   reject. Approval records a human decision; it does not publish or message.
6. Track user-created lead cards, integer-money commission estimates and owner
   confirmation, UTM links, and directional experiment observations locally.
   There is no Facebook sales import, automatic attribution, payout, invoicing,
   or reconciliation.

The Facebook Campaign playbook accepts optional notes manually copied from Meta
Ads Library and metadata about first-party audience assets. Its result includes
a creative-gap analysis and Audience Source Plan. It never accepts a scraped
competitor customer list as a valid source.

## Backend contract exposed through Wails

The Go façade in `app.go` should expose these methods. Wails may return Go
errors as rejected promises.

- `Bootstrap() (BootstrapData, error)`
- `BuildPrompt(brief ContentBrief) (PromptPreview, error)`
- `GenerateContent(request GenerationRequest) (GenerationResult, error)`
- `EvaluateContent(brief ContentBrief, content GeneratedContent) QualityReport`
- `SaveProject(project Project) (Project, error)`
- `LoadProject(id string) (Project, error)`
- `ListProjects() ([]ProjectSummary, error)`
- `DeleteProject(id string) error`
- `SaveSettings(settings ProviderSettings) error`
- `ExportProject(project Project, format string) (string, error)`
- `CheckForUpdates() (UpdateInfo, error)`
- `DownloadUpdate(version string) (UpdateInfo, error)`
- `LaunchDownloadedUpdate(version string) error`

The updater is fixed to the public `Useless007/content-blueprint` repository.
The frontend can submit only a newer stable version, never a URL or file path.
The backend accepts the exact Windows installer and checksum assets, downloads
to an application-owned temporary directory, verifies SHA-256, and records the
verified path in memory. It launches the visible installer only after a second
user confirmation and quits the app only after process start succeeds. There
is no silent install, downgrade, or automatic download.

Growth methods:

- `GrowthBootstrap() (GrowthBootstrapData, error)`
- `SyncGrowthBrief(brief GrowthBrief) (GrowthBriefReceipt, error)`
- `GenerateGrowthPack(request GrowthGenerationRequest) (GrowthPackSnapshot, error)`
- `GetLatestGrowthPack() (GrowthLatestResult, error)`
- `ReviewGrowthPack(briefRevision, status, reviewerNote string) (GrowthPackSnapshot, error)`
- `CancelGrowthGeneration(runID string) bool`
- `SaveGrowthLead(lead GrowthLead) (GrowthLead, error)`
- `DeleteGrowthLead(id string) error`
- `SaveGrowthExperiment(experiment GrowthExperiment) (GrowthExperimentView, error)`
- `DeleteGrowthExperiment(id string) error`
- `BuildGrowthUTM(request GrowthUTMRequest) (GrowthUTMResult, error)`

The Growth catalog is defined in `internal/workbench`; inputs are checked against
that catalog and prompts treat all user fields, evidence, strategies, and drafts
as untrusted data. CLI processes receive read-only isolated workspaces, strict
JSON schemas, bounded output, cancellation, and a minimal environment allowlist.

## Local companion and Extension contract

The companion exposes exactly seven MCP tools. Facebook uses
`get_facebook_brief`, `save_facebook_pack`, and
`get_latest_facebook_pack`. Growth uses `list_growth_playbooks`,
`get_growth_brief`, `save_growth_pack`, and `get_latest_growth_pack`.
`get_growth_brief` returns the exact `briefRevision`, trusted task contract, and
contextual schema. Saves use strict decoding, semantic validation, and stale
revision rejection. MCP accepts no caller-defined prompt, schema, or path and
cannot run AI, open a browser, scrape, publish, send messages, or upload an
audience.

The Manifest V3 Extension v0.3 fetches the latest typed Growth snapshot through
Native Messaging and renders finite blocks as safe text/table/code. It does not
create a Growth Brief or run Growth AI. `stale` and `rejected` packs remain
readable/copyable but cannot be inserted; `needs_review` requires explicit
checkbox confirmation, and `approved` may be inserted. Confirmation never
inserts by itself. All insertion is plain text into the most recently
user-focused Facebook editor, capped at 50,000 characters, and never clicks
Post.

## Shared data shape

- `EvidenceSource`: `id`, `title`, `url`, `notes`
- `GroundingSource`: `title`, `url`
- `ContentBrief`: `keyword`, `audience`, `intent`, `objective`, `brandVoice`,
  `language`, `additionalInstructions`, `evidence[]`
- `ProviderSettings`: `provider`, `model`, `useGrounding`, `baseUrl`
- `GenerationRequest`: `brief`, `settings`, `apiKey`
- `GeneratedContent`: `title`, `slug`, `metaTitle`, `metaDescription`,
  `summaryBox`, `mainContentHtml`, `keyTakeaways[]`, `faqData[]`
- `KeyTakeaway`: `statement`, `sourceIds[]`
- `FAQItem`: `question`, `answer`, `sourceIds[]`
- `QualityCheck`: `id`, `label`, `status` (`pass|warning|fail`), `message`
- `QualityReport`: `score`, `checks[]`, `wordCount`, `sourceCoverage`
- `PromptPreview`: `system`, `user`, `schemaJson`
- `Usage`: `inputTokens`, `outputTokens`, `thoughtTokens`, `totalTokens`
- `GenerationResult`: `content`, `groundingSources`, `quality`, `prompt`, `usage`, `model`
- `Project`: `id`, `name`, `brief`, `content`, `groundingSources`, `quality`, `settings`,
  `createdAt`, `updatedAt`
- `ProjectSummary`: `id`, `name`, `keyword`, `score`, `updatedAt`
- `BootstrapData`: `settings`, `projects`, `apiKeyFromEnvironment`

API keys are accepted for the current generation request only and must never be
written to project/settings files. If the field is empty, use
`GEMINI_API_KEY` from the process environment.

## Gemini integration

- Default model: `gemini-3.7-flash`.
- Endpoint: `POST https://generativelanguage.googleapis.com/v1beta/interactions`.
- Send the key in `x-goog-api-key`.
- Use `response_format` with `type: text`, `mime_type: application/json`, and
  an explicit JSON Schema.
- Do not send deprecated sampling parameters for Gemini 3.x.
- When grounding is enabled, include `google_search` and `url_context` tools.
- Collect only `url_citation` annotations attached to model-output text, normalize
  and de-duplicate valid HTTP(S) URLs, then persist/export them as grounding sources.
- Treat non-2xx status, non-completed interactions, missing model output, and
  invalid JSON as actionable errors. Never silently invent fallback content.

Evidence-backed factual claims in `mainContentHtml` use an exact inline marker
such as `<sup data-source-id="S1">[S1]</sup>`. Source coverage includes these body
markers as well as `sourceIds[]` on takeaways and FAQ items; unknown or unsafe IDs
must not count as coverage.

## UI direction

- Thai-first desktop productivity app using React + TypeScript.
- Flat, content-first visual system from
  `design-system/content-blueprint/MASTER.md`.
- Left project rail, central brief/editor workspace, right quality panel.
- Make the API key disclosure explicit: session-only, never persisted.
- Every form field has a visible label; all icon buttons have accessible names;
  keyboard focus remains visible; loading and failure states are local and clear.
- At narrow widths, stack panels with no horizontal scrolling.
- Preview generated HTML in a sandboxed iframe with a restrictive CSP.
- Keep Facebook, Growth Hub, and SEO as three top-level workspaces. The ten
  playbooks belong inside Growth Hub rather than becoming top-level tabs.
- React Joyride 3.2.0 provides exactly eight Thai missions: รู้จักพื้นที่ทำงาน,
  ทำ Facebook Content Pack, ศึกษาคู่แข่งโดยไม่ดูดฐานลูกค้า,
  ตามลีดและค่าคอม, ทำ UTM และบันทึกการทดลอง, วางแผน Google SEO,
  เขียนบทความ SEO ฉบับเต็ม, and ส่งงานผ่าน MCP แล้วใช้ใน Chrome/Brave.
  Missions navigate asynchronously to the relevant workspace/tab and implement
  saved progress, missing-target handling, focus trapping, disabled overlay and
  Escape dismissal while active, and reduced-motion behavior. Tour state stores
  only welcome/active/step/completed state, never brief, evidence, lead, or
  output data.

## Live Growth smoke test

Live checks are opt-in and spend the signed-in CLI account's quota:

```powershell
$env:CONTENT_BLUEPRINT_LIVE_GROWTH_PROVIDERS = "codex,claude"
$env:CONTENT_BLUEPRINT_LIVE_GROWTH_WORKFLOW = "single"
go test -count=1 -run TestLiveGrowthProviderSmoke -v .\internal\cliprovider

$env:CONTENT_BLUEPRINT_LIVE_GROWTH_PROVIDERS = "codex"
$env:CONTENT_BLUEPRINT_LIVE_GROWTH_WORKFLOW = "team"
go test -count=1 -run TestLiveGrowthProviderSmoke -v .\internal\cliprovider
```

Team mode normally makes three provider calls. Producer and Reviewer may each
make at most one additional semantic-repair call.

## Definition of done

- `go test ./...`, frontend typecheck/build, and `wails build` succeed.
- Prompt builder, quality evaluator, storage, and API response parsing have tests.
- The default screen is useful without a key: users can build/copy the prompt.
- README covers prerequisites, API-key handling, development, and production build.
- Facebook direct-CLI generation has strict schemas, cancellation, isolated worker
  sessions, revision checks, safe stage events, and no shell-constructed commands.
- The local companion exposes the seven Facebook/Growth tools above over MCP
  and retains exact-origin Native Messaging compatibility.
- Real installed Chrome and Brave pass isolated Extension smoke tests, including
  stale-result blocking and proof that insertion does not click Post.
- A per-user Windows installer packages Wails, companion, and Extension runtime,
  registers only HKCU Native Messaging keys, and preserves user data on uninstall.
- GitHub Releases update checks run automatically at most once every 24 hours
  and on manual request. Download and installer launch each require an explicit
  user action, and an installer with a mismatched SHA-256 never launches.
- The Growth catalog contains exactly ten stable IDs; validators reject unknown
  input fields, malformed block shapes, unsupported evidence references, stale
  revisions, fabricated imported metrics, and structured-data code that is not
  valid JSON.
- Lead commission uses satang and basis points, keeps estimates separate from
  owner-confirmed amounts, and never lets AI confirm a sale. It does not infer
  a sale, attribute Facebook revenue, pay commissions, or reconcile invoices.
- Competitor tooling offers manual Ads Library observations and lawful audience
  planning only. There is no page/customer scraping or automated audience upload.
