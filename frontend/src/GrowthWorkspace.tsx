import {
  BadgeCheck,
  BookOpen,
  Clipboard,
  FileText,
  FlaskConical,
  Link2,
  Plus,
  Save,
  ShieldAlert,
  Sparkles,
  ThumbsDown,
  ThumbsUp,
  Trash2,
  Users,
  WalletCards,
  Zap,
} from "lucide-react";
import {
  type FormEvent,
  type ReactNode,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import "./GrowthWorkspace.css";
import { wailsApi } from "./lib/wails";
import type {
  EvidenceSource,
  FacebookProviderStatus,
  GrowthBlock,
  GrowthBrief,
  GrowthExperiment,
  GrowthExperimentView,
  GrowthLead,
  GrowthPackSnapshot,
  GrowthPlaybookDefinition,
  GrowthPlaybookID,
  GrowthProviderID,
  GrowthReviewStatus,
  GrowthUTMRequest,
  GrowthWorkflow,
} from "./types";
export type GrowthTab = "playbooks" | "leads" | "utm" | "experiments";
type Props = {
  aiStudio?: ReactNode;
  onRunStart?: (run: {
    runId: string;
    provider: Exclude<GrowthProviderID, "mcp">;
    workflow: GrowthWorkflow;
  }) => void;
  onRunFinish?: (run: {
    runId: string;
    workflow: GrowthWorkflow;
    error?: string;
  }) => void;
  initialTab?: GrowthTab;
  initialPlaybook?: string;
  onNavigate?: (tab: GrowthTab) => void;
  activeTab?: GrowthTab;
  onTabChange?: (tab: GrowthTab) => void;
  preferredPlaybookId?: string;
};
const field = (
  key: string,
  label: string,
  help: string,
  inputType = "textarea",
  required = true,
  maxChars = 12000,
  sensitive = false,
): GrowthPlaybookDefinition["fields"][number] => ({
  key,
  label,
  help,
  placeholder: "",
  inputType,
  required,
  maxChars,
  sensitive,
});
export const GROWTH_PLAYBOOK_FALLBACKS: GrowthPlaybookDefinition[] = [
  [
    "offer-audience",
    "sales",
    "Offer & Audience",
    "Clarify an evidence-backed offer and audience.",
    ["offer", "audience", "problems", "proof", "constraints"],
  ],
  [
    "facebook-campaign",
    "facebook",
    "Facebook Campaign",
    "Plan a human-reviewed Facebook content campaign.",
    [
      "objective",
      "audience",
      "offer",
      "channels",
      "cta",
      "competitorAdsNotes",
      "ownedAudienceAssets",
    ],
  ],
  [
    "sales-reply",
    "sales",
    "Sales Reply Copilot",
    "Draft replies from text deliberately pasted by the user.",
    ["message", "offerFacts", "intentHint", "handoffRules"],
  ],
  [
    "seo-topic-map",
    "seo",
    "SEO Topic Map",
    "Map supplied topics and questions to people-first pages.",
    ["seedTopics", "audience", "existingPages", "businessGoals"],
  ],
  [
    "seo-content-brief",
    "seo",
    "SEO Content Brief",
    "Create an evidence-aware brief for one useful page.",
    ["topic", "audience", "pageGoal", "knownFacts"],
  ],
  [
    "seo-onpage-review",
    "seo",
    "SEO On-page Review",
    "Review user-supplied page text without crawling.",
    ["pageUrl", "pageText", "pageGoal", "targetTopic"],
  ],
  [
    "seo-internal-links",
    "seo",
    "SEO Internal Links",
    "Propose internal-link tasks from a supplied page inventory.",
    ["pages", "focusPage", "constraints"],
  ],
  [
    "seo-structured-data",
    "seo",
    "SEO Structured Data",
    "Draft schema from facts visible in supplied page content.",
    ["pageType", "visibleContent", "canonicalUrl", "knownIdentifiers"],
  ],
  [
    "seo-search-console-opportunities",
    "seo",
    "Search Console Opportunities",
    "Interpret metrics explicitly imported by the user.",
    ["importedMetrics", "dateRange", "property", "goal"],
  ],
  [
    "cross-channel-repurpose",
    "cross-channel",
    "Cross-channel Repurpose",
    "Adapt one approved source asset across channels.",
    ["sourceAsset", "sourceFacts", "targetChannels", "cta"],
  ],
].map(([id, category, title, summary, keys]) => ({
  id: id as GrowthPlaybookID,
  category: category as GrowthPlaybookDefinition["category"],
  title: title as string,
  summary: summary as string,
  outcome:
    "Structured outputs, evidence labels, risks, and human review checks.",
  fields: (keys as string[]).map((k) =>
    field(
      k,
      k.replace(/([A-Z])/g, " $1"),
      k === "message"
        ? "Paste the minimum needed and redact personal data first."
        : "Use verified, user-supplied information.",
      k === "pageUrl" || k === "canonicalUrl"
        ? "url"
        : k === "objective" ||
            k === "cta" ||
            k === "channels" ||
            k === "dateRange" ||
            k === "property"
          ? "text"
          : "textarea",
      ![
        "proof",
        "constraints",
        "channels",
        "competitorAdsNotes",
        "ownedAudienceAssets",
        "intentHint",
        "handoffRules",
        "existingPages",
        "knownFacts",
        "pageUrl",
        "focusPage",
        "knownIdentifiers",
      ].includes(k),
      k === "pageText" || k === "importedMetrics" ? 50000 : 12000,
      [
        "message",
        "ownedAudienceAssets",
        "importedMetrics",
        "property",
      ].includes(k),
    ),
  ),
}));
const fallbackProviders: Record<GrowthProviderID, FacebookProviderStatus> = {
  claude: {
    id: "claude",
    label: "Claude CLI",
    available: false,
    version: "",
    message: "ไม่พร้อม",
  },
  codex: {
    id: "codex",
    label: "Codex CLI",
    available: false,
    version: "",
    message: "ไม่พร้อม",
  },
  mcp: {
    id: "mcp",
    label: "MCP Companion",
    available: true,
    version: "",
    message: "พร้อมซิงก์",
  },
};
const uid = () => crypto.randomUUID(),
  satang = (v: string) => Math.max(0, Math.round((+v || 0) * 100)),
  bps = (v: string) => Math.max(0, Math.round((+v || 0) * 100)),
  money = (v: number) =>
    new Intl.NumberFormat("th-TH", {
      style: "currency",
      currency: "THB",
    }).format(v / 100);
const redact = (v: string) =>
  v
    .replace(/[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}/g, "[REDACTED EMAIL]")
    .replace(/(?:\+?66|0)[\s().-]*\d(?:[\s().-]*\d){7,9}/g, "[REDACTED PHONE]");
function Field({
  label,
  help,
  wide,
  children,
}: {
  label: string;
  help?: string;
  wide?: boolean;
  children: ReactNode;
}) {
  return (
    <label className={`grw-field ${wide ? "grw-field-wide" : ""}`}>
      <span>{label}</span>
      {children}
      {help && <small>{help}</small>}
    </label>
  );
}
function BlockCard({
  block,
  copy,
}: {
  block: GrowthBlock;
  copy: (s: string) => void;
}) {
  const items = block.items ?? [];
  return (
    <article className="grw-output-card">
      <header>
        <div>
          <span className="grw-kind">{block.kind}</span>
          <h3>{block.title}</h3>
        </div>
        <button
          className="grw-icon-button"
          onClick={() =>
            copy(
              [
                block.body,
                block.code,
                ...items.map((x) => `${x.label}: ${x.value} ${x.note}`),
              ].join("\n"),
            )
          }
          aria-label="Copy block"
        >
          <Clipboard size={16} />
        </button>
      </header>
      <div className="grw-output-body">
        {block.purpose && <p>{block.purpose}</p>}
        {block.body && <p className="grw-prose">{block.body}</p>}
        {block.kind === "table" ? (
          <div className="grw-table-wrap">
            <table>
              <thead>
                <tr>
                  {block.columns?.map((x) => (
                    <th key={x}>{x}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {block.rows?.map((r, i) => (
                  <tr key={i}>
                    {r.map((x, j) => (
                      <td key={j}>{x}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          items.length > 0 && (
            <ul className="grw-card-list">
              {items.map((x, i) => (
                <li key={i}>
                  <strong>{x.label}</strong>: {x.value}
                  {x.note && <small>{x.note}</small>}
                </li>
              ))}
            </ul>
          )
        )}
        {block.code && (
          <pre className="grw-code">
            <code>{block.code}</code>
          </pre>
        )}
      </div>
      <footer>
        <span className="grw-basis">ที่มา: {block.evidenceBasis}</span>
        <span>{block.sourceIds?.join(", ")}</span>
      </footer>
    </article>
  );
}
export function GrowthWorkspace(p: Props) {
  const storedTab = localStorage.getItem("content-blueprint.growth-tab.v1");
  const restoredTab: GrowthTab = [
    "playbooks",
    "leads",
    "utm",
    "experiments",
  ].includes(storedTab ?? "")
    ? (storedTab as GrowthTab)
    : "playbooks";
  const restoredBrief = (() => {
    try {
      const value = JSON.parse(
        localStorage.getItem("content-blueprint.growth-draft.v2") ?? "null",
      ) as GrowthBrief | null;
      if (
        !value ||
        !GROWTH_PLAYBOOK_FALLBACKS.some((item) => item.id === value.playbookId)
      )
        return null;
      const definition = GROWTH_PLAYBOOK_FALLBACKS.find(
        (item) => item.id === value.playbookId,
      )!;
      const inputs = { ...(value.inputs ?? {}) };
      definition.fields
        .filter((item) => item.sensitive)
        .forEach((item) => delete inputs[item.key]);
      return {
        ...value,
        inputs,
        evidence: Array.isArray(value.evidence) ? value.evidence : [],
      };
    } catch {
      return null;
    }
  })();
  const bridge = wailsApi.isAvailable(),
    [innerTab, setInnerTab] = useState<GrowthTab>(p.initialTab ?? restoredTab),
    tab = p.activeTab ?? innerTab;
  const [catalog, setCatalog] = useState(GROWTH_PLAYBOOK_FALLBACKS),
    [filter, setFilter] = useState("facebook-sales"),
    [brief, setBrief] = useState<GrowthBrief>(
      restoredBrief ?? {
        playbookId: (p.preferredPlaybookId ??
          p.initialPlaybook ??
          "offer-audience") as GrowthPlaybookID,
        language: "th",
        brandVoice: "ชัดเจน เป็นธรรมชาติ",
        inputs: {},
        evidence: [],
      },
    ),
    [privateEvidence, setPrivateEvidence] = useState<Set<string>>(new Set()),
    [providers, setProviders] = useState(Object.values(fallbackProviders)),
    [provider, setProvider] = useState<GrowthProviderID>("mcp"),
    [workflow, setWorkflow] = useState<GrowthWorkflow>("team"),
    [snapshot, setSnapshot] = useState<GrowthPackSnapshot | null>(null),
    [stale, setStale] = useState(false),
    [reviewNote, setReviewNote] = useState(""),
    [busy, setBusy] = useState(""),
    [notice, setNotice] = useState("");
  const briefInitialized = useRef(false);
  const [leads, setLeads] = useState<GrowthLead[]>([]),
    [experiments, setExperiments] = useState<GrowthExperimentView[]>([]),
    [lead, setLead] = useState({
      label: "",
      stage: "new" as GrowthLead["stage"],
      source: "",
      note: "",
      sale: "",
      rate: "",
      confirmed: false,
      amount: "",
      by: "",
    }),
    [exp, setExp] = useState({
      title: "",
      hypothesis: "",
      variable: "",
      variantA: "",
      variantB: "",
      primaryMetric: "conversion",
      comparable: false,
      aImp: "",
      aClicks: "",
      aLeads: "",
      aSales: "",
      bImp: "",
      bClicks: "",
      bLeads: "",
      bSales: "",
    }),
    [utm, setUtm] = useState<GrowthUTMRequest>({
      destinationUrl: "",
      source: "facebook",
      medium: "social",
      campaign: "",
      term: "",
      content: "",
    }),
    [utmResult, setUtmResult] = useState("");
  const playbook = useMemo(
      () => catalog.find((x) => x.id === brief.playbookId) ?? catalog[0],
      [catalog, brief.playbookId],
    ),
    estimate = Math.round((satang(lead.sale) * bps(lead.rate)) / 10000);
  useEffect(() => {
    if (!bridge) return;
    wailsApi
      .growthBootstrap()
      .then((d) => {
        setCatalog(d.catalog.length ? d.catalog : GROWTH_PLAYBOOK_FALLBACKS);
        setProviders(d.providers);
        setSnapshot(d.latest ?? null);
        setStale(d.latestStale);
        setReviewNote(d.latest?.reviewerNote ?? "");
        setLeads(d.leads);
        setExperiments(d.experiments);
      })
      .catch((e) => setNotice(String(e)));
  }, [bridge]);
  useEffect(() => {
    if (
      p.preferredPlaybookId &&
      catalog.some((x) => x.id === p.preferredPlaybookId)
    )
      setBrief((b) =>
        b.playbookId === p.preferredPlaybookId
          ? b
          : {
              ...b,
              playbookId: p.preferredPlaybookId as GrowthPlaybookID,
              inputs: {},
            },
      );
  }, [p.preferredPlaybookId, catalog]);
  useEffect(() => {
    const safe = {
      ...brief,
      inputs: { ...brief.inputs },
      evidence: brief.evidence.filter((x) => !privateEvidence.has(x.id)),
    };
    playbook.fields
      .filter((x) => x.sensitive)
      .forEach((x) => delete safe.inputs[x.key]);
    localStorage.setItem(
      "content-blueprint.growth-draft.v2",
      JSON.stringify(safe),
    );
  }, [brief, playbook, privateEvidence]);
  useEffect(() => {
    if (!briefInitialized.current) {
      briefInitialized.current = true;
      return;
    }
    if (snapshot) setStale(true);
  }, [brief]);
  const nav = (v: GrowthTab) => {
      setInnerTab(v);
      localStorage.setItem("content-blueprint.growth-tab.v1", v);
      p.onTabChange?.(v);
      p.onNavigate?.(v);
    },
    copy = (s: string) =>
      navigator.clipboard.writeText(s).then(() => setNotice("คัดลอกแล้ว")),
    updateEvidence = (id: string, x: Partial<EvidenceSource>) =>
      setBrief((b) => ({
        ...b,
        evidence: b.evidence.map((e) => (e.id === id ? { ...e, ...x } : e)),
      }));
  const safeCopy = async (value: string) => {
    try {
      await copy(value);
    } catch (error) {
      setNotice(`Copy failed: ${String(error)}`);
    }
  };
  const validateBrief = () => {
    const missing = playbook.fields.filter(
      (item) => item.required && !brief.inputs[item.key]?.trim(),
    );
    if (missing.length)
      throw new Error(
        `Required: ${missing.map((item) => item.label).join(", ")}`,
      );
    for (const item of playbook.fields.filter(
      (value) => value.inputType === "url",
    )) {
      const raw = brief.inputs[item.key]?.trim();
      if (raw) {
        const parsed = new URL(raw);
        if (!["http:", "https:"].includes(parsed.protocol))
          throw new Error(`${item.label} must use http/https`);
      }
    }
    for (const evidence of brief.evidence) {
      if (!evidence.title.trim() || !evidence.notes.trim())
        throw new Error("Evidence notes require a title and verified facts");
      if (evidence.url) {
        const parsed = new URL(evidence.url);
        if (!["http:", "https:"].includes(parsed.protocol))
          throw new Error("Evidence URL must use http/https");
      }
    }
  };
  const run = async () => {
    try {
      validateBrief();
    } catch (error) {
      setNotice(String(error));
      return;
    }
    if (provider === "mcp") {
      setBusy("sync");
      try {
        await wailsApi.syncGrowthBrief(brief);
        setNotice("ซิงก์ Brief แล้ว");
      } catch (error) {
        setNotice(String(error));
      } finally {
        setBusy("");
      }
      return;
    }
    const id = uid();
    let runError: string | undefined;
    p.onRunStart?.({ runId: id, provider, workflow });
    setBusy("run");
    try {
      setSnapshot(
        await wailsApi.generateGrowthPack({
          runId: id,
          provider,
          workflow,
          brief,
        }),
      );
      setStale(false);
    } catch (e) {
      runError = String(e);
      setNotice(runError);
    } finally {
      setBusy("");
      p.onRunFinish?.({ runId: id, workflow, error: runError });
    }
  };
  const review = async (
    status: Exclude<GrowthReviewStatus, "needs_review">,
  ) => {
    if (!snapshot) return;
    if (!reviewNote.trim()) {
      setNotice("Reviewer note is required");
      return;
    }
    setBusy("review");
    try {
      setSnapshot(
        await wailsApi.reviewGrowthPack({
          briefRevision: snapshot.briefRevision,
          status,
          reviewerNote: reviewNote,
        }),
      );
    } catch (error) {
      setNotice(String(error));
    } finally {
      setBusy("");
    }
  };
  const saveLead = async (e: FormEvent) => {
    e.preventDefault();
    const item: GrowthLead = {
      id: "",
      label: lead.label,
      source: lead.source,
      offer: "",
      stage: lead.stage,
      needs: "",
      objections: "",
      assignee: "",
      nextFollowUp: "",
      handoffNote: lead.note,
      campaignId: "",
      utm: "",
      saleAmountSatang: satang(lead.sale),
      commissionRateBps: bps(lead.rate),
      estimatedCommissionSatang: estimate,
      commissionConfirmed: lead.confirmed,
      confirmedCommissionSatang: lead.confirmed ? satang(lead.amount) : 0,
      confirmedBy: lead.confirmed ? lead.by : "",
      confirmedAt: "",
      createdAt: "",
      updatedAt: "",
    };
    try {
      const saved = await wailsApi.saveGrowthLead(item);
      setLeads((x) => [...x, saved]);
      setLead({
        label: "",
        stage: "new",
        source: "",
        note: "",
        sale: "",
        rate: "",
        confirmed: false,
        amount: "",
        by: "",
      });
    } catch (err) {
      setNotice(String(err));
    }
  };
  const saveExp = async (e: FormEvent) => {
    e.preventDefault();
    const m = (v: "a" | "b") => ({
      impressions: +exp[`${v}Imp`] || 0,
      clicks: +exp[`${v}Clicks`] || 0,
      leads: +exp[`${v}Leads`] || 0,
      sales: +exp[`${v}Sales`] || 0,
      revenueSatang: 0,
    });
    const x: GrowthExperiment = {
      id: "",
      title: exp.title,
      hypothesis: exp.hypothesis,
      variable: exp.variable,
      variantA: exp.variantA,
      variantB: exp.variantB,
      startDate: "",
      endDate: "",
      audience: "",
      channel: "",
      primaryMetric: exp.primaryMetric,
      guardrail: "",
      comparable: exp.comparable,
      metricsA: m("a"),
      metricsB: m("b"),
      learning: "",
      decision: "",
      approvedBy: "",
      createdAt: "",
      updatedAt: "",
    };
    if (
      x.metricsA.clicks > x.metricsA.impressions ||
      x.metricsA.leads > x.metricsA.clicks ||
      x.metricsA.sales > x.metricsA.leads ||
      x.metricsB.clicks > x.metricsB.impressions ||
      x.metricsB.leads > x.metricsB.clicks ||
      x.metricsB.sales > x.metricsB.leads
    ) {
      setNotice("Funnel must be impressions >= clicks >= leads >= sales");
      return;
    }
    try {
      const saved = await wailsApi.saveGrowthExperiment(x);
      setExperiments((v) => [...v, saved]);
      setExp({
        title: "",
        hypothesis: "",
        variable: "",
        variantA: "",
        variantB: "",
        primaryMetric: "conversion",
        comparable: false,
        aImp: "",
        aClicks: "",
        aLeads: "",
        aSales: "",
        bImp: "",
        bClicks: "",
        bLeads: "",
        bSales: "",
      });
    } catch (err) {
      setNotice(String(err));
    }
  };
  const buildUtm = async (e: FormEvent) => {
    e.preventDefault();
    try {
      if (bridge) setUtmResult((await wailsApi.buildGrowthUTM(utm)).url);
      else {
        const u = new URL(utm.destinationUrl);
        (["source", "medium", "campaign", "term", "content"] as const).forEach(
          (k) =>
            utm[k] &&
            u.searchParams.set(
              `utm_${k}`,
              utm[k].trim().toLowerCase().replace(/\s+/g, "-"),
            ),
        );
        setUtmResult(u.toString());
      }
    } catch (err) {
      setNotice(String(err));
    }
  };
  const tabs: [GrowthTab, string, typeof BookOpen][] = [
    ["playbooks", "AI Playbooks", BookOpen],
    ["leads", "Leads & Commission", WalletCards],
    ["utm", "UTM Builder", Link2],
    ["experiments", "Experiment Log", FlaskConical],
  ];
  return (
    <div className="grw-shell" data-tour="growth-shell">
      <header className="grw-header">
        <div>
          <span className="grw-eyebrow">
            <Sparkles size={15} /> Growth Copilot
          </span>
          <h1>งานขายและ SEO ที่ตรวจทานได้</h1>
          <p>AI ช่วยร่าง มนุษย์ตรวจและอนุมัติก่อนใช้จริง</p>
        </div>
        <span className={`grw-bridge ${bridge ? "is-ready" : ""}`}>
          {bridge ? "Wails พร้อม" : "Preview only"}
        </span>
      </header>
      <nav className="grw-tabs" data-tour="growth-tabs">
        {tabs.map(([id, label, Icon]) => (
          <button
            key={id}
            className={tab === id ? "is-active" : ""}
            onClick={() => nav(id)}
          >
            <Icon size={18} />
            {label}
          </button>
        ))}
      </nav>
      {notice && (
        <div className="grw-notice" aria-live="polite">
          {notice}
        </div>
      )}
      <main className="grw-main">
        {tab === "playbooks" && (
          <section className="grw-playbook-layout">
            <aside className="grw-catalog" data-tour="growth-playbook-catalog">
              <header>
                <h2>Playbooks</h2>
                <span>{catalog.length}</span>
              </header>
              <div className="grw-filters">
                {["facebook-sales", "google-seo", "cross-channel"].map((c) => (
                  <button
                    key={c}
                    className={filter === c ? "is-active" : ""}
                    onClick={() => setFilter(c)}
                  >
                    {c}
                  </button>
                ))}
              </div>
              <div className="grw-playbook-list">
                {catalog
                  .filter(
                    (x) =>
                      (x.category === "seo"
                        ? "google-seo"
                        : x.category === "cross-channel"
                          ? "cross-channel"
                          : "facebook-sales") === filter,
                  )
                  .map((x) => (
                    <button
                      className={brief.playbookId === x.id ? "is-selected" : ""}
                      key={x.id}
                      onClick={() =>
                        setBrief((b) => ({ ...b, playbookId: x.id, inputs: {} }))
                      }
                    >
                      <strong>{x.title}</strong>
                      <small>{x.summary}</small>
                    </button>
                  ))}
              </div>
            </aside>
            <div className="grw-workspace grw-builder">
              <section className="grw-panel" data-tour="growth-playbook-form">
                <h2>{playbook.title}</h2>
                <p>{playbook.outcome}</p>
                {playbook.id === "facebook-campaign" && (
                  <div
                    className="grw-safety-callout grw-audience-safety"
                    data-tour="growth-audience-safety"
                  >
                    <ShieldAlert size={20} />
                    <p>
                      ใช้เฉพาะ first-party/consented data และ engagement ของ
                      Page, IG, video, website หรือ lead ที่คุณมีสิทธิ์ใช้
                      ไม่รองรับการดูดรายชื่อผู้ติดตาม คอมเมนต์
                      หรือลูกค้าของเพจอื่น
                    </p>
                  </div>
                )}
                {playbook.id === "sales-reply" && (
                  <div className="grw-safety-callout grw-sensitive-warning">
                    <ShieldAlert size={20} />
                    <div>
                      <strong>ลบข้อมูลส่วนบุคคลก่อนส่งให้ AI</strong>
                      <button
                        className="grw-button grw-button-secondary"
                        onClick={() =>
                          setBrief((b) => ({
                            ...b,
                            inputs: {
                              ...b.inputs,
                              message: redact(b.inputs.message ?? ""),
                            },
                          }))
                        }
                      >
                        Redact email / phone
                      </button>
                    </div>
                  </div>
                )}
                <div className="grw-form-grid">
                  {playbook.fields.map((x) => (
                    <Field
                      key={x.key}
                      label={x.label}
                      help={`${x.help} · สูงสุด ${x.maxChars} ตัวอักษร`}
                      wide={x.inputType === "textarea"}
                    >
                      {x.inputType === "textarea" ? (
                        <textarea
                          required={x.required}
                          maxLength={x.maxChars}
                          value={brief.inputs[x.key] ?? ""}
                          onChange={(e) =>
                            setBrief((b) => ({
                              ...b,
                              inputs: { ...b.inputs, [x.key]: e.target.value },
                            }))
                          }
                        />
                      ) : (
                        <input
                          type={x.inputType === "url" ? "url" : "text"}
                          required={x.required}
                          maxLength={x.maxChars}
                          value={brief.inputs[x.key] ?? ""}
                          onChange={(e) =>
                            setBrief((b) => ({
                              ...b,
                              inputs: { ...b.inputs, [x.key]: e.target.value },
                            }))
                          }
                        />
                      )}
                    </Field>
                  ))}
                </div>
              </section>
              <section className="grw-panel grw-evidence-list">
                <header className="grw-panel-heading">
                  <div>
                    <FileText size={20} />
                    <h2>Evidence Notes</h2>
                  </div>
                  <button
                    className="grw-button grw-button-secondary"
                    onClick={() =>
                      setBrief((b) => ({
                        ...b,
                        evidence: [
                          ...b.evidence,
                          { id: uid(), title: "", url: "", notes: "" },
                        ],
                      }))
                    }
                  >
                    <Plus size={17} />
                    เพิ่ม
                  </button>
                </header>
                {brief.evidence.map((x, i) => (
                  <fieldset key={x.id}>
                    <legend>Note {i + 1}</legend>
                    <div className="grw-form-grid">
                      <Field label="ชื่อ">
                        <input
                          value={x.title}
                          onChange={(e) =>
                            updateEvidence(x.id, { title: e.target.value })
                          }
                        />
                      </Field>
                      <Field label="URL">
                        <input
                          value={x.url}
                          onChange={(e) =>
                            updateEvidence(x.id, { url: e.target.value })
                          }
                        />
                      </Field>
                      <Field label="ข้อเท็จจริง" wide>
                        <textarea
                          value={x.notes}
                          onChange={(e) =>
                            updateEvidence(x.id, { notes: e.target.value })
                          }
                        />
                      </Field>
                    </div>
                    <div className="grw-evidence-actions">
                      <label>
                        <input
                          type="checkbox"
                          onChange={(e) =>
                            setPrivateEvidence((s) => {
                              const n = new Set(s);
                              e.target.checked ? n.add(x.id) : n.delete(x.id);
                              return n;
                            })
                          }
                        />
                        ไม่เก็บใน draft
                      </label>
                      <button
                        className="grw-icon-button is-danger"
                        onClick={() =>
                          setBrief((b) => ({
                            ...b,
                            evidence: b.evidence.filter((v) => v.id !== x.id),
                          }))
                        }
                      >
                        <Trash2 size={16} />
                      </button>
                    </div>
                  </fieldset>
                ))}
              </section>
              <section className="grw-panel" data-tour="growth-provider">
                <h2>AI provider & workflow</h2>
                <div className="grw-provider-grid">
                  {(["claude", "codex", "mcp"] as GrowthProviderID[]).map(
                    (id) => {
                      const x =
                        providers.find((v) => v.id === id) ??
                        fallbackProviders[id];
                      return (
                        <label
                          key={id}
                          className={provider === id ? "is-selected" : ""}
                        >
                          <input
                            type="radio"
                            checked={provider === id}
                            disabled={id !== "mcp" && !x.available}
                            onChange={() => setProvider(id)}
                          />
                          <strong>{x.label}</strong>
                          <span>{x.available ? "พร้อม" : x.message}</span>
                        </label>
                      );
                    },
                  )}
                </div>
                <fieldset
                  className="grw-workflows"
                  data-tour="growth-team-mode"
                >
                  <legend>รูปแบบการทำงาน</legend>
                  {(["team", "single"] as GrowthWorkflow[]).map((v) => (
                    <label
                      key={v}
                      className={workflow === v ? "is-selected" : ""}
                    >
                      <input
                        type="radio"
                        checked={workflow === v}
                        onChange={() => setWorkflow(v)}
                      />
                      {v === "team" ? <Users size={18} /> : <Zap size={18} />}
                      <strong>{v}</strong>
                    </label>
                  ))}
                </fieldset>
              </section>
              {p.aiStudio && (
                <section className="grw-ai-studio" data-tour="growth-ai-studio">
                  {p.aiStudio}
                </section>
              )}
              <div className="grw-actions" data-tour="growth-run">
                <button
                  className="grw-button grw-button-secondary"
                  disabled={!bridge || !!busy}
                  onClick={async () => {
                    try {
                      setBusy("fetch");
                      const r = await wailsApi.getLatestGrowthPack();
                      setSnapshot(r.snapshot ?? null);
                      setReviewNote(r.snapshot?.reviewerNote ?? "");
                      setStale(r.stale);
                    } catch (error) {
                      setNotice(String(error));
                    } finally {
                      setBusy("");
                    }
                  }}
                >
                  รับผลล่าสุด
                </button>
                <button
                  className="grw-button grw-button-primary"
                  disabled={!bridge || !!busy}
                  onClick={run}
                >
                  <Sparkles size={18} />
                  {provider === "mcp" ? "ซิงก์ Brief" : "สร้าง Growth Pack"}
                </button>
              </div>
            </div>
            <section
              className={`grw-results ${stale ? "is-stale" : ""}`}
              data-tour="growth-output"
            >
              <header>
                <h2>Growth Pack</h2>
                {snapshot && (
                  <div className="grw-badges">
                    <span className={`grw-badge is-${snapshot.reviewStatus}`}>
                      {snapshot.reviewStatus}
                    </span>
                    {stale && <span className="grw-badge is-stale">Stale</span>}
                  </div>
                )}
              </header>
              {!snapshot && (
                <div className="grw-results-empty">
                  <Sparkles size={28} />
                  <h3>No Growth Pack yet</h3>
                  <p>Complete the required fields, then sync or generate.</p>
                </div>
              )}
              {snapshot && (
                <>
                  <div className="grw-pack-intro">
                    <h3>{snapshot.pack.title}</h3>
                    <p>{snapshot.pack.summary}</p>
                  </div>
                  <div className="grw-output-grid">
                    {snapshot.pack.blocks.map((x) => (
                      <BlockCard key={x.id} block={x} copy={safeCopy} />
                    ))}
                  </div>
                  <div className="grw-review-lists">
                    <section>
                      <h3>Open questions</h3>
                      <ul>
                        {snapshot.pack.openQuestions.map((x, i) => (
                          <li key={i}>{x}</li>
                        ))}
                      </ul>
                    </section>
                    <section className="is-risk">
                      <h3>Risk flags</h3>
                      <ul>
                        {snapshot.pack.riskFlags.map((x, i) => (
                          <li key={i}>{x}</li>
                        ))}
                      </ul>
                    </section>
                  </div>
                  <section className="grw-checks">
                    {snapshot.pack.reviewChecks.map((x, i) => (
                      <article key={i}>
                        <span className={`grw-check-status is-${x.status}`}>
                          {x.status}
                        </span>
                        <div>
                          <strong>{x.label}</strong>
                          <p>{x.reason}</p>
                        </div>
                      </article>
                    ))}
                  </section>
                  <section
                    className="grw-human-review"
                    data-tour="growth-review"
                  >
                    <header>
                      <BadgeCheck size={20} />
                      <h3>Human Approval</h3>
                    </header>
                    <Field label="Reviewer note">
                      <textarea
                        required
                        maxLength={3000}
                        value={reviewNote}
                        onChange={(e) => setReviewNote(e.target.value)}
                      />
                    </Field>
                    <div>
                      <button
                        className="grw-button grw-button-danger"
                        disabled={
                          !bridge ||
                          stale ||
                          busy === "review" ||
                          !reviewNote.trim()
                        }
                        onClick={() => review("rejected")}
                      >
                        <ThumbsDown size={17} />
                        Reject
                      </button>
                      <button
                        className="grw-button grw-button-primary"
                        disabled={
                          !bridge ||
                          stale ||
                          busy === "review" ||
                          !reviewNote.trim()
                        }
                        onClick={() => review("approved")}
                      >
                        <ThumbsUp size={17} />
                        Approve
                      </button>
                    </div>
                  </section>
                </>
              )}
            </section>
          </section>
        )}
        {tab === "leads" && (
          <section className="grw-tool-page" data-tour="growth-leads">
            <header>
              <WalletCards size={24} />
              <h2>Leads & Commission</h2>
            </header>
            <div className="grw-tool-grid">
              <form className="grw-panel grw-tool-form" onSubmit={saveLead}>
                <div className="grw-form-grid">
                  <Field label="ชื่อ">
                    <input
                      required
                      maxLength={500}
                      value={lead.label}
                      onChange={(e) =>
                        setLead({ ...lead, label: e.target.value })
                      }
                    />
                  </Field>
                  <Field label="Source">
                    <input
                      maxLength={4000}
                      value={lead.source}
                      onChange={(e) =>
                        setLead({ ...lead, source: e.target.value })
                      }
                    />
                  </Field>
                  <Field label="Stage">
                    <select
                      value={lead.stage}
                      onChange={(e) =>
                        setLead({
                          ...lead,
                          stage: e.target.value as GrowthLead["stage"],
                        })
                      }
                    >
                      {["new", "qualified", "owner_review", "won", "lost"].map(
                        (x) => (
                          <option key={x}>{x}</option>
                        ),
                      )}
                    </select>
                  </Field>
                  <Field label="ยอดขาย (บาท)">
                    <input
                      type="number"
                      min="0"
                      step="0.01"
                      value={lead.sale}
                      onChange={(e) =>
                        setLead({ ...lead, sale: e.target.value })
                      }
                    />
                  </Field>
                  <Field label="ส่วนแบ่ง (%)">
                    <input
                      type="number"
                      min="0"
                      max="100"
                      step="0.01"
                      value={lead.rate}
                      onChange={(e) =>
                        setLead({ ...lead, rate: e.target.value })
                      }
                    />
                  </Field>
                  <Field label="Handoff note" wide>
                    <textarea
                      maxLength={4000}
                      value={lead.note}
                      onChange={(e) =>
                        setLead({ ...lead, note: e.target.value })
                      }
                    />
                  </Field>
                  <Field label="Estimated">
                    <output
                      className="grw-money is-estimate"
                      data-tour="growth-commission"
                    >
                      {money(estimate)}
                    </output>
                  </Field>
                </div>
                <label className="grw-confirm">
                  <input
                    type="checkbox"
                    checked={lead.confirmed}
                    onChange={(e) =>
                      setLead({ ...lead, confirmed: e.target.checked })
                    }
                  />
                  Owner-confirmed
                </label>
                {lead.confirmed && (
                  <div className="grw-form-grid">
                    <Field label="Confirmed amount">
                      <input
                        type="number"
                        min="0"
                        step="0.01"
                        value={lead.amount}
                        onChange={(e) =>
                          setLead({ ...lead, amount: e.target.value })
                        }
                      />
                    </Field>
                    <Field label="Confirmed by">
                      <input
                        required
                        maxLength={500}
                        value={lead.by}
                        onChange={(e) =>
                          setLead({ ...lead, by: e.target.value })
                        }
                      />
                    </Field>
                  </div>
                )}
                <button
                  className="grw-button grw-button-primary"
                  disabled={!bridge}
                >
                  <Save size={17} />
                  บันทึก
                </button>
              </form>
              <section className="grw-board">
                {(
                  [
                    "new",
                    "qualified",
                    "owner_review",
                    "won",
                    "lost",
                  ] as GrowthLead["stage"][]
                ).map((stage) => (
                  <section key={stage}>
                    <header>
                      <h3>{stage}</h3>
                    </header>
                    {leads
                      .filter((x) => x.stage === stage)
                      .map((x) => (
                        <article key={x.id}>
                          <div>
                            <strong>{x.label}</strong>
                            <button
                              type="button"
                              className="grw-icon-button is-danger"
                              aria-label={`Delete ${x.label}`}
                              onClick={async () => {
                                if (!window.confirm(`Delete ${x.label}?`))
                                  return;
                                try {
                                  await wailsApi.deleteGrowthLead(x.id);
                                  setLeads((items) =>
                                    items.filter((item) => item.id !== x.id),
                                  );
                                } catch (error) {
                                  setNotice(String(error));
                                }
                              }}
                            >
                              <Trash2 size={16} />
                            </button>
                          </div>
                          <dl>
                            <div>
                              <dt>Estimate</dt>
                              <dd>{money(x.estimatedCommissionSatang)}</dd>
                            </div>
                            <div>
                              <dt>Confirmed</dt>
                              <dd>
                                {x.commissionConfirmed
                                  ? money(x.confirmedCommissionSatang)
                                  : "ยังไม่ยืนยัน"}
                              </dd>
                            </div>
                          </dl>
                        </article>
                      ))}
                  </section>
                ))}
              </section>
            </div>
          </section>
        )}
        {tab === "utm" && (
          <section className="grw-tool-page" data-tour="growth-utm">
            <header>
              <Link2 size={24} />
              <h2>UTM Builder</h2>
            </header>
            <form className="grw-panel grw-utm-form" onSubmit={buildUtm}>
              <div className="grw-form-grid">
                {(Object.keys(utm) as (keyof GrowthUTMRequest)[]).map((k) => (
                  <Field
                    label={
                      (
                        {
                          destinationUrl: "Destination URL",
                          source: "UTM source",
                          medium: "UTM medium",
                          campaign: "UTM campaign",
                          term: "UTM term",
                          content: "UTM content",
                        } as Record<keyof GrowthUTMRequest, string>
                      )[k]
                    }
                    key={k}
                    wide={k === "destinationUrl"}
                  >
                    <input
                      type={k === "destinationUrl" ? "url" : "text"}
                      maxLength={k === "destinationUrl" ? 4096 : 200}
                      required={[
                        "destinationUrl",
                        "source",
                        "medium",
                        "campaign",
                      ].includes(k)}
                      value={utm[k]}
                      onChange={(e) => setUtm({ ...utm, [k]: e.target.value })}
                    />
                  </Field>
                ))}
              </div>
              <button className="grw-button grw-button-primary">
                สร้าง Preview
              </button>
              {utmResult && (
                <div className="grw-utm-result">
                  <code>{utmResult}</code>
                  <button type="button" onClick={() => safeCopy(utmResult)}>
                    Copy
                  </button>
                </div>
              )}
            </form>
          </section>
        )}
        {tab === "experiments" && (
          <section className="grw-tool-page" data-tour="growth-experiment">
            <header>
              <FlaskConical size={24} />
              <h2>Experiment Log</h2>
            </header>
            <div className="grw-tool-grid">
              <form className="grw-panel" onSubmit={saveExp}>
                <div className="grw-form-grid">
                  {(
                    [
                      "title",
                      "hypothesis",
                      "variable",
                      "variantA",
                      "variantB",
                      "primaryMetric",
                    ] as const
                  ).map((k) => (
                    <Field key={k} label={k}>
                      <input
                        required
                        value={exp[k]}
                        onChange={(e) =>
                          setExp({ ...exp, [k]: e.target.value })
                        }
                      />
                    </Field>
                  ))}
                </div>
                <label className="grw-confirm">
                  <input
                    type="checkbox"
                    checked={exp.comparable}
                    onChange={(e) =>
                      setExp({ ...exp, comparable: e.target.checked })
                    }
                  />
                  Comparable
                </label>
                <div className="grw-metric-grid">
                  {(["a", "b"] as const).map((v) => (
                    <div key={v}>
                      <strong>{v.toUpperCase()}</strong>
                      {(["Imp", "Clicks", "Leads", "Sales"] as const).map(
                        (m) => (
                          <input
                            key={m}
                            type="number"
                            min="0"
                            step="1"
                            aria-label={`Variant ${v.toUpperCase()} ${m === "Imp" ? "impressions" : m.toLowerCase()}`}
                            placeholder={m}
                            value={exp[`${v}${m}`]}
                            onChange={(e) =>
                              setExp({ ...exp, [`${v}${m}`]: e.target.value })
                            }
                          />
                        ),
                      )}
                    </div>
                  ))}
                </div>
                <button
                  className="grw-button grw-button-primary"
                  disabled={!bridge}
                >
                  <Save size={17} />
                  บันทึก
                </button>
              </form>
              <section className="grw-experiment-list">
                {experiments.map((v) => (
                  <article key={v.experiment.id}>
                    <header>
                      <div>
                        <span>{v.analysisLabel}</span>
                        <h3>{v.experiment.title}</h3>
                      </div>
                      <button
                        type="button"
                        className="grw-icon-button is-danger"
                        aria-label={`Delete ${v.experiment.title}`}
                        onClick={async () => {
                          if (!window.confirm(`Delete ${v.experiment.title}?`))
                            return;
                          try {
                            await wailsApi.deleteGrowthExperiment(
                              v.experiment.id,
                            );
                            setExperiments((items) =>
                              items.filter(
                                (item) =>
                                  item.experiment.id !== v.experiment.id,
                              ),
                            );
                          } catch (error) {
                            setNotice(String(error));
                          }
                        }}
                      >
                        <Trash2 size={16} />
                      </button>
                    </header>
                    <p>{v.experiment.hypothesis}</p>
                    <div className="grw-rate-grid">
                      <span>
                        A{" "}
                        <strong>{(v.ratesA.ctr * 100).toFixed(2)}% CTR</strong>
                      </span>
                      <span>
                        B{" "}
                        <strong>{(v.ratesB.ctr * 100).toFixed(2)}% CTR</strong>
                      </span>
                    </div>
                    <div className="grw-direction">
                      Directional only · ไม่ประกาศผู้ชนะ
                    </div>
                  </article>
                ))}
              </section>
            </div>
          </section>
        )}
      </main>
    </div>
  );
}
export default GrowthWorkspace;
