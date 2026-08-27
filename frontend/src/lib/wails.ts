import type {
  BootstrapData,
  ContentBrief,
  FacebookBootstrapData,
  FacebookBrief,
  FacebookGenerateRequest,
  FacebookLatestResult,
  FacebookPackSnapshot,
  FacebookSyncResult,
  GrowthBootstrapData,
  GrowthBrief,
  GrowthExperiment,
  GrowthExperimentView,
  GrowthGenerateRequest,
  GrowthLatestResult,
  GrowthLead,
  GrowthPackSnapshot,
  GrowthReviewRequest,
  GrowthSyncResult,
  GrowthUTMRequest,
  GrowthUTMResult,
  GeneratedContent,
  GenerationRequest,
  GenerationResult,
  Project,
  ProjectSummary,
  PromptPreview,
  ProviderSettings,
  QualityReport,
} from "../types";
import type { UpdateInfo } from "../update/types";

type WailsMethod = (...args: unknown[]) => Promise<unknown>;

interface WailsWindow extends Window {
  go?: {
    main?: {
      App?: Record<string, WailsMethod>;
    };
  };
}

function getBridge(): Record<string, WailsMethod> {
  const app = (window as WailsWindow).go?.main?.App;
  if (!app) {
    throw new Error(
      "ไม่พบ Wails bridge — กรุณาเปิดแอปผ่าน wails dev หรือไฟล์โปรแกรมที่ build แล้ว",
    );
  }
  return app;
}

async function invoke<T>(method: string, ...args: unknown[]): Promise<T> {
  const bridge = getBridge();
  const fn = bridge[method];
  if (typeof fn !== "function") {
    throw new Error(
      `Wails binding ยังไม่มีเมธอด ${method} — กรุณารัน wails dev หรือ wails build ใหม่`,
    );
  }

  try {
    return (await fn.apply(bridge, args)) as T;
  } catch (error) {
    if (error instanceof Error) throw error;
    if (
      error &&
      typeof error === "object" &&
      "message" in error &&
      typeof error.message === "string"
    ) {
      throw new Error(error.message);
    }
    throw new Error(
      typeof error === "string" ? error : `เรียก ${method} ไม่สำเร็จ`,
    );
  }
}

export const wailsApi = {
  isAvailable: () => Boolean((window as WailsWindow).go?.main?.App),
  bootstrap: () => invoke<BootstrapData>("Bootstrap"),
  buildPrompt: (brief: ContentBrief) =>
    invoke<PromptPreview>("BuildPrompt", brief),
  generateContent: (request: GenerationRequest) =>
    invoke<GenerationResult>("GenerateContent", request),
  evaluateContent: (brief: ContentBrief, content: GeneratedContent) =>
    invoke<QualityReport>("EvaluateContent", brief, content),
  saveProject: (project: Project) => invoke<Project>("SaveProject", project),
  loadProject: (id: string) => invoke<Project>("LoadProject", id),
  listProjects: () => invoke<ProjectSummary[]>("ListProjects"),
  deleteProject: (id: string) => invoke<void>("DeleteProject", id),
  saveSettings: (settings: ProviderSettings) =>
    invoke<void>("SaveSettings", settings),
  exportProject: (project: Project, format: string) =>
    invoke<string>("ExportProject", project, format),
  facebookBootstrap: () => invoke<FacebookBootstrapData>("FacebookBootstrap"),
  generateFacebookPack: (request: FacebookGenerateRequest) =>
    invoke<FacebookPackSnapshot>("GenerateFacebookPack", request),
  syncFacebookBrief: (brief: FacebookBrief) =>
    invoke<FacebookSyncResult>("SyncFacebookBrief", brief),
  getLatestFacebookPack: () =>
    invoke<FacebookLatestResult>("GetLatestFacebookPack"),
  cancelFacebookGeneration: (runId: string) =>
    invoke<boolean>("CancelFacebookGeneration", runId),
  growthBootstrap: () => invoke<GrowthBootstrapData>("GrowthBootstrap"),
  generateGrowthPack: (request: GrowthGenerateRequest) =>
    invoke<GrowthPackSnapshot>("GenerateGrowthPack", request),
  syncGrowthBrief: (brief: GrowthBrief) =>
    invoke<GrowthSyncResult>("SyncGrowthBrief", brief),
  getLatestGrowthPack: () => invoke<GrowthLatestResult>("GetLatestGrowthPack"),
  reviewGrowthPack: (request: GrowthReviewRequest) =>
    invoke<GrowthPackSnapshot>(
      "ReviewGrowthPack",
      request.briefRevision,
      request.status,
      request.reviewerNote,
    ),
  cancelGrowthGeneration: (runId: string) =>
    invoke<boolean>("CancelGrowthGeneration", runId),
  saveGrowthLead: (lead: GrowthLead) =>
    invoke<GrowthLead>("SaveGrowthLead", lead),
  deleteGrowthLead: (id: string) => invoke<void>("DeleteGrowthLead", id),
  saveGrowthExperiment: (experiment: GrowthExperiment) =>
    invoke<GrowthExperimentView>("SaveGrowthExperiment", experiment),
  deleteGrowthExperiment: (id: string) =>
    invoke<void>("DeleteGrowthExperiment", id),
  buildGrowthUTM: (request: GrowthUTMRequest) =>
    invoke<GrowthUTMResult>("BuildGrowthUTM", request),
  checkForUpdates: () => invoke<UpdateInfo>("CheckForUpdates"),
  downloadUpdate: (version: string) =>
    invoke<UpdateInfo>("DownloadUpdate", version),
  launchDownloadedUpdate: (version: string) =>
    invoke<void>("LaunchDownloadedUpdate", version),
};
