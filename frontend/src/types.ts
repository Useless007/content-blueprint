export type CheckStatus = "pass" | "warning" | "fail";

export interface EvidenceSource {
  id: string;
  title: string;
  url: string;
  notes: string;
}

export interface ContentBrief {
  keyword: string;
  audience: string;
  intent: string;
  objective: string;
  brandVoice: string;
  language: string;
  additionalInstructions: string;
  evidence: EvidenceSource[];
}

export interface ProviderSettings {
  provider: string;
  model: string;
  useGrounding: boolean;
  baseUrl: string;
}

export interface GroundingSource {
  title: string;
  url: string;
}

export interface KeyTakeaway {
  statement: string;
  sourceIds: string[];
}

export interface FAQItem {
  question: string;
  answer: string;
  sourceIds: string[];
}

export interface GeneratedContent {
  title: string;
  slug: string;
  metaTitle: string;
  metaDescription: string;
  summaryBox: string;
  mainContentHtml: string;
  keyTakeaways: KeyTakeaway[];
  faqData: FAQItem[];
}

export interface QualityCheck {
  id: string;
  label: string;
  status: CheckStatus;
  message: string;
}

export interface QualityReport {
  score: number;
  checks: QualityCheck[];
  wordCount: number;
  sourceCoverage: number;
}

export interface PromptPreview {
  system: string;
  user: string;
  schemaJson: string;
}

export interface Usage {
  inputTokens: number;
  outputTokens: number;
  thoughtTokens: number;
  totalTokens: number;
}

export interface GenerationRequest {
  brief: ContentBrief;
  settings: ProviderSettings;
  apiKey: string;
}

export interface GenerationResult {
  content: GeneratedContent;
  quality: QualityReport;
  prompt: PromptPreview;
  usage: Usage;
  model: string;
  groundingSources?: GroundingSource[];
}

export interface Project {
  id: string;
  name: string;
  brief: ContentBrief;
  content?: GeneratedContent | null;
  quality?: QualityReport | null;
  groundingSources: GroundingSource[];
  settings: ProviderSettings;
  createdAt: string;
  updatedAt: string;
}

export interface ProjectSummary {
  id: string;
  name: string;
  keyword: string;
  score: number;
  updatedAt: string;
}

export interface BootstrapData {
  settings: ProviderSettings;
  projects: ProjectSummary[];
  apiKeyFromEnvironment: boolean;
}

export type FacebookProviderID = "claude" | "codex" | "mcp";

export interface FacebookProviderStatus {
  id: FacebookProviderID;
  label: string;
  available: boolean;
  authenticationChecked?: boolean;
  authenticated?: boolean;
  version: string;
  message: string;
}

export type FacebookWorkflow = "single" | "team";

export interface FacebookBrief {
  topic: string;
  audience: string;
  objective: string;
  offer: string;
  brandVoice: string;
  language: string;
  productDetails: string;
  evidence: EvidenceSource[];
  additionalInstructions: string;
}

export interface FacebookCarouselSlide {
  headline: string;
  body: string;
}

export interface FacebookReply {
  intent: string;
  reply: string;
}

export interface FacebookContentPack {
  hooks: string[];
  longPost: string;
  shortPost: string;
  reelScript: string;
  carouselSlides: FacebookCarouselSlide[];
  cta: string;
  firstComment: string;
  replyBank: FacebookReply[];
  complianceNotes: string[];
}

export interface FacebookPackSnapshot {
  version: number;
  briefRevision: string;
  pack: FacebookContentPack;
  groundingSources: GroundingSource[];
  generatedBy: string;
  updatedAt: string;
}

export interface FacebookBootstrapData {
  providers: FacebookProviderStatus[];
  latest?: FacebookPackSnapshot | null;
}

export interface FacebookGenerateRequest {
  runId: string;
  provider: FacebookProviderID;
  workflow: FacebookWorkflow;
  brief: FacebookBrief;
}

export interface FacebookStageUpdate {
  runId: string;
  stage: "strategist" | "copywriter" | "reviewer" | "browserCourier";
  status: "idle" | "queued" | "working" | "done" | "error";
  provider: FacebookProviderID;
  workflow: FacebookWorkflow;
  message: string;
  occurredAt: string;
}

export interface FacebookSyncResult {
  briefRevision: string;
  updatedAt: string;
}

export interface FacebookLatestResult {
  found: boolean;
  stale: boolean;
  snapshot?: FacebookPackSnapshot;
}

export type GrowthProviderID = "claude" | "codex" | "mcp";
export type GrowthWorkflow = "single" | "team";
export type GrowthPlaybookCategory =
  "sales" | "facebook" | "seo" | "cross-channel";
export type GrowthPlaybookID =
  | "offer-audience"
  | "facebook-campaign"
  | "sales-reply"
  | "seo-topic-map"
  | "seo-content-brief"
  | "seo-onpage-review"
  | "seo-internal-links"
  | "seo-structured-data"
  | "seo-search-console-opportunities"
  | "cross-channel-repurpose";

export interface GrowthPlaybookOption {
  value: string;
  label: string;
}
export interface GrowthPlaybookField {
  key: string;
  label: string;
  help: string;
  inputType: string;
  required: boolean;
  maxChars: number;
  placeholder: string;
  sensitive: boolean;
}

export interface GrowthPlaybookDefinition {
  id: GrowthPlaybookID;
  title: string;
  summary: string;
  outcome: string;
  category: GrowthPlaybookCategory;
  fields: GrowthPlaybookField[];
}

export interface GrowthBrief {
  playbookId: GrowthPlaybookID;
  language: string;
  brandVoice: string;
  inputs: Record<string, string>;
  evidence: EvidenceSource[];
}

export type GrowthBlockKind =
  "prose" | "cards" | "table" | "sequence" | "checklist" | "tasks" | "code";
export type GrowthBasis =
  | "user_input"
  | "supplied_evidence"
  | "ai_inference"
  | "imported_metric"
  | "mixed";
export interface GrowthBlock {
  id: string;
  title: string;
  kind: GrowthBlockKind;
  purpose: string;
  evidenceBasis: GrowthBasis;
  sourceIds: string[];
  body?: string;
  items?: Array<{ label: string; value: string; note: string }>;
  columns?: string[];
  rows?: string[][];
  code?: string;
}
export interface GrowthReviewCheck {
  label: string;
  status: "ready" | "review" | "blocked";
  reason: string;
}
export interface GrowthPack {
  title: string;
  summary: string;
  blocks: GrowthBlock[];
  openQuestions: string[];
  riskFlags: string[];
  reviewChecks: GrowthReviewCheck[];
}
export type GrowthReviewStatus = "needs_review" | "approved" | "rejected";
export interface GrowthPackSnapshot {
  version: number;
  briefRevision: string;
  playbookId: GrowthPlaybookID;
  evidenceSourceIds: string[];
  pack: GrowthPack;
  reviewStatus: GrowthReviewStatus;
  reviewerNote: string;
  reviewUpdatedAt?: string | null;
  generatedBy: string;
  updatedAt: string;
}
export interface GrowthLead {
  id: string;
  label: string;
  stage: "new" | "qualified" | "owner_review" | "won" | "lost";
  source: string;
  offer: string;
  needs: string;
  objections: string;
  assignee: string;
  nextFollowUp: string;
  handoffNote: string;
  campaignId: string;
  utm: string;
  saleAmountSatang: number;
  commissionRateBps: number;
  estimatedCommissionSatang: number;
  commissionConfirmed: boolean;
  confirmedCommissionSatang: number;
  confirmedBy: string;
  confirmedAt: string;
  createdAt: string;
  updatedAt: string;
}
export interface GrowthVariantMetrics {
  impressions: number;
  clicks: number;
  leads: number;
  sales: number;
  revenueSatang: number;
}
export interface GrowthExperiment {
  id: string;
  title: string;
  hypothesis: string;
  variable: string;
  variantA: string;
  variantB: string;
  startDate: string;
  endDate: string;
  audience: string;
  channel: string;
  primaryMetric: string;
  guardrail: string;
  comparable: boolean;
  metricsA: GrowthVariantMetrics;
  metricsB: GrowthVariantMetrics;
  learning: string;
  decision: string;
  approvedBy: string;
  createdAt: string;
  updatedAt: string;
}
export interface GrowthExperimentView {
  experiment: GrowthExperiment;
  ratesA: { ctr: number; leadRate: number; closeRate: number };
  ratesB: { ctr: number; leadRate: number; closeRate: number };
  analysisLabel: "directional" | "comparable";
  winner: "";
}
export interface GrowthBootstrapData {
  catalog: GrowthPlaybookDefinition[];
  providers: FacebookProviderStatus[];
  latest?: GrowthPackSnapshot | null;
  leads: GrowthLead[];
  experiments: GrowthExperimentView[];
  latestStale: boolean;
}
export interface GrowthGenerateRequest {
  runId: string;
  provider: Exclude<GrowthProviderID, "mcp">;
  workflow: GrowthWorkflow;
  brief: GrowthBrief;
}
export interface GrowthSyncResult {
  briefRevision: string;
  updatedAt: string;
}
export interface GrowthLatestResult {
  found: boolean;
  stale: boolean;
  snapshot?: GrowthPackSnapshot;
}
export interface GrowthReviewRequest {
  briefRevision: string;
  status: Exclude<GrowthReviewStatus, "needs_review">;
  reviewerNote: string;
}
export interface GrowthUTMRequest {
  destinationUrl: string;
  source: string;
  medium: string;
  campaign: string;
  term: string;
  content: string;
}
export interface GrowthUTMResult {
  url: string;
  campaignId: string;
}

export const createEmptyFacebookBrief = (): FacebookBrief => ({
  topic: "",
  audience: "",
  objective: "",
  offer: "",
  brandVoice: "ชัดเจน น่าเชื่อถือ เป็นธรรมชาติ",
  language: "th",
  productDetails: "",
  evidence: [],
  additionalInstructions: "",
});

export const DEFAULT_SETTINGS: ProviderSettings = {
  provider: "gemini",
  model: "gemini-3.7-flash",
  useGrounding: true,
  baseUrl: "https://generativelanguage.googleapis.com/v1beta/interactions",
};

export const createEmptyBrief = (): ContentBrief => ({
  keyword: "",
  audience: "",
  intent: "informational",
  objective: "",
  brandVoice: "ชัดเจน น่าเชื่อถือ และเป็นธรรมชาติ",
  language: "th",
  additionalInstructions: "",
  evidence: [],
});

export const createEmptyProject = (
  settings: ProviderSettings = DEFAULT_SETTINGS,
): Project => ({
  id: "",
  name: "โปรเจกต์ใหม่",
  brief: createEmptyBrief(),
  content: null,
  quality: null,
  groundingSources: [],
  settings: { ...settings },
  createdAt: "",
  updatedAt: "",
});
