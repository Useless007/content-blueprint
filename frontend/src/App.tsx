import {
  AlertCircle,
  BookOpenText,
  Check,
  CheckCircle2,
  ChevronRight,
  CircleX,
  Clipboard,
  Code2,
  Download,
  Eye,
  ExternalLink,
  FilePlus2,
  FileText,
  KeyRound,
  Link2,
  LoaderCircle,
  Megaphone,
  PencilLine,
  Plus,
  Save,
  Search,
  Settings,
  ShieldCheck,
  Sparkles,
  Trash2,
  TriangleAlert,
  WandSparkles,
  X,
} from 'lucide-react'
import {
  type CSSProperties,
  type FormEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {EventsOn} from '../wailsjs/runtime/runtime'
import './App.css'
import {AIStudio, type AIActivityMessage, type AIConnectionStatus, type AIStageId, type AIStudioStages} from './AIStudio'
import {FacebookWorkspace} from './FacebookWorkspace'
import {GrowthWorkspace, type GrowthTab} from './GrowthWorkspace'
import {wailsApi} from './lib/wails'
import {OnboardingProvider} from './onboarding/OnboardingProvider'
import type {TourDestination} from './onboarding/missions'
import {UpdateCenter} from './update/UpdateCenter'
import {
  DEFAULT_SETTINGS,
  createEmptyBrief,
  createEmptyProject,
  type ContentBrief,
  type EvidenceSource,
  type FacebookProviderID,
  type FacebookStageUpdate,
  type FacebookWorkflow,
  type GeneratedContent,
  type GroundingSource,
  type Project,
  type ProjectSummary,
  type PromptPreview,
  type ProviderSettings,
  type QualityReport,
  type Usage,
} from './types'

type WorkspaceView = 'brief' | 'prompt' | 'editor'
type EditorView = 'fields' | 'html' | 'preview'
type AppMode = 'facebook' | 'growth' | 'seo'
type Notice = {tone: 'error' | 'success' | 'info'; message: string}
type BusyAction = 'bootstrap' | 'load' | 'prompt' | 'generate' | 'save' | 'export' | 'settings' | 'delete' | null
type BriefErrors = Record<string, string | undefined>

const EMPTY_CONTENT: GeneratedContent = {
  title: '',
  slug: '',
  metaTitle: '',
  metaDescription: '',
  summaryBox: '',
  mainContentHtml: '',
  keyTakeaways: [],
  faqData: [],
}

const EMPTY_AI_STAGES: AIStudioStages = {
  strategist: 'idle',
  copywriter: 'idle',
  reviewer: 'idle',
  browserCourier: 'idle',
}

const AI_STAGE_IDS: ReadonlySet<AIStageId> = new Set([
  'strategist',
  'copywriter',
  'reviewer',
  'browserCourier',
])

const AI_STAGE_STATUSES = new Set(['idle', 'queued', 'working', 'done', 'error'])

function initialAppMode(): AppMode {
  try {
    const saved = window.localStorage.getItem('content-blueprint:workspace')
    return saved === 'growth' || saved === 'seo' || saved === 'facebook' ? saved : 'growth'
  } catch {
    return 'growth'
  }
}

function SuiteModeSwitcher({mode, onChange}: {mode: AppMode; onChange: (mode: AppMode) => void}) {
  return (
    <nav className="suite-mode-switcher" aria-label="เลือกพื้นที่ทำงาน" data-tour="suite-mode-switcher">
      <button type="button" className={mode === 'facebook' ? 'is-active' : ''} aria-current={mode === 'facebook' ? 'page' : undefined} onClick={() => onChange('facebook')}>
        <Megaphone size={16} aria-hidden="true" />
        Facebook
      </button>
      <button type="button" className={mode === 'growth' ? 'is-active' : ''} aria-current={mode === 'growth' ? 'page' : undefined} onClick={() => onChange('growth')}>
        <WandSparkles size={16} aria-hidden="true" />
        Growth Hub
      </button>
      <button type="button" className={mode === 'seo' ? 'is-active' : ''} aria-current={mode === 'seo' ? 'page' : undefined} onClick={() => onChange('seo')}>
        <Search size={16} aria-hidden="true" />
        SEO Blueprint
      </button>
    </nav>
  )
}

function normalizeSettings(settings?: Partial<ProviderSettings> | null): ProviderSettings {
  return {...DEFAULT_SETTINGS, ...(settings ?? {})}
}

function normalizeProject(value: Partial<Project>): Project {
  const empty = createEmptyProject(normalizeSettings(value.settings))
  const brief = value.brief
    ? {
        ...createEmptyBrief(),
        ...value.brief,
        evidence: Array.isArray(value.brief.evidence) ? value.brief.evidence : [],
      }
    : empty.brief
  const content = value.content ? normalizeContent(value.content) : null

  return {
    ...empty,
    ...value,
    brief,
    content,
    quality: value.quality ?? null,
    groundingSources: Array.isArray(value.groundingSources) ? value.groundingSources : [],
    settings: normalizeSettings(value.settings),
  }
}

function validateBrief(brief: ContentBrief): BriefErrors {
  const errors: BriefErrors = {}
  if (!brief.keyword.trim()) errors['brief-keyword'] = 'ใส่คีย์เวิร์ดหลักก่อนสร้างพรอมป์'
  if (!brief.audience.trim()) errors['brief-audience'] = 'ระบุกลุ่มผู้อ่านที่ต้องการสื่อสาร'
  if (!brief.objective.trim()) errors['brief-objective'] = 'ระบุผลลัพธ์ที่บทความควรทำให้สำเร็จ'
  const sourceIds = brief.evidence.map((source) => source.id.trim())
  brief.evidence.forEach((source, index) => {
    const id = source.id.trim()
    if (!id) errors[`source-id-${index}`] = 'ต้องมีรหัสสำหรับเชื่อมข้อกล่าวอ้างกับแหล่งข้อมูล'
    else if (sourceIds.filter((candidate) => candidate === id).length > 1) errors[`source-id-${index}`] = `รหัส ${id} ซ้ำกับแหล่งอื่น`
    if (!source.title.trim()) errors[`source-title-${index}`] = 'ใส่ชื่อแหล่งข้อมูลเพื่อให้ตรวจสอบย้อนหลังได้'
    if (source.url.trim()) {
      try {
        const parsed = new URL(source.url.trim())
        if (!['http:', 'https:'].includes(parsed.protocol)) throw new Error('unsupported protocol')
      } catch {
        errors[`source-url-${index}`] = 'URL ต้องขึ้นต้นด้วย http:// หรือ https:// และมีรูปแบบถูกต้อง'
      }
    }
  })
  return errors
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return typeof error === 'string' ? error : 'เกิดข้อผิดพลาดที่ไม่ทราบสาเหตุ'
}

function formatUpdatedAt(value: string): string {
  if (!value) return 'ยังไม่บันทึก'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('th-TH', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function sourceCoveragePercent(value: number): number {
  return Math.max(0, Math.min(100, Math.round(value || 0)))
}

function safeExternalURL(value: string): string | null {
  try {
    const parsed = new URL(value)
    return ['http:', 'https:'].includes(parsed.protocol) ? parsed.toString() : null
  } catch {
    return null
  }
}

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;')
}

function sanitizePreviewHtml(value: string): string {
  const parsed = new DOMParser().parseFromString(value, 'text/html')
  parsed.querySelectorAll('script, iframe, object, embed, link, meta, base, form, input, button, textarea, select').forEach((node) => node.remove())
  parsed.body.querySelectorAll('*').forEach((node) => {
    for (const attribute of Array.from(node.attributes)) {
      const name = attribute.name.toLowerCase()
      if (name.startsWith('on') || ['href', 'src', 'srcset', 'action', 'formaction'].includes(name)) {
        node.removeAttribute(attribute.name)
      }
    }
  })
  return parsed.body.innerHTML
}

function normalizeContent(value: Partial<GeneratedContent>): GeneratedContent {
  return {
    ...EMPTY_CONTENT,
    ...value,
    keyTakeaways: Array.isArray(value.keyTakeaways)
      ? value.keyTakeaways.map((item) => ({...item, sourceIds: Array.isArray(item.sourceIds) ? item.sourceIds : []}))
      : [],
    faqData: Array.isArray(value.faqData)
      ? value.faqData.map((item) => ({...item, sourceIds: Array.isArray(item.sourceIds) ? item.sourceIds : []}))
      : [],
  }
}

function buildPreviewDocument(content: GeneratedContent): string {
  const takeaways = content.keyTakeaways
    .filter((item) => item.statement.trim())
    .map(
      (item) =>
        `<li>${escapeHtml(item.statement)}${item.sourceIds.length ? `<small>อ้างอิง: ${escapeHtml(item.sourceIds.join(', '))}</small>` : ''}</li>`,
    )
    .join('')
  const faq = content.faqData
    .filter((item) => item.question.trim() || item.answer.trim())
    .map(
      (item) =>
        `<section class="faq"><h3>${escapeHtml(item.question)}</h3><p>${escapeHtml(item.answer)}</p>${item.sourceIds.length ? `<small>อ้างอิง: ${escapeHtml(item.sourceIds.join(', '))}</small>` : ''}</section>`,
    )
    .join('')

  return `<!doctype html>
<html lang="th"><head><meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src data:; media-src 'none'; connect-src 'none'; font-src 'none'; object-src 'none'; frame-src 'none'; form-action 'none'; base-uri 'none'">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${escapeHtml(content.title || 'ตัวอย่างบทความ')}</title>
<style>
body{margin:0;padding:42px 44px;color:#1e293b;background:#fff;font:16px/1.75 Inter,system-ui,sans-serif}main{max-width:760px;margin:auto}h1,h2,h3{color:#134e4a;line-height:1.3}h1{font-size:2.15rem;letter-spacing:-.03em;margin:0 0 20px}.summary{padding:20px 22px;border-left:4px solid #0d9488;background:#f0fdfa;border-radius:0 10px 10px 0;margin:0 0 30px}.takeaways{padding:22px 26px;background:#f8fafc;border:1px solid #e2e8f0;border-radius:12px}.takeaways li{margin:9px 0}.faq{padding:18px 0;border-bottom:1px solid #e2e8f0}small{display:block;color:#64748b;margin-top:4px}a{color:#0f766e}img{max-width:100%;height:auto}pre{white-space:pre-wrap;overflow-wrap:anywhere;background:#f8fafc;padding:16px;border-radius:8px}blockquote{border-left:3px solid #99f6e4;margin-left:0;padding-left:18px;color:#475569}
</style></head><body><main>
<h1>${escapeHtml(content.title || 'ยังไม่มีชื่อบทความ')}</h1>
${content.summaryBox ? `<div class="summary">${escapeHtml(content.summaryBox)}</div>` : ''}
${takeaways ? `<section class="takeaways"><h2>ประเด็นสำคัญ</h2><ul>${takeaways}</ul></section>` : ''}
<article>${content.mainContentHtml ? sanitizePreviewHtml(content.mainContentHtml) : '<p>เริ่มแก้ไข HTML เพื่อดูตัวอย่างบทความที่นี่</p>'}</article>
${faq ? `<section><h2>คำถามที่พบบ่อย</h2>${faq}</section>` : ''}
</main></body></html>`
}

async function copyText(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  document.execCommand('copy')
  textarea.remove()
}

interface ProjectRailProps {
  projects: ProjectSummary[]
  activeId: string
  apiReady: boolean
  loadingId: string
  disabled: boolean
  onNew: () => void
  onOpen: (id: string) => void
  onDelete: (project: ProjectSummary) => void
  onSettings: () => void
}

function ProjectRail({
  projects,
  activeId,
  apiReady,
  loadingId,
  disabled,
  onNew,
  onOpen,
  onDelete,
  onSettings,
}: ProjectRailProps) {
  return (
    <aside className="project-rail" aria-label="รายการโปรเจกต์">
      <div className="brand-lockup">
        <div className="brand-mark" aria-hidden="true"><BookOpenText size={20} /></div>
        <div>
          <strong>Content Blueprint</strong>
          <span>Evidence-first workspace</span>
        </div>
      </div>

      <button className="button button-primary button-full" type="button" onClick={onNew} disabled={disabled}>
        <FilePlus2 size={17} aria-hidden="true" />
        โปรเจกต์ใหม่
      </button>

      <div className="rail-section-heading">
        <span>งานของฉัน</span>
        <span className="count-badge" aria-label={`${projects.length} โปรเจกต์`}>{projects.length}</span>
      </div>

      <div className="project-list">
        {projects.length === 0 ? (
          <div className="rail-empty">
            <FileText size={22} aria-hidden="true" />
            <p>โปรเจกต์ที่บันทึกจะอยู่ตรงนี้</p>
          </div>
        ) : projects.map((item) => {
          const active = item.id === activeId
          const loading = item.id === loadingId
          return (
            <div className={`project-item ${active ? 'is-active' : ''}`} key={item.id}>
              <button
                className="project-open"
                type="button"
                onClick={() => onOpen(item.id)}
                aria-current={active ? 'page' : undefined}
                disabled={disabled}
              >
                <span className="project-icon" aria-hidden="true">
                  {loading ? <LoaderCircle className="spin" size={17} /> : <FileText size={17} />}
                </span>
                <span className="project-copy">
                  <strong>{item.name || item.keyword || 'ไม่มีชื่อ'}</strong>
                  <small>{item.keyword || 'ยังไม่มีคีย์เวิร์ด'} · {formatUpdatedAt(item.updatedAt)}</small>
                </span>
                {item.score > 0 && <span className="mini-score" aria-label={`คะแนนคุณภาพ ${Math.round(item.score)}`}>{Math.round(item.score)}</span>}
              </button>
              <button
                className="icon-button project-delete"
                type="button"
                aria-label={`ลบโปรเจกต์ ${item.name}`}
                onClick={() => onDelete(item)}
                disabled={disabled}
              >
                <Trash2 size={15} aria-hidden="true" />
              </button>
            </div>
          )
        })}
      </div>

      <div className="rail-footer">
        <div className={`api-state ${apiReady ? 'is-ready' : ''}`}>
          <span className="api-dot" aria-hidden="true" />
          <span>{apiReady ? 'พร้อมสร้างเนื้อหา' : 'ยังไม่ได้ตั้ง API key'}</span>
        </div>
        <button className="button button-quiet button-full" type="button" onClick={onSettings} disabled={disabled}>
          <Settings size={17} aria-hidden="true" />
          API และโมเดล
        </button>
      </div>
    </aside>
  )
}

interface BriefWorkspaceProps {
  brief: ContentBrief
  errors: BriefErrors
  onChange: (brief: ContentBrief) => void
}

function BriefWorkspace({brief, errors, onChange}: BriefWorkspaceProps) {
  const update = <K extends keyof ContentBrief>(key: K, value: ContentBrief[K]) => {
    onChange({...brief, [key]: value})
  }

  const addEvidence = () => {
    let number = brief.evidence.length + 1
    while (brief.evidence.some((source) => source.id === `S${number}`)) number += 1
    update('evidence', [
      ...brief.evidence,
      {id: `S${number}`, title: '', url: '', notes: ''},
    ])
  }

  const updateEvidence = (index: number, patch: Partial<EvidenceSource>) => {
    update('evidence', brief.evidence.map((source, sourceIndex) =>
      sourceIndex === index ? {...source, ...patch} : source,
    ))
  }

  const removeEvidence = (index: number) => {
    update('evidence', brief.evidence.filter((_, sourceIndex) => sourceIndex !== index))
  }

  return (
    <div className="workspace-content brief-workspace" data-tour="seo-brief">
      <section className="section-card">
        <div className="section-heading">
          <div>
            <span className="eyebrow">Content brief</span>
            <h2>กำหนดทิศทางบทความ</h2>
            <p>ข้อมูลชุดนี้จะถูกส่งเข้า data contract โดยตรง ตรวจให้ชัดก่อนใช้โควตา</p>
          </div>
          <span className="section-step">01</span>
        </div>

        <div className="form-grid">
          <div className="field field-wide">
            <label htmlFor="brief-keyword">คีย์เวิร์ดหลัก <span aria-hidden="true">*</span></label>
            <input
              id="brief-keyword"
              className={errors['brief-keyword'] ? 'has-error' : ''}
              value={brief.keyword}
              onChange={(event) => update('keyword', event.target.value)}
              placeholder="เช่น วิธีวางแผนคอนเทนต์ SEO ด้วย AI"
              aria-describedby={errors['brief-keyword'] ? 'brief-keyword-error' : 'brief-keyword-help'}
              aria-invalid={Boolean(errors['brief-keyword'])}
            />
            {errors['brief-keyword'] ? <span id="brief-keyword-error" className="field-error"><AlertCircle size={14} aria-hidden="true" />{errors['brief-keyword']}</span> : <span id="brief-keyword-help" className="field-help">ใช้หนึ่งหัวข้อหลักที่สะท้อนสิ่งที่ผู้อ่านค้นหา</span>}
          </div>

          <div className="field">
            <label htmlFor="brief-audience">กลุ่มผู้อ่าน <span aria-hidden="true">*</span></label>
            <input
              id="brief-audience"
              className={errors['brief-audience'] ? 'has-error' : ''}
              value={brief.audience}
              onChange={(event) => update('audience', event.target.value)}
              placeholder="เช่น เจ้าของธุรกิจขนาดเล็ก"
              aria-describedby={errors['brief-audience'] ? 'brief-audience-error' : undefined}
              aria-invalid={Boolean(errors['brief-audience'])}
            />
            {errors['brief-audience'] && <span id="brief-audience-error" className="field-error"><AlertCircle size={14} aria-hidden="true" />{errors['brief-audience']}</span>}
          </div>

          <div className="field">
            <label htmlFor="brief-intent">เจตนาการค้นหา</label>
            <select id="brief-intent" value={brief.intent} onChange={(event) => update('intent', event.target.value)}>
              <option value="informational">หาข้อมูล (Informational)</option>
              <option value="commercial">เปรียบเทียบก่อนตัดสินใจ (Commercial)</option>
              <option value="transactional">พร้อมลงมือทำ (Transactional)</option>
              <option value="navigational">ค้นหาปลายทาง (Navigational)</option>
            </select>
          </div>

          <div className="field field-wide">
            <label htmlFor="brief-objective">เป้าหมายของเนื้อหา <span aria-hidden="true">*</span></label>
            <textarea
              id="brief-objective"
              className={errors['brief-objective'] ? 'has-error' : ''}
              rows={3}
              value={brief.objective}
              onChange={(event) => update('objective', event.target.value)}
              placeholder="หลังอ่านจบ ผู้อ่านควรรู้ ตัดสินใจ หรือทำอะไรได้"
              aria-describedby={errors['brief-objective'] ? 'brief-objective-error' : undefined}
              aria-invalid={Boolean(errors['brief-objective'])}
            />
            {errors['brief-objective'] && <span id="brief-objective-error" className="field-error"><AlertCircle size={14} aria-hidden="true" />{errors['brief-objective']}</span>}
          </div>

          <div className="field">
            <label htmlFor="brief-voice">น้ำเสียงแบรนด์</label>
            <input id="brief-voice" value={brief.brandVoice} onChange={(event) => update('brandVoice', event.target.value)} placeholder="ชัดเจน เป็นมิตร มีหลักฐาน" />
          </div>

          <div className="field">
            <label htmlFor="brief-language">ภาษาของบทความ</label>
            <select id="brief-language" value={brief.language} onChange={(event) => update('language', event.target.value)}>
              <option value="th">ภาษาไทย</option>
              <option value="en">English</option>
            </select>
          </div>

          <div className="field field-wide">
            <label htmlFor="brief-instructions">คำสั่งเพิ่มเติม</label>
            <textarea id="brief-instructions" rows={3} value={brief.additionalInstructions} onChange={(event) => update('additionalInstructions', event.target.value)} placeholder="ข้อจำกัด โครงสร้างที่ต้องการ หรือคำที่ไม่ควรใช้" />
            <span className="field-help">อย่าใส่ข้อเท็จจริงที่ยังไม่มีใน evidence pack</span>
          </div>
        </div>
      </section>

      <section className="section-card evidence-section">
        <div className="section-heading compact">
          <div>
            <span className="eyebrow">Evidence pack</span>
            <h2>หลักฐานและแหล่งอ้างอิง</h2>
            <p>เพิ่มข้อมูลที่โมเดลใช้ยืนยันข้อกล่าวอ้าง ยิ่งโน้ตเฉพาะเจาะจงยิ่งตรวจง่าย</p>
          </div>
          <button className="button button-secondary" type="button" onClick={addEvidence}>
            <Plus size={17} aria-hidden="true" />เพิ่มแหล่งข้อมูล
          </button>
        </div>

        {brief.evidence.length === 0 ? (
          <button className="evidence-empty" type="button" onClick={addEvidence}>
            <span className="empty-icon" aria-hidden="true"><Link2 size={24} /></span>
            <strong>เริ่มจากแหล่งข้อมูลแรก</strong>
            <span>ใส่ URL รายงาน เอกสารภายใน หรือโน้ตจากผู้เชี่ยวชาญ</span>
          </button>
        ) : (
          <div className="evidence-list">
            {brief.evidence.map((source, index) => (
              <article className="evidence-card" key={`evidence-${index}`}>
                <div className="evidence-card-head">
                  <span className="source-number">แหล่งที่ {index + 1}</span>
                  <button className="icon-button danger-hover" type="button" onClick={() => removeEvidence(index)} aria-label={`ลบแหล่งข้อมูลที่ ${index + 1}`}>
                    <Trash2 size={16} aria-hidden="true" />
                  </button>
                </div>
                <div className="form-grid evidence-grid">
                  <div className="field source-id-field">
                    <label htmlFor={`source-id-${index}`}>รหัสอ้างอิง</label>
                    <input id={`source-id-${index}`} className={errors[`source-id-${index}`] ? 'has-error' : ''} value={source.id} onChange={(event) => updateEvidence(index, {id: event.target.value})} placeholder={`S${index + 1}`} aria-invalid={Boolean(errors[`source-id-${index}`])} aria-describedby={errors[`source-id-${index}`] ? `source-id-${index}-error` : undefined} />
                    {errors[`source-id-${index}`] && <span id={`source-id-${index}-error`} className="field-error"><AlertCircle size={14} aria-hidden="true" />{errors[`source-id-${index}`]}</span>}
                  </div>
                  <div className="field evidence-title-field">
                    <label htmlFor={`source-title-${index}`}>ชื่อแหล่งข้อมูล</label>
                    <input id={`source-title-${index}`} className={errors[`source-title-${index}`] ? 'has-error' : ''} value={source.title} onChange={(event) => updateEvidence(index, {title: event.target.value})} placeholder="ชื่อรายงาน บทความ หรือบทสัมภาษณ์" aria-invalid={Boolean(errors[`source-title-${index}`])} aria-describedby={errors[`source-title-${index}`] ? `source-title-${index}-error` : undefined} />
                    {errors[`source-title-${index}`] && <span id={`source-title-${index}-error`} className="field-error"><AlertCircle size={14} aria-hidden="true" />{errors[`source-title-${index}`]}</span>}
                  </div>
                  <div className="field field-wide">
                    <label htmlFor={`source-url-${index}`}>URL <span className="label-optional">ถ้ามี</span></label>
                    <input id={`source-url-${index}`} className={errors[`source-url-${index}`] ? 'has-error' : ''} type="url" value={source.url} onChange={(event) => updateEvidence(index, {url: event.target.value})} placeholder="https://example.com/source" aria-invalid={Boolean(errors[`source-url-${index}`])} aria-describedby={errors[`source-url-${index}`] ? `source-url-${index}-error` : undefined} />
                    {errors[`source-url-${index}`] && <span id={`source-url-${index}-error`} className="field-error"><AlertCircle size={14} aria-hidden="true" />{errors[`source-url-${index}`]}</span>}
                  </div>
                  <div className="field field-wide">
                    <label htmlFor={`source-notes-${index}`}>ข้อเท็จจริงที่ใช้ได้</label>
                    <textarea id={`source-notes-${index}`} rows={4} value={source.notes} onChange={(event) => updateEvidence(index, {notes: event.target.value})} placeholder="สรุปตัวเลข ข้อค้นพบ ขอบเขตข้อมูล และข้อควรระวัง โดยไม่คัดลอกบทความทั้งชิ้น" />
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}

interface PromptWorkspaceProps {
  prompt: PromptPreview | null
  loading: boolean
  onBuild: () => void
  onCopy: (text: string, label: string) => void
}

function PromptWorkspace({prompt, loading, onBuild, onCopy}: PromptWorkspaceProps) {
  const [tab, setTab] = useState<'system' | 'user' | 'schemaJson'>('system')

  if (!prompt) {
    return (
      <div className="workspace-content centered-empty">
        <div className="large-empty-icon" aria-hidden="true"><Code2 size={30} /></div>
        <span className="eyebrow">Prompt contract</span>
        <h2>ตรวจพรอมป์ก่อนใช้โควตา</h2>
        <p>ระบบจะแสดง system prompt, user prompt และ JSON Schema ที่ส่งให้ Gemini แบบตรงไปตรงมา</p>
        <button className="button button-primary" type="button" onClick={onBuild} disabled={loading}>
          {loading ? <LoaderCircle className="spin" size={17} aria-hidden="true" /> : <WandSparkles size={17} aria-hidden="true" />}
          {loading ? 'กำลังประกอบพรอมป์…' : 'สร้างตัวอย่างพรอมป์'}
        </button>
      </div>
    )
  }

  const activeText = prompt[tab]
  const allText = `SYSTEM\n\n${prompt.system}\n\nUSER\n\n${prompt.user}\n\nJSON SCHEMA\n\n${prompt.schemaJson}`
  return (
    <div className="workspace-content prompt-workspace">
      <section className="section-card prompt-card">
        <div className="section-heading compact">
          <div>
            <span className="eyebrow">Prompt contract</span>
            <h2>ตรวจสิ่งที่จะส่งให้โมเดล</h2>
            <p>แก้ brief แล้วกดสร้างใหม่ได้เสมอ การดูหน้านี้ไม่ใช้ API key</p>
          </div>
          <div className="button-row">
            <button className="button button-secondary" type="button" onClick={onBuild} disabled={loading}>
              {loading ? <LoaderCircle className="spin" size={16} aria-hidden="true" /> : <WandSparkles size={16} aria-hidden="true" />}
              สร้างใหม่
            </button>
            <button className="button button-quiet" type="button" onClick={() => onCopy(allText, 'พรอมป์ทั้งหมด')}>
              <Clipboard size={16} aria-hidden="true" />คัดลอกทั้งหมด
            </button>
          </div>
        </div>

        <div className="tab-row" role="group" aria-label="ส่วนของพรอมป์">
          {([
            ['system', 'System'],
            ['user', 'User + brief'],
            ['schemaJson', 'JSON Schema'],
          ] as const).map(([value, label]) => (
            <button key={value} type="button" aria-pressed={tab === value} className={tab === value ? 'is-active' : ''} onClick={() => setTab(value)}>{label}</button>
          ))}
        </div>
        <div className="prompt-meta">
          <span>{activeText.length.toLocaleString('th-TH')} ตัวอักษร</span>
          <button className="text-button" type="button" onClick={() => onCopy(activeText, 'ส่วนนี้')}><Clipboard size={14} aria-hidden="true" />คัดลอกส่วนนี้</button>
        </div>
        <pre className="prompt-code" tabIndex={0}>{activeText}</pre>
      </section>
    </div>
  )
}

interface EditorWorkspaceProps {
  content: GeneratedContent | null | undefined
  view: EditorView
  onViewChange: (view: EditorView) => void
  onChange: (content: GeneratedContent) => void
}

function EditorWorkspace({content, view, onViewChange, onChange}: EditorWorkspaceProps) {
  const previewDocument = useMemo(() => content ? buildPreviewDocument(content) : '', [content])
  if (!content) {
    return (
      <div className="workspace-content centered-empty">
        <div className="large-empty-icon" aria-hidden="true"><PencilLine size={30} /></div>
        <span className="eyebrow">Article editor</span>
        <h2>พื้นที่แก้ไขยังว่างอยู่</h2>
        <p>สร้างเนื้อหาจาก brief ก่อน แล้วผลลัพธ์แบบมีโครงสร้างจะเปิดให้แก้ไขและพรีวิวที่นี่</p>
      </div>
    )
  }

  const update = <K extends keyof GeneratedContent>(key: K, value: GeneratedContent[K]) => {
    onChange({...content, [key]: value})
  }
  const parseSourceIds = (value: string) => value.split(',').map((item) => item.trim()).filter(Boolean)

  return (
    <div className="workspace-content editor-workspace">
      <section className="section-card editor-card">
        <div className="section-heading compact editor-heading">
          <div>
            <span className="eyebrow">Article editor</span>
            <h2>{content.title || 'บทความไม่มีชื่อ'}</h2>
            <p>การแก้ไขจะเรียกตัวตรวจคุณภาพใหม่โดยอัตโนมัติ</p>
          </div>
          <div className="tab-row compact-tabs" role="group" aria-label="มุมมองตัวแก้ไข">
            <button type="button" aria-pressed={view === 'fields'} className={view === 'fields' ? 'is-active' : ''} onClick={() => onViewChange('fields')}><PencilLine size={15} aria-hidden="true" />ข้อมูล</button>
            <button type="button" aria-pressed={view === 'html'} className={view === 'html' ? 'is-active' : ''} onClick={() => onViewChange('html')}><Code2 size={15} aria-hidden="true" />HTML</button>
            <button type="button" aria-pressed={view === 'preview'} className={view === 'preview' ? 'is-active' : ''} onClick={() => onViewChange('preview')}><Eye size={15} aria-hidden="true" />พรีวิว</button>
          </div>
        </div>

        {view === 'fields' && (
          <div className="editor-fields">
            <div className="form-grid">
              <div className="field field-wide">
                <label htmlFor="content-title">ชื่อบทความ</label>
                <input id="content-title" value={content.title} onChange={(event) => update('title', event.target.value)} />
              </div>
              <div className="field">
                <label htmlFor="content-slug">Slug</label>
                <input id="content-slug" value={content.slug} onChange={(event) => update('slug', event.target.value)} />
              </div>
              <div className="field">
                <label htmlFor="content-meta-title">Meta title <span className="character-count">{content.metaTitle.length}/60</span></label>
                <input id="content-meta-title" value={content.metaTitle} onChange={(event) => update('metaTitle', event.target.value)} />
              </div>
              <div className="field field-wide">
                <label htmlFor="content-meta-description">Meta description <span className="character-count">{content.metaDescription.length}/160</span></label>
                <textarea id="content-meta-description" rows={3} value={content.metaDescription} onChange={(event) => update('metaDescription', event.target.value)} />
              </div>
              <div className="field field-wide">
                <label htmlFor="content-summary">Summary box</label>
                <textarea id="content-summary" rows={4} value={content.summaryBox} onChange={(event) => update('summaryBox', event.target.value)} />
              </div>
            </div>

            <div className="repeater-section">
              <div className="repeater-heading">
                <div><h3>ประเด็นสำคัญ</h3><p>ระบุรหัสแหล่งอ้างอิงคั่นด้วยเครื่องหมายจุลภาค</p></div>
                <button className="button button-quiet" type="button" onClick={() => update('keyTakeaways', [...content.keyTakeaways, {statement: '', sourceIds: []}])}><Plus size={16} aria-hidden="true" />เพิ่มประเด็น</button>
              </div>
              {content.keyTakeaways.map((item, index) => (
                <div className="repeater-row" key={`takeaway-${index}`}>
                  <div className="field grow-field">
                    <label htmlFor={`takeaway-${index}`}>ประเด็นที่ {index + 1}</label>
                    <input id={`takeaway-${index}`} value={item.statement} onChange={(event) => update('keyTakeaways', content.keyTakeaways.map((current, itemIndex) => itemIndex === index ? {...current, statement: event.target.value} : current))} />
                  </div>
                  <div className="field source-list-field">
                    <label htmlFor={`takeaway-source-${index}`}>Source IDs</label>
                    <input id={`takeaway-source-${index}`} value={item.sourceIds.join(', ')} onChange={(event) => update('keyTakeaways', content.keyTakeaways.map((current, itemIndex) => itemIndex === index ? {...current, sourceIds: parseSourceIds(event.target.value)} : current))} />
                  </div>
                  <button className="icon-button danger-hover aligned-icon-button" type="button" aria-label={`ลบประเด็นที่ ${index + 1}`} onClick={() => update('keyTakeaways', content.keyTakeaways.filter((_, itemIndex) => itemIndex !== index))}><Trash2 size={16} aria-hidden="true" /></button>
                </div>
              ))}
            </div>

            <div className="repeater-section">
              <div className="repeater-heading">
                <div><h3>คำถามที่พบบ่อย</h3><p>ใช้เพื่อช่วยผู้อ่าน ไม่ใช่เพื่อรับประกัน rich result</p></div>
                <button className="button button-quiet" type="button" onClick={() => update('faqData', [...content.faqData, {question: '', answer: '', sourceIds: []}])}><Plus size={16} aria-hidden="true" />เพิ่มคำถาม</button>
              </div>
              {content.faqData.map((item, index) => (
                <div className="faq-editor" key={`faq-${index}`}>
                  <div className="faq-editor-head">
                    <strong>คำถามที่ {index + 1}</strong>
                    <button className="icon-button danger-hover" type="button" aria-label={`ลบคำถามที่ ${index + 1}`} onClick={() => update('faqData', content.faqData.filter((_, itemIndex) => itemIndex !== index))}><Trash2 size={16} aria-hidden="true" /></button>
                  </div>
                  <div className="field"><label htmlFor={`faq-question-${index}`}>คำถาม</label><input id={`faq-question-${index}`} value={item.question} onChange={(event) => update('faqData', content.faqData.map((current, itemIndex) => itemIndex === index ? {...current, question: event.target.value} : current))} /></div>
                  <div className="field"><label htmlFor={`faq-answer-${index}`}>คำตอบ</label><textarea id={`faq-answer-${index}`} rows={3} value={item.answer} onChange={(event) => update('faqData', content.faqData.map((current, itemIndex) => itemIndex === index ? {...current, answer: event.target.value} : current))} /></div>
                  <div className="field"><label htmlFor={`faq-source-${index}`}>Source IDs</label><input id={`faq-source-${index}`} value={item.sourceIds.join(', ')} onChange={(event) => update('faqData', content.faqData.map((current, itemIndex) => itemIndex === index ? {...current, sourceIds: parseSourceIds(event.target.value)} : current))} /></div>
                </div>
              ))}
            </div>
          </div>
        )}

        {view === 'html' && (
          <div className="html-editor-wrap">
            <label htmlFor="content-html">เนื้อหา HTML</label>
            <textarea id="content-html" className="html-editor" spellCheck={false} value={content.mainContentHtml} onChange={(event) => update('mainContentHtml', event.target.value)} />
            <p><ShieldCheck size={15} aria-hidden="true" />พรีวิวแยกอยู่ใน sandbox และบล็อกสคริปต์ เครือข่าย ฟอร์ม และไฟล์ภายนอก</p>
          </div>
        )}

        {view === 'preview' && (
          <div className="preview-frame-wrap">
            <div className="preview-toolbar"><span><Eye size={15} aria-hidden="true" />ตัวอย่างหน้าเผยแพร่</span><small>Sandboxed preview</small></div>
            <iframe className="preview-frame" title="ตัวอย่างบทความที่สร้าง" sandbox="" srcDoc={previewDocument} />
          </div>
        )}
      </section>
    </div>
  )
}

interface QualityPanelProps {
  brief: ContentBrief
  report: QualityReport | null | undefined
  usage: Usage | null
  model: string
  evaluating: boolean
  error: string
  groundingSources: GroundingSource[]
}

function QualityPanel({brief, report, usage, model, evaluating, error, groundingSources}: QualityPanelProps) {
  const completeFields = [brief.keyword, brief.audience, brief.objective].filter((value) => value.trim()).length
  const score = report ? Math.max(0, Math.min(100, Math.round(report.score || 0))) : 0
  const scoreStyle = {'--score': `${score}%`} as CSSProperties

  return (
    <aside className="quality-panel" aria-label="คุณภาพบทความ" data-tour="seo-quality">
      <div className="quality-heading">
        <div><span className="eyebrow">Quality desk</span><h2>ตรวจคุณภาพ</h2></div>
        {evaluating && <span className="evaluating"><LoaderCircle className="spin" size={14} aria-hidden="true" />กำลังตรวจ</span>}
      </div>

      {report ? (
        <>
          <div className="score-block">
            <div className="score-ring" style={scoreStyle} role="img" aria-label={`คะแนนคุณภาพ ${score} จาก 100`}><div><strong>{score}</strong><span>/100</span></div></div>
            <div><strong>{score >= 80 ? 'พร้อมตรวจรอบสุดท้าย' : score >= 60 ? 'ยังปรับให้คมขึ้นได้' : 'ควรแก้ก่อนเผยแพร่'}</strong><p>คะแนนจากกฎที่ตรวจซ้ำได้ ไม่ใช่คะแนนจัดอันดับของ Google</p></div>
          </div>
          <div className="metric-grid">
            <div><span>จำนวนคำ</span><strong>{(report.wordCount || 0).toLocaleString('th-TH')}</strong></div>
            <div><span>ครอบคลุมแหล่งข้อมูล</span><strong>{sourceCoveragePercent(report.sourceCoverage)}%</strong></div>
          </div>
          <div className="checks-list">
            {report.checks?.map((check) => {
              const Icon = check.status === 'pass' ? CheckCircle2 : check.status === 'warning' ? TriangleAlert : CircleX
              return (
                <div className={`quality-check status-${check.status}`} key={check.id}>
                  <Icon size={18} aria-hidden="true" />
                  <div><strong>{check.label}</strong><p>{check.message}</p></div>
                </div>
              )
            })}
          </div>
        </>
      ) : (
        <div className="quality-empty">
          <div className="brief-progress"><span style={{width: `${(completeFields / 3) * 100}%`}} /></div>
          <strong>Brief พร้อม {completeFields}/3 ส่วนหลัก</strong>
          <p>ผลตรวจบทความจะแสดงหลังสร้างเนื้อหา และอัปเดตเมื่อคุณแก้ไข</p>
          <ul>
            <li className={brief.keyword.trim() ? 'done' : ''}><Check size={14} aria-hidden="true" />คีย์เวิร์ดหลัก</li>
            <li className={brief.audience.trim() ? 'done' : ''}><Check size={14} aria-hidden="true" />กลุ่มผู้อ่าน</li>
            <li className={brief.objective.trim() ? 'done' : ''}><Check size={14} aria-hidden="true" />เป้าหมายเนื้อหา</li>
          </ul>
        </div>
      )}

      {error && <div className="panel-error"><AlertCircle size={16} aria-hidden="true" /><span>{error}</span></div>}

      <div className="source-summary">
        <div className="source-summary-head"><span>Evidence pack</span><strong>{brief.evidence.length}</strong></div>
        <p>{brief.evidence.length ? `${brief.evidence.filter((source) => source.notes.trim()).length} แหล่งมีโน้ตข้อเท็จจริง` : 'ยังไม่มีแหล่งข้อมูล แนวคิดสำคัญอาจตรวจยืนยันไม่ได้'}</p>
      </div>

      {groundingSources.length > 0 && (
        <div className="grounding-sources">
          <div className="source-summary-head"><span>แหล่งข้อมูลจาก Grounding</span><strong>{groundingSources.length}</strong></div>
          <ul>
            {groundingSources.map((source, index) => {
              const safeURL = safeExternalURL(source.url)
              if (!safeURL) return null
              return (
                <li key={`${safeURL}-${index}`}>
                  <a href={safeURL} target="_blank" rel="noopener noreferrer" title={safeURL}>
                    <span>{source.title || new URL(safeURL).hostname}</span>
                    <ExternalLink size={13} aria-hidden="true" />
                  </a>
                </li>
              )
            })}
          </ul>
        </div>
      )}

      {usage && (
        <div className="usage-card">
          <div className="source-summary-head"><span>การใช้โทเคน</span><strong>{usage.totalTokens.toLocaleString('th-TH')}</strong></div>
          <dl><div><dt>Input</dt><dd>{usage.inputTokens.toLocaleString('th-TH')}</dd></div><div><dt>Output</dt><dd>{usage.outputTokens.toLocaleString('th-TH')}</dd></div><div><dt>Thought</dt><dd>{usage.thoughtTokens.toLocaleString('th-TH')}</dd></div></dl>
          <small>{model}</small>
        </div>
      )}

      <div className="quality-note"><ShieldCheck size={16} aria-hidden="true" /><p>ตรวจข้อเท็จจริง ลิงก์ และภาษารอบสุดท้ายด้วยมนุษย์ก่อนเผยแพร่เสมอ</p></div>
    </aside>
  )
}

interface SettingsDialogProps {
  settings: ProviderSettings
  apiKey: string
  apiKeyFromEnvironment: boolean
  saving: boolean
  onApiKeyChange: (value: string) => void
  onClose: () => void
  onSave: (settings: ProviderSettings) => Promise<void>
}

function SettingsDialog({settings, apiKey, apiKeyFromEnvironment, saving, onApiKeyChange, onClose, onSave}: SettingsDialogProps) {
  const [draft, setDraft] = useState(settings)
  const [showKey, setShowKey] = useState(false)
  const [localError, setLocalError] = useState('')
  const dialogRef = useRef<HTMLDivElement>(null)
  const keyInputRef = useRef<HTMLInputElement>(null)
  const savingRef = useRef(saving)

  useEffect(() => {
    savingRef.current = saving
  }, [saving])

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    keyInputRef.current?.focus()
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !savingRef.current) onClose()
      if (event.key !== 'Tab' || !dialogRef.current) return
      const focusable = Array.from(dialogRef.current.querySelectorAll<HTMLElement>('button:not([disabled]), input:not([disabled]), select:not([disabled]), summary, [tabindex]:not([tabindex="-1"])'))
      if (!focusable.length) return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = previousOverflow
      previous?.focus()
    }
  }, [onClose])

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setLocalError('')
    try {
      await onSave(draft)
      onClose()
    } catch (error) {
      setLocalError(getErrorMessage(error))
    }
  }

  return (
    <div className="modal-overlay" onMouseDown={(event) => { if (event.target === event.currentTarget && !saving) onClose() }}>
      <div className="settings-dialog" ref={dialogRef} role="dialog" aria-modal="true" aria-labelledby="settings-title" aria-describedby="settings-description">
        <form onSubmit={submit}>
          <div className="dialog-heading">
            <div className="dialog-icon" aria-hidden="true"><KeyRound size={21} /></div>
            <div><span className="eyebrow">Provider settings</span><h2 id="settings-title">API และโมเดล</h2><p id="settings-description">ตั้งค่าการสร้างเนื้อหาโดยไม่เปิดเผยกุญแจของคุณ</p></div>
            <button className="icon-button dialog-close" type="button" onClick={onClose} disabled={saving} aria-label="ปิดหน้าต่างตั้งค่า"><X size={19} aria-hidden="true" /></button>
          </div>

          <div className="session-disclosure">
            <ShieldCheck size={20} aria-hidden="true" />
            <div><strong>API key อยู่ในหน่วยความจำของ session นี้เท่านั้น</strong><p>แอปจะไม่เขียน key ลงไฟล์โปรเจกต์หรือไฟล์ตั้งค่า เมื่อปิดแอป key จะหายไป</p></div>
          </div>

          <div className="field">
            <label htmlFor="api-key">Gemini API key</label>
            <div className="input-with-action">
              <input ref={keyInputRef} id="api-key" type={showKey ? 'text' : 'password'} value={apiKey} onChange={(event) => onApiKeyChange(event.target.value)} placeholder={apiKeyFromEnvironment ? 'ใช้ GEMINI_API_KEY จาก environment ได้แล้ว' : 'AIza…'} autoComplete="off" spellCheck={false} />
              <button type="button" className="input-action" onClick={() => setShowKey((value) => !value)} aria-label={showKey ? 'ซ่อน API key' : 'แสดง API key'}>{showKey ? <Eye size={17} aria-hidden="true" /> : <KeyRound size={17} aria-hidden="true" />}</button>
            </div>
            <span className="field-help">{apiKeyFromEnvironment ? 'ตรวจพบ GEMINI_API_KEY — เว้นช่องนี้ว่างเพื่อใช้ค่านั้น' : 'สร้างเนื้อหาไม่ได้จนกว่าจะใส่ key หรือกำหนด GEMINI_API_KEY'}</span>
          </div>

          <div className="form-grid dialog-form-grid">
            <div className="field">
              <label htmlFor="provider">ผู้ให้บริการ</label>
              <select id="provider" value={draft.provider} onChange={(event) => setDraft({...draft, provider: event.target.value})}><option value="gemini">Google Gemini</option></select>
            </div>
            <div className="field">
              <label htmlFor="model">โมเดล</label>
              <input id="model" value={draft.model} onChange={(event) => setDraft({...draft, model: event.target.value})} />
            </div>
          </div>

          <label className="checkbox-row" htmlFor="grounding">
            <input id="grounding" type="checkbox" checked={draft.useGrounding} onChange={(event) => setDraft({...draft, useGrounding: event.target.checked})} />
            <span><strong>ใช้ Google Search และ URL context</strong><small>ช่วยค้นบริบทเพิ่ม แต่ไม่ได้แทนการตรวจข้อเท็จจริงจาก evidence pack</small></span>
          </label>

          <details className="advanced-settings">
            <summary>การตั้งค่าขั้นสูง <ChevronRight size={15} aria-hidden="true" /></summary>
            <div className="field"><label htmlFor="base-url">Interactions API endpoint</label><input id="base-url" type="url" value={draft.baseUrl} onChange={(event) => setDraft({...draft, baseUrl: event.target.value})} /></div>
          </details>

          {localError && <div className="dialog-error" role="alert"><AlertCircle size={16} aria-hidden="true" />{localError}</div>}

          <div className="dialog-actions">
            {apiKey && <button className="button button-quiet danger-text" type="button" onClick={() => onApiKeyChange('')} disabled={saving}>ล้าง key จาก session</button>}
            <span className="dialog-spacer" />
            <button className="button button-quiet" type="button" onClick={onClose} disabled={saving}>ยกเลิก</button>
            <button className="button button-primary" type="submit" disabled={saving}>
              {saving ? <LoaderCircle className="spin" size={17} aria-hidden="true" /> : <Save size={17} aria-hidden="true" />}
              {saving ? 'กำลังบันทึก…' : 'บันทึกการตั้งค่า'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function App() {
  const [appMode, setAppMode] = useState<AppMode>(initialAppMode)
  const [aiStages, setAIStages] = useState<AIStudioStages>(EMPTY_AI_STAGES)
  const [aiProvider, setAIProvider] = useState<FacebookProviderID | 'none'>('none')
  const [aiWorkflow, setAIWorkflow] = useState('ยังไม่มีงานที่กำลังทำ')
  const [aiConnection, setAIConnection] = useState<AIConnectionStatus>('connecting')
  const [aiActivity, setAIActivity] = useState<AIActivityMessage[]>([])
  const activeAIRunRef = useRef('')
  const activitySequenceRef = useRef(0)
  const [settings, setSettings] = useState<ProviderSettings>(DEFAULT_SETTINGS)
  const [project, setProject] = useState<Project>(() => createEmptyProject(DEFAULT_SETTINGS))
  const [projects, setProjects] = useState<ProjectSummary[]>([])
  const [view, setView] = useState<WorkspaceView>('brief')
  const [editorView, setEditorView] = useState<EditorView>('fields')
  const [prompt, setPrompt] = useState<PromptPreview | null>(null)
  const [usage, setUsage] = useState<Usage | null>(null)
  const [modelUsed, setModelUsed] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [apiKeyFromEnvironment, setApiKeyFromEnvironment] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [briefErrors, setBriefErrors] = useState<BriefErrors>({})
  const [busy, setBusy] = useState<BusyAction>('bootstrap')
  const [loadingProjectId, setLoadingProjectId] = useState('')
  const [qualityEvaluating, setQualityEvaluating] = useState(false)
  const [qualityError, setQualityError] = useState('')
  const [notice, setNotice] = useState<Notice | null>(null)
  const [dirty, setDirty] = useState(false)
  const [exportFormat, setExportFormat] = useState('html')
  const [growthTab, setGrowthTab] = useState<GrowthTab>('playbooks')
  const [preferredGrowthPlaybook, setPreferredGrowthPlaybook] = useState('')

  const changeAppMode = useCallback((mode: AppMode) => {
    setAppMode(mode)
    try {
      window.localStorage.setItem('content-blueprint:workspace', mode)
    } catch {
      // The workspace switch remains usable when persistence is unavailable.
    }
  }, [])

  const navigateOnboarding = useCallback(async (destination: TourDestination) => {
    setAppMode(destination.mode)
    if (destination.growthTab) setGrowthTab(destination.growthTab)
    if (destination.playbookId) setPreferredGrowthPlaybook(destination.playbookId)
    if (destination.seoView) setView(destination.seoView)
    try {
      window.localStorage.setItem('content-blueprint:workspace', destination.mode)
    } catch {
      // Navigation still works if the webview blocks local persistence.
    }
  }, [])

  useEffect(() => {
    if (!preferredGrowthPlaybook) return
    const timer = window.setTimeout(() => setPreferredGrowthPlaybook(''), 1200)
    return () => window.clearTimeout(timer)
  }, [preferredGrowthPlaybook])

  const startAIRun = useCallback((run: {
    runId: string
    provider: Exclude<FacebookProviderID, 'mcp'>
    workflow: FacebookWorkflow
  }) => {
    activeAIRunRef.current = run.runId
    setAIProvider(run.provider)
    setAIWorkflow(run.workflow === 'team'
      ? 'AI Team · Strategist → Copywriter → Reviewer'
      : 'Quick draft · worker เดียว')
    setAIConnection('connecting')
    setAIStages({...EMPTY_AI_STAGES})
    activitySequenceRef.current += 1
    setAIActivity([{
      id: `${run.runId}:start:${activitySequenceRef.current}`,
      message: run.workflow === 'team'
        ? 'เปิดทีม AI แบบแยก process และเตรียมส่ง Brief ให้ Strategist'
        : 'เปิด AI worker สำหรับสร้าง Quick draft',
      timestamp: new Date().toISOString(),
      tone: 'info',
    }])
  }, [])

  const finishAIRun = useCallback((run: {
    runId: string
    workflow: FacebookWorkflow
    error?: string
  }) => {
    if (run.runId !== activeAIRunRef.current) return
    setAIConnection(wailsApi.isAvailable() ? 'connected' : 'disconnected')
    if (run.error) {
      const fallbackStage: AIStageId = run.workflow === 'team' ? 'strategist' : 'copywriter'
      setAIStages((current) => {
        if (Object.values(current).some((stage) => typeof stage === 'object' && stage.status === 'error')) {
          return current
        }
        return {
          ...current,
          [fallbackStage]: {status: 'error', summary: run.error, updatedAt: new Date().toISOString()},
        }
      })
      activitySequenceRef.current += 1
      setAIActivity((current) => [...current, {
        id: `${run.runId}:finish:${activitySequenceRef.current}`,
        stage: fallbackStage,
        message: run.error ?? 'งานหยุดก่อนรับผลลัพธ์',
        timestamp: new Date().toISOString(),
        tone: 'error' as const,
      }].slice(-24))
    }
    activeAIRunRef.current = ''
  }, [])

  useEffect(() => {
    const runtimeWindow = window as Window & {
      runtime?: {EventsOnMultiple?: (...args: unknown[]) => unknown}
    }
    if (!wailsApi.isAvailable() || typeof runtimeWindow.runtime?.EventsOnMultiple !== 'function') {
      setAIConnection('disconnected')
      return
    }

    setAIConnection('connected')
    const cancelListeners: Array<() => void> = []
    try {
      const handleStageUpdate = (raw: unknown) => {
        if (!raw || typeof raw !== 'object') return
        const candidate = raw as Partial<FacebookStageUpdate>
        if (
          typeof candidate.runId !== 'string'
          || typeof candidate.stage !== 'string'
          || !AI_STAGE_IDS.has(candidate.stage as AIStageId)
          || typeof candidate.status !== 'string'
          || !AI_STAGE_STATUSES.has(candidate.status)
          || typeof candidate.message !== 'string'
        ) return

        if (!activeAIRunRef.current) activeAIRunRef.current = candidate.runId
        if (candidate.runId !== activeAIRunRef.current) return

        const stage = candidate.stage as AIStageId
        const status = candidate.status as FacebookStageUpdate['status']
        const timestamp = typeof candidate.occurredAt === 'string'
          ? candidate.occurredAt
          : new Date().toISOString()
        if (candidate.provider === 'claude' || candidate.provider === 'codex' || candidate.provider === 'mcp') {
          setAIProvider(candidate.provider)
        }
        if (candidate.workflow === 'team' || candidate.workflow === 'single') {
          setAIWorkflow(candidate.workflow === 'team'
            ? 'AI Team · Strategist → Copywriter → Reviewer'
            : 'Quick draft · worker เดียว')
        }
        setAIConnection('connected')
        setAIStages((current) => ({
          ...current,
          [stage]: {status, summary: candidate.message, updatedAt: timestamp},
        }))
        activitySequenceRef.current += 1
        const activity: AIActivityMessage = {
          id: `${candidate.runId}:${activitySequenceRef.current}`,
          stage,
          message: candidate.message,
          timestamp,
          tone: status === 'error' ? 'error' : status === 'done' ? 'success' : 'info',
        }
        setAIActivity((current) => [...current, activity].slice(-24))
      }
      cancelListeners.push(EventsOn('facebook:ai-stage', handleStageUpdate))
      cancelListeners.push(EventsOn('growth:ai-stage', handleStageUpdate))
    } catch {
      setAIConnection('error')
    }

    return () => cancelListeners.forEach((cancel) => cancel())
  }, [])

  useEffect(() => {
    let active = true
    const bootstrap = async () => {
      if (!wailsApi.isAvailable()) {
        if (!active) return
        setBusy(null)
        setNotice({tone: 'info', message: 'กำลังแสดง frontend preview — เปิดผ่าน Wails เพื่อบันทึก สร้าง และส่งออกจริง'})
        return
      }
      try {
        const data = await wailsApi.bootstrap()
        if (!active) return
        const nextSettings = normalizeSettings(data.settings)
        setSettings(nextSettings)
        setProject(createEmptyProject(nextSettings))
        setProjects(Array.isArray(data.projects) ? data.projects : [])
        setApiKeyFromEnvironment(Boolean(data.apiKeyFromEnvironment))
      } catch (error) {
        if (active) setNotice({tone: 'error', message: getErrorMessage(error)})
      } finally {
        if (active) setBusy(null)
      }
    }
    void bootstrap()
    return () => { active = false }
  }, [])

  useEffect(() => {
    if (!project.content || !wailsApi.isAvailable()) return
    let active = true
    const timer = window.setTimeout(async () => {
      setQualityEvaluating(true)
      setQualityError('')
      try {
        const quality = await wailsApi.evaluateContent(project.brief, project.content as GeneratedContent)
        if (active) setProject((current) => ({...current, quality}))
      } catch (error) {
        if (active) setQualityError(getErrorMessage(error))
      } finally {
        if (active) setQualityEvaluating(false)
      }
    }, 650)
    return () => {
      active = false
      window.clearTimeout(timer)
    }
  }, [project.brief, project.content])

  const apiReady = Boolean(apiKey.trim() || apiKeyFromEnvironment)
  const currentName = project.name || project.brief.keyword || 'โปรเจกต์ใหม่'
  const workspaceLocked = busy === 'load' || busy === 'prompt' || busy === 'generate'
  const closeSettings = useCallback(() => setSettingsOpen(false), [])

  const updateBrief = (brief: ContentBrief) => {
    setProject((current) => {
      const shouldAdoptKeyword = !current.id && (!current.name || current.name === 'โปรเจกต์ใหม่' || current.name === current.brief.keyword)
      return {...current, name: shouldAdoptKeyword && brief.keyword.trim() ? brief.keyword.trim() : current.name, brief}
    })
    setBriefErrors((current) => {
      const currentValidation = validateBrief(brief)
      return Object.fromEntries(
        Object.keys(current)
          .filter((key) => currentValidation[key])
          .map((key) => [key, currentValidation[key]]),
      )
    })
    setDirty(true)
  }

  const ensureValidBrief = () => {
    const errors = validateBrief(project.brief)
    setBriefErrors(errors)
    const firstError = Object.keys(errors)[0]
    if (firstError) {
      setView('brief')
      setNotice({tone: 'error', message: 'เติมข้อมูลจำเป็นใน brief ให้ครบก่อนดำเนินการ'})
      window.setTimeout(() => document.getElementById(firstError)?.focus(), 0)
      return false
    }
    return true
  }

  const handleBuildPrompt = async () => {
    if (!ensureValidBrief()) return
    setBusy('prompt')
    setNotice(null)
    try {
      const result = await wailsApi.buildPrompt(project.brief)
      setPrompt(result)
      setView('prompt')
    } catch (error) {
      setNotice({tone: 'error', message: getErrorMessage(error)})
    } finally {
      setBusy(null)
    }
  }

  const handleGenerate = async () => {
    if (!ensureValidBrief()) return
    if (!apiReady) {
      setSettingsOpen(true)
      setNotice({tone: 'error', message: 'ใส่ Gemini API key สำหรับ session นี้ก่อนสร้างเนื้อหา'})
      return
    }
    setBusy('generate')
    setNotice(null)
    try {
      const result = await wailsApi.generateContent({brief: project.brief, settings: project.settings, apiKey})
      setProject((current) => ({
        ...current,
        content: normalizeContent(result.content),
        quality: result.quality,
        groundingSources: Array.isArray(result.groundingSources) ? result.groundingSources : [],
      }))
      setPrompt(result.prompt)
      setUsage(result.usage)
      setModelUsed(result.model)
      setDirty(true)
      setEditorView('fields')
      setView('editor')
      setNotice({tone: 'success', message: 'สร้างบทความแล้ว ตรวจคะแนนและแก้ไขก่อนบันทึกหรือส่งออก'})
    } catch (error) {
      setNotice({tone: 'error', message: getErrorMessage(error)})
    } finally {
      setBusy(null)
    }
  }

  const refreshProjects = async () => {
    try {
      const nextProjects = await wailsApi.listProjects()
      setProjects(Array.isArray(nextProjects) ? nextProjects : [])
    } catch {
      // Saving succeeded; a stale rail is less disruptive than replacing that success with an error.
    }
  }

  const handleSave = async () => {
    setBusy('save')
    setNotice(null)
    try {
      const saved = normalizeProject(await wailsApi.saveProject({...project, name: currentName, settings: project.settings}))
      setProject(saved)
      setDirty(false)
      setNotice({tone: 'success', message: 'บันทึกโปรเจกต์ไว้ในเครื่องแล้ว'})
      await refreshProjects()
    } catch (error) {
      setNotice({tone: 'error', message: getErrorMessage(error)})
    } finally {
      setBusy(null)
    }
  }

  const confirmDiscard = () => !dirty || window.confirm('มีการแก้ไขที่ยังไม่ได้บันทึก ต้องการดำเนินการต่อและทิ้งการแก้ไขหรือไม่?')

  const handleNew = () => {
    if (!confirmDiscard()) return
    setProject(createEmptyProject(settings))
    setPrompt(null)
    setUsage(null)
    setModelUsed('')
    setBriefErrors({})
    setNotice(null)
    setDirty(false)
    setView('brief')
  }

  const handleOpen = async (id: string) => {
    if (id === project.id || !confirmDiscard()) return
    setBusy('load')
    setLoadingProjectId(id)
    setNotice(null)
    try {
      const loaded = normalizeProject(await wailsApi.loadProject(id))
      setProject(loaded)
      setPrompt(null)
      setUsage(null)
      setModelUsed('')
      setDirty(false)
      setView(loaded.content ? 'editor' : 'brief')
    } catch (error) {
      setNotice({tone: 'error', message: getErrorMessage(error)})
    } finally {
      setBusy(null)
      setLoadingProjectId('')
    }
  }

  const handleDelete = async (summary: ProjectSummary) => {
    if (!window.confirm(`ลบโปรเจกต์ “${summary.name}” ออกจากเครื่องหรือไม่? การดำเนินการนี้ย้อนกลับไม่ได้`)) return
    setBusy('delete')
    setLoadingProjectId(summary.id)
    try {
      await wailsApi.deleteProject(summary.id)
      setProjects((current) => current.filter((item) => item.id !== summary.id))
      if (summary.id === project.id) {
        setProject(createEmptyProject(settings))
        setPrompt(null)
        setUsage(null)
        setModelUsed('')
        setBriefErrors({})
        setDirty(false)
        setView('brief')
      }
      setNotice({tone: 'success', message: `ลบโปรเจกต์ “${summary.name}” แล้ว`})
    } catch (error) {
      setNotice({tone: 'error', message: getErrorMessage(error)})
    } finally {
      setBusy(null)
      setLoadingProjectId('')
    }
  }

  const handleExport = async () => {
    if (!project.content) {
      setNotice({tone: 'error', message: 'ยังไม่มีบทความสำหรับส่งออก กรุณาสร้างเนื้อหาก่อน'})
      setView('editor')
      return
    }
    setBusy('export')
    setNotice(null)
    try {
      const path = await wailsApi.exportProject(project, exportFormat)
      setNotice(path
        ? {tone: 'success', message: `ส่งออกไฟล์แล้ว: ${path}`}
        : {tone: 'info', message: 'ยกเลิกการส่งออกแล้ว ไม่มีไฟล์ถูกสร้าง'})
    } catch (error) {
      setNotice({tone: 'error', message: getErrorMessage(error)})
    } finally {
      setBusy(null)
    }
  }

  const handleCopy = async (text: string, label: string) => {
    try {
      await copyText(text)
      setNotice({tone: 'success', message: `คัดลอก${label}แล้ว`})
    } catch (error) {
      setNotice({tone: 'error', message: `คัดลอกไม่สำเร็จ: ${getErrorMessage(error)}`})
    }
  }

  const handleSettingsSave = async (nextSettings: ProviderSettings) => {
    setBusy('settings')
    try {
      if (wailsApi.isAvailable()) await wailsApi.saveSettings(nextSettings)
      setSettings(nextSettings)
      setProject((current) => ({...current, settings: nextSettings}))
      setDirty(true)
      setNotice({tone: 'success', message: wailsApi.isAvailable() ? 'บันทึกการตั้งค่าแล้ว โดยไม่บันทึก API key' : 'อัปเดตค่าใน frontend preview แล้ว'})
    } finally {
      setBusy(null)
    }
  }

  const aiStudio: ReactNode = (
    <AIStudio
      provider={aiProvider}
      workflow={aiWorkflow}
      stages={aiStages}
      connectionStatus={aiConnection}
      activityMessages={aiActivity}
    />
  )

  const seoWorkspace: ReactNode = busy === 'bootstrap' ? (
    <main className="app-loading" aria-live="polite">
      <div className="brand-mark large" aria-hidden="true"><BookOpenText size={26} /></div>
      <LoaderCircle className="spin" size={22} aria-hidden="true" />
      <p>กำลังเปิดพื้นที่ทำงาน…</p>
    </main>
  ) : (
    <div className="app-shell">
      <a className="skip-link" href="#workspace-content">ข้ามไปพื้นที่ทำงาน</a>
      <ProjectRail projects={projects} activeId={project.id} apiReady={apiReady} loadingId={loadingProjectId} disabled={busy !== null} onNew={handleNew} onOpen={handleOpen} onDelete={handleDelete} onSettings={() => setSettingsOpen(true)} />

      <main className="main-workspace" id="workspace-content">
        <h1 className="sr-only">Content Blueprint: {currentName}</h1>
        <header className="workspace-header">
          <div className="project-title-field">
            <label htmlFor="project-name">ชื่อโปรเจกต์</label>
            <div className="project-title-input-wrap">
              <input id="project-name" value={project.name} onChange={(event) => { setProject((current) => ({...current, name: event.target.value})); setDirty(true) }} aria-label="ชื่อโปรเจกต์" disabled={busy !== null} />
              {dirty && <span className="unsaved-dot" title="มีการแก้ไขที่ยังไม่บันทึก"><span className="sr-only">มีการแก้ไขที่ยังไม่บันทึก</span></span>}
            </div>
          </div>

          <div className="header-actions">
            <button className={`button button-quiet api-button ${apiReady ? 'is-ready' : ''}`} type="button" onClick={() => setSettingsOpen(true)} disabled={busy !== null}>
              {apiReady ? <ShieldCheck size={16} aria-hidden="true" /> : <KeyRound size={16} aria-hidden="true" />}
              {apiReady ? 'API พร้อม' : 'ตั้ง API key'}
            </button>
            <button className="button button-secondary" type="button" onClick={handleSave} disabled={busy !== null}>
              {busy === 'save' ? <LoaderCircle className="spin" size={16} aria-hidden="true" /> : <Save size={16} aria-hidden="true" />}
              บันทึก
            </button>
            <div className="export-control">
              <label htmlFor="export-format">ส่งออก</label>
              <select id="export-format" value={exportFormat} onChange={(event) => setExportFormat(event.target.value)} disabled={busy === 'export'}>
                <option value="html">HTML</option>
                <option value="markdown">Markdown</option>
                <option value="json">JSON</option>
              </select>
              <button className="icon-button export-button" type="button" onClick={handleExport} disabled={busy !== null} aria-label={`ส่งออกเป็น ${exportFormat}`}>
                {busy === 'export' ? <LoaderCircle className="spin" size={17} aria-hidden="true" /> : <Download size={17} aria-hidden="true" />}
              </button>
            </div>
          </div>
        </header>

        <div className="workflow-bar" data-tour="seo-workflow">
          <nav className="workflow-tabs" aria-label="ขั้นตอนการทำงาน">
            <button type="button" className={view === 'brief' ? 'is-active' : ''} aria-current={view === 'brief' ? 'step' : undefined} onClick={() => setView('brief')} disabled={busy !== null}><span>1</span>Brief</button>
            <ChevronRight size={14} aria-hidden="true" />
            <button type="button" className={view === 'prompt' ? 'is-active' : ''} aria-current={view === 'prompt' ? 'step' : undefined} onClick={() => setView('prompt')} disabled={busy !== null}><span>2</span>Prompt</button>
            <ChevronRight size={14} aria-hidden="true" />
            <button type="button" className={view === 'editor' ? 'is-active' : ''} aria-current={view === 'editor' ? 'step' : undefined} onClick={() => setView('editor')} disabled={busy !== null}><span>3</span>Editor</button>
          </nav>
          <div className="workflow-actions">
            <button className="button button-secondary" type="button" onClick={handleBuildPrompt} disabled={busy !== null}>
              {busy === 'prompt' ? <LoaderCircle className="spin" size={16} aria-hidden="true" /> : <Code2 size={16} aria-hidden="true" />}
              ดูพรอมป์
            </button>
            <button className="button button-primary" type="button" onClick={handleGenerate} disabled={busy !== null} data-tour="seo-generate">
              {busy === 'generate' ? <LoaderCircle className="spin" size={17} aria-hidden="true" /> : <Sparkles size={17} aria-hidden="true" />}
              {busy === 'generate' ? 'กำลังสร้างเนื้อหา…' : 'สร้างบทความ'}
            </button>
          </div>
        </div>

        {notice && (
          <div className={`notice notice-${notice.tone}`} role={notice.tone === 'error' ? 'alert' : 'status'}>
            {notice.tone === 'success' ? <CheckCircle2 size={17} aria-hidden="true" /> : notice.tone === 'error' ? <AlertCircle size={17} aria-hidden="true" /> : <ShieldCheck size={17} aria-hidden="true" />}
            <span>{notice.message}</span>
            <button className="icon-button" type="button" onClick={() => setNotice(null)} aria-label="ปิดข้อความ"><X size={15} aria-hidden="true" /></button>
          </div>
        )}

        <div className="workspace-scroll" aria-busy={workspaceLocked} inert={workspaceLocked ? true : undefined}>
          {view === 'brief' && <BriefWorkspace brief={project.brief} errors={briefErrors} onChange={updateBrief} />}
          {view === 'prompt' && <PromptWorkspace prompt={prompt} loading={busy === 'prompt'} onBuild={handleBuildPrompt} onCopy={handleCopy} />}
          {view === 'editor' && <EditorWorkspace content={project.content} view={editorView} onViewChange={setEditorView} onChange={(content) => { setProject((current) => ({...current, content})); setDirty(true) }} />}
        </div>
      </main>

      <QualityPanel brief={project.brief} report={project.quality} usage={usage} model={modelUsed} evaluating={qualityEvaluating} error={qualityError} groundingSources={project.groundingSources} />

      {settingsOpen && <SettingsDialog settings={project.settings} apiKey={apiKey} apiKeyFromEnvironment={apiKeyFromEnvironment} saving={busy === 'settings'} onApiKeyChange={setApiKey} onClose={closeSettings} onSave={handleSettingsSave} />}
    </div>
  )

  return (
    <>
      <SuiteModeSwitcher mode={appMode} onChange={changeAppMode} />

      <section className="suite-facebook-root" hidden={appMode !== 'facebook'} aria-hidden={appMode !== 'facebook'} inert={appMode !== 'facebook' ? true : undefined}>
        <FacebookWorkspace
          onRunStart={startAIRun}
          onRunFinish={finishAIRun}
          aiStudio={aiStudio}
        />
      </section>

      <section className="suite-growth-root" hidden={appMode !== 'growth'} aria-hidden={appMode !== 'growth'} inert={appMode !== 'growth' ? true : undefined}>
        <GrowthWorkspace
          activeTab={growthTab}
          onTabChange={setGrowthTab}
          preferredPlaybookId={preferredGrowthPlaybook}
          onRunStart={startAIRun}
          onRunFinish={finishAIRun}
          aiStudio={aiStudio}
        />
      </section>

      <section className="suite-seo-root" hidden={appMode !== 'seo'} aria-hidden={appMode !== 'seo'} inert={appMode !== 'seo' ? true : undefined}>
        {seoWorkspace}
      </section>

      <UpdateCenter />
      <OnboardingProvider onNavigate={navigateOnboarding} />
    </>
  )
}

export default App
