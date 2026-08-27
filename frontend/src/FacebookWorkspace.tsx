import {
  AlertCircle,
  Check,
  CheckCircle2,
  Clipboard,
  ExternalLink,
  FilePlus2,
  FileText,
  Link2,
  LoaderCircle,
  Plus,
  Save,
  ShieldCheck,
  Sparkles,
  Square,
  Trash2,
  TriangleAlert,
  Users,
  WandSparkles,
  Zap,
} from 'lucide-react'
import {type FormEvent, type ReactNode, useEffect, useMemo, useRef, useState} from 'react'
import './FacebookWorkspace.css'
import {wailsApi} from './lib/wails'
import {
  createEmptyFacebookBrief,
  type EvidenceSource,
  type FacebookBrief,
  type FacebookContentPack,
  type FacebookPackSnapshot,
  type FacebookProviderID,
  type FacebookProviderStatus,
  type FacebookWorkflow,
} from './types'

type BusyAction = 'bootstrap' | 'generate' | 'sync' | 'fetch' | null
type Notice = {tone: 'error' | 'success' | 'info'; message: string}
type BriefErrors = Record<string, string | undefined>

const PROVIDER_ORDER: FacebookProviderID[] = ['claude', 'codex', 'mcp']
const PROVIDER_FALLBACKS: Record<FacebookProviderID, FacebookProviderStatus> = {
  claude: {id: 'claude', label: 'Claude CLI', available: false, version: '', message: 'กำลังตรวจสอบ'},
  codex: {id: 'codex', label: 'Codex CLI', available: false, version: '', message: 'กำลังตรวจสอบ'},
  mcp: {id: 'mcp', label: 'MCP Companion', available: true, version: '', message: 'ซิงก์งานกับ Claude หรือ Codex'},
}

const providerDescription: Record<FacebookProviderID, string> = {
  claude: 'สร้างผ่าน Claude CLI ด้วยบัญชีที่ล็อกอินในเครื่อง',
  codex: 'สร้างผ่าน Codex CLI ด้วยบัญชีที่ล็อกอินในเครื่อง',
  mcp: 'ส่ง Brief ให้เซิร์ฟเวอร์ MCP แล้วดึงผลที่โมเดลบันทึกไว้',
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return typeof error === 'string' ? error : 'เกิดข้อผิดพลาดที่ไม่ทราบสาเหตุ'
}

function formatUpdatedAt(value?: string): string {
  if (!value) return 'ยังไม่มีผลลัพธ์'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('th-TH', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function safeExternalURL(value: string): string | null {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.toString() : null
  } catch {
    return null
  }
}

function validateBrief(brief: FacebookBrief): BriefErrors {
  const errors: BriefErrors = {}
  if (!brief.topic.trim()) errors.topic = 'ระบุสินค้า บริการ หรือหัวข้อที่จะเขียน'
  if (!brief.audience.trim()) errors.audience = 'ระบุว่าโพสต์นี้ต้องการสื่อสารกับใคร'
  if (!brief.objective.trim()) errors.objective = 'ระบุผลลัพธ์ที่ต้องการจากโพสต์'
  if (!brief.language.trim()) errors.language = 'ระบุภาษาของเนื้อหา'

  const ids = brief.evidence.map((source) => source.id.trim())
  brief.evidence.forEach((source, index) => {
    const id = source.id.trim()
    if (!id || !/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(id)) {
      errors[`evidence-id-${index}`] = 'รหัสต้องขึ้นต้นด้วยตัวอักษรหรือตัวเลข'
    } else if (ids.filter((candidate) => candidate === id).length > 1) {
      errors[`evidence-id-${index}`] = 'รหัสแหล่งข้อมูลซ้ำ'
    }
    if (!source.title.trim()) errors[`evidence-title-${index}`] = 'ใส่ชื่อแหล่งข้อมูล'
    if (source.url.trim() && !safeExternalURL(source.url.trim())) {
      errors[`evidence-url-${index}`] = 'URL ต้องขึ้นต้นด้วย http:// หรือ https://'
    }
  })
  return errors
}

function packText(pack: FacebookContentPack): string {
  const slides = pack.carouselSlides.map((slide, index) => `สไลด์ ${index + 1}: ${slide.headline}\n${slide.body}`).join('\n\n')
  const replies = pack.replyBank.map((item) => `${item.intent}\n${item.reply}`).join('\n\n')
  return [
    `HOOKS\n${pack.hooks.map((hook, index) => `${index + 1}. ${hook}`).join('\n')}`,
    `LONG POST\n${pack.longPost}`,
    `SHORT POST\n${pack.shortPost}`,
    `REEL SCRIPT\n${pack.reelScript}`,
    `CAROUSEL\n${slides}`,
    `CTA\n${pack.cta}`,
    `FIRST COMMENT\n${pack.firstComment}`,
    `REPLY BANK\n${replies}`,
    `COMPLIANCE NOTES\n${pack.complianceNotes.join('\n') || 'ไม่มี'}`,
  ].join('\n\n---\n\n')
}

interface OutputCardProps {
  title: string
  number: number
  copyValue: string
  children: ReactNode
  caution?: boolean
  onCopy: (value: string, label: string) => void
}

export interface FacebookWorkspaceProps {
  /** Optional live agent-stage UI supplied by the app shell (for example AIStudio). */
  aiStudio?: ReactNode
  /** Lets the app shell bind live Wails events to the run that this workspace started. */
  onRunStart?: (run: {
    runId: string
    provider: Exclude<FacebookProviderID, 'mcp'>
    workflow: FacebookWorkflow
  }) => void
  /** Prevents the visualizer from remaining in a connecting state if a run fails before its first Wails event. */
  onRunFinish?: (run: {
    runId: string
    workflow: FacebookWorkflow
    error?: string
  }) => void
}

function OutputCard({title, number, copyValue, children, caution, onCopy}: OutputCardProps) {
  return (
    <section className={`fbw-output-card${caution ? ' fbw-output-card-caution' : ''}`} aria-labelledby={`fbw-output-${number}`}>
      <header className="fbw-output-card-header">
        <div>
          <span className="fbw-output-number" aria-hidden="true">{String(number).padStart(2, '0')}</span>
          <h3 id={`fbw-output-${number}`}>{title}</h3>
        </div>
        <button className="fbw-icon-button" type="button" onClick={() => onCopy(copyValue, title)} aria-label={`คัดลอก${title}`} disabled={!copyValue}>
          <Clipboard size={17} aria-hidden="true" />
        </button>
      </header>
      <div className="fbw-output-body">{children}</div>
    </section>
  )
}

export function FacebookWorkspace({aiStudio, onRunStart, onRunFinish}: FacebookWorkspaceProps) {
  const [brief, setBrief] = useState<FacebookBrief>(() => createEmptyFacebookBrief())
  const briefRef = useRef(brief)
  const syncedBriefRevisionRef = useRef('')
  const [providers, setProviders] = useState<FacebookProviderStatus[]>(() => PROVIDER_ORDER.map((id) => PROVIDER_FALLBACKS[id]))
  const [provider, setProvider] = useState<FacebookProviderID>('mcp')
  const [workflow, setWorkflow] = useState<FacebookWorkflow>('team')
  const [activeRunId, setActiveRunId] = useState('')
  const [snapshot, setSnapshot] = useState<FacebookPackSnapshot | null>(null)
  const [outputStale, setOutputStale] = useState(false)
  const [errors, setErrors] = useState<BriefErrors>({})
  const [notice, setNotice] = useState<Notice | null>(null)
  const [busy, setBusy] = useState<BusyAction>('bootstrap')
  const resultsRef = useRef<HTMLElement>(null)

  const providerMap = useMemo(() => {
    const map = new Map<FacebookProviderID, FacebookProviderStatus>()
    PROVIDER_ORDER.forEach((id) => map.set(id, PROVIDER_FALLBACKS[id]))
    providers.forEach((item) => map.set(item.id, item))
    return map
  }, [providers])

  useEffect(() => {
    let active = true
    void wailsApi.facebookBootstrap()
      .then((data) => {
        if (!active) return
        const nextProviders = Array.isArray(data.providers) ? data.providers : []
        setProviders(nextProviders)
        setSnapshot(data.latest ?? null)
        setOutputStale(Boolean(data.latest))
        const preferred = PROVIDER_ORDER.find((id) => nextProviders.some((item) => item.id === id && item.available)) ?? 'mcp'
        setProvider(preferred)
      })
      .catch((error) => {
        if (active) setNotice({tone: 'error', message: getErrorMessage(error)})
      })
      .finally(() => {
        if (active) setBusy(null)
      })
    return () => { active = false }
  }, [])

  const updateBrief = <K extends keyof FacebookBrief>(key: K, value: FacebookBrief[K]) => {
    syncedBriefRevisionRef.current = ''
    setBrief((current) => {
      const next = {...current, [key]: value}
      briefRef.current = next
      return next
    })
    setErrors((current) => ({...current, [key]: undefined}))
    if (snapshot) setOutputStale(true)
  }

  const updateEvidence = (index: number, key: keyof EvidenceSource, value: string) => {
    const next = brief.evidence.map((source, sourceIndex) => sourceIndex === index ? {...source, [key]: value} : source)
    updateBrief('evidence', next)
    setErrors((current) => ({...current, [`evidence-${key}-${index}`]: undefined}))
  }

  const addEvidence = () => {
    const used = new Set(brief.evidence.map((source) => source.id))
    let sequence = brief.evidence.length + 1
    while (used.has(`source-${sequence}`)) sequence += 1
    updateBrief('evidence', [...brief.evidence, {id: `source-${sequence}`, title: '', url: '', notes: ''}])
  }

  const removeEvidence = (index: number) => {
    updateBrief('evidence', brief.evidence.filter((_, sourceIndex) => sourceIndex !== index))
  }

  const ensureValidBrief = (): boolean => {
    const nextErrors = validateBrief(brief)
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length === 0) return true
    setNotice({tone: 'error', message: 'ยังมีข้อมูลจำเป็นที่ต้องแก้ไข กรุณาตรวจช่องที่มีข้อความสีแดง'})
    requestAnimationFrame(() => document.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus())
    return false
  }

  const generate = async (event: FormEvent) => {
    event.preventDefault()
    if (provider === 'mcp') {
      await syncBrief()
      return
    }
    if (!ensureValidBrief()) return
    const status = providerMap.get(provider)
    if (!status?.available) {
      setNotice({tone: 'error', message: `${status?.label ?? provider} ยังไม่พร้อมใช้งาน กรุณาตรวจการติดตั้งและการล็อกอิน`})
      return
    }
    setBusy('generate')
    const runId = globalThis.crypto?.randomUUID?.() ?? `run_${Date.now()}`
    setActiveRunId(runId)
    onRunStart?.({runId, provider, workflow})
    setNotice({tone: 'info', message: `กำลังให้ ${status.label} สร้าง Content Pack…`})
    const submittedBrief = JSON.stringify(brief)
    let runError: string | undefined
    try {
      const result = await wailsApi.generateFacebookPack({runId, provider, workflow, brief})
      setSnapshot(result)
      const changedWhileRunning = JSON.stringify(briefRef.current) !== submittedBrief
      syncedBriefRevisionRef.current = changedWhileRunning ? '' : result.briefRevision
      setOutputStale(changedWhileRunning)
      setNotice(changedWhileRunning
        ? {tone: 'info', message: `สร้าง Content Pack ด้วย ${status.label} แล้ว แต่ Brief ถูกแก้ระหว่างรัน กรุณาตรวจหรือสร้างใหม่`}
        : {tone: 'success', message: `สร้าง Content Pack ด้วย ${status.label} เรียบร้อย`})
      requestAnimationFrame(() => resultsRef.current?.focus())
    } catch (error) {
      const message = getErrorMessage(error)
      runError = message
      setNotice({tone: 'error', message})
    } finally {
      onRunFinish?.({runId, workflow, error: runError})
      setActiveRunId('')
      setBusy(null)
    }
  }

  const cancelGeneration = async () => {
    if (!activeRunId) return
    try {
      const cancelled = await wailsApi.cancelFacebookGeneration(activeRunId)
      setNotice(cancelled
        ? {tone: 'info', message: 'ส่งคำสั่งหยุดให้ทีม AI แล้ว กำลังรอ CLI ปิดงานอย่างปลอดภัย'}
        : {tone: 'info', message: 'งานนี้สิ้นสุดไปแล้ว'} )
    } catch (error) {
      setNotice({tone: 'error', message: getErrorMessage(error)})
    }
  }

  const syncBrief = async () => {
    if (!ensureValidBrief()) return
    setBusy('sync')
    setNotice({tone: 'info', message: 'กำลังซิงก์ Brief ไปยัง MCP Companion…'})
    const submittedBrief = JSON.stringify(brief)
    try {
      const result = await wailsApi.syncFacebookBrief(brief)
      const changedWhileSyncing = JSON.stringify(briefRef.current) !== submittedBrief
      syncedBriefRevisionRef.current = changedWhileSyncing ? '' : result.briefRevision
      setOutputStale(changedWhileSyncing || Boolean(snapshot && snapshot.briefRevision !== result.briefRevision))
      setNotice(changedWhileSyncing
        ? {tone: 'info', message: 'Brief ถูกแก้ระหว่างซิงก์ กรุณาซิงก์อีกครั้งเพื่อส่งเวอร์ชันล่าสุด'}
        : {tone: 'success', message: 'ซิงก์แล้ว — เปิด Claude หรือ Codex ให้อ่าน Brief ผ่าน MCP และบันทึก Content Pack กลับมา'})
    } catch (error) {
      setNotice({tone: 'error', message: getErrorMessage(error)})
    } finally {
      setBusy(null)
    }
  }

  const fetchLatest = async () => {
    setBusy('fetch')
    setNotice({tone: 'info', message: 'กำลังดึง Content Pack ล่าสุด…'})
    try {
      const result = await wailsApi.getLatestFacebookPack()
      if (!result.found || !result.snapshot) {
        setNotice({tone: 'info', message: 'ยังไม่พบ Content Pack — ให้ Claude หรือ Codex เรียกเครื่องมือ save_facebook_pack ก่อน'})
        return
      }
      setSnapshot(result.snapshot)
      const doesNotMatchSyncedBrief = syncedBriefRevisionRef.current === ''
        || result.snapshot.briefRevision !== syncedBriefRevisionRef.current
      const stale = result.stale || doesNotMatchSyncedBrief
      setOutputStale(stale)
      setNotice(stale
        ? {tone: 'info', message: 'ดึงผลลัพธ์แล้ว แต่สร้างจาก Brief คนละเวอร์ชัน กรุณาตรวจก่อนใช้'}
        : {tone: 'success', message: 'ดึง Content Pack ล่าสุดเรียบร้อย'})
      requestAnimationFrame(() => resultsRef.current?.focus())
    } catch (error) {
      setNotice({tone: 'error', message: getErrorMessage(error)})
    } finally {
      setBusy(null)
    }
  }

  const copy = async (value: string, label: string) => {
    try {
      await navigator.clipboard.writeText(value)
      setNotice({tone: 'success', message: `คัดลอก${label}แล้ว`})
    } catch {
      setNotice({tone: 'error', message: 'คัดลอกไม่สำเร็จ กรุณาเลือกข้อความแล้วคัดลอกด้วยตนเอง'})
    }
  }

  const pack = snapshot?.pack
  const isWorking = busy !== null

  return (
    <div className="fbw-shell" data-tour="facebook-shell">
      <a className="fbw-skip-link" href="#facebook-brief">ข้ามไปยัง Brief</a>
      <header className="fbw-page-header">
        <div>
          <span className="fbw-eyebrow"><Sparkles size={15} aria-hidden="true" /> Facebook Workspace</span>
          <h1>เปลี่ยน Brief เป็นชุดคอนเทนต์ที่พร้อมใช้งาน</h1>
          <p>ใช้ Claude หรือ Codex ที่ล็อกอินอยู่ในเครื่อง โดยไม่ต้องใส่ API key เพิ่ม</p>
        </div>
        {snapshot && (
          <div className="fbw-latest-meta" aria-label="ข้อมูลผลลัพธ์ล่าสุด">
            <span>{snapshot.generatedBy || 'ไม่ระบุโมเดล'}</span>
            <time dateTime={snapshot.updatedAt}>{formatUpdatedAt(snapshot.updatedAt)}</time>
          </div>
        )}
      </header>

      {aiStudio && (
        <section className="fbw-studio-slot" aria-label="สถานะทีม AI" data-tour="ai-studio">
          {aiStudio}
        </section>
      )}

      <div className="fbw-workspace">
        <main className="fbw-brief-pane" id="facebook-brief">
          <form onSubmit={generate} noValidate>
            <section className="fbw-section" aria-labelledby="fbw-provider-title">
              <div className="fbw-section-heading">
                <div>
                  <span className="fbw-step">01</span>
                  <h2 id="fbw-provider-title">เลือกวิธีสร้าง</h2>
                </div>
                {busy === 'bootstrap' && <LoaderCircle className="spin" size={18} aria-label="กำลังตรวจสอบผู้ให้บริการ" />}
              </div>
              <div className="fbw-provider-grid" role="radiogroup" aria-label="วิธีสร้างคอนเทนต์" data-tour="facebook-provider">
                {PROVIDER_ORDER.map((id) => {
                  const item = providerMap.get(id) ?? PROVIDER_FALLBACKS[id]
                  const disabled = id !== 'mcp' && !item.available
                  return (
                    <label className={`fbw-provider-card${provider === id ? ' is-selected' : ''}${disabled ? ' is-disabled' : ''}`} key={id}>
                      <input type="radio" name="facebook-provider" value={id} checked={provider === id} disabled={disabled || isWorking} onChange={() => setProvider(id)} />
                      <span className="fbw-provider-card-top">
                        <strong>{item.label}</strong>
                        <span className={`fbw-status${item.available ? ' is-ready' : ''}`}>
                          {item.available ? <Check size={13} aria-hidden="true" /> : <AlertCircle size={13} aria-hidden="true" />}
                          {item.available ? 'พร้อม' : 'ยังไม่พร้อม'}
                        </span>
                      </span>
                      <span className="fbw-provider-description">{providerDescription[id]}</span>
                      <span className="fbw-provider-message">{item.version ? `${item.version} · ` : ''}{item.message}</span>
                    </label>
                  )
                })}
              </div>
              <div className="fbw-provider-help">
                <ShieldCheck size={19} aria-hidden="true" />
                <p>{provider === 'mcp'
                  ? 'MCP ไม่ได้เรียกโมเดลเอง: หลังซิงก์ Brief ให้พิมพ์สั่ง Claude/Codex ให้ใช้เครื่องมือ get_facebook_brief และ save_facebook_pack แล้วจึงกดดึงผลลัพธ์'
                  : `${providerMap.get(provider)?.label ?? provider} จะถูกเรียกผ่าน CLI โดยใช้สิทธิ์ของบัญชีที่ล็อกอินอยู่ การใช้งานยังขึ้นกับแพลนและข้อกำหนดของผู้ให้บริการ`}</p>
              </div>
              {provider !== 'mcp' && (
                <fieldset className="fbw-workflow-picker" disabled={isWorking} data-tour="facebook-workflow">
                  <legend>รูปแบบทีม AI</legend>
                  <label className={workflow === 'team' ? 'is-selected' : ''}>
                    <input type="radio" name="facebook-workflow" value="team" checked={workflow === 'team'} onChange={() => setWorkflow('team')} />
                    <Users size={19} aria-hidden="true" />
                    <span><strong>AI Team</strong><small>Strategist → Copywriter → Reviewer ตรวจต่อกัน เหมาะกับงานจริง</small></span>
                  </label>
                  <label className={workflow === 'single' ? 'is-selected' : ''}>
                    <input type="radio" name="facebook-workflow" value="single" checked={workflow === 'single'} onChange={() => setWorkflow('single')} />
                    <Zap size={19} aria-hidden="true" />
                    <span><strong>Quick draft</strong><small>worker เดียว เร็วกว่าและใช้โควตาน้อยกว่า</small></span>
                  </label>
                </fieldset>
              )}
            </section>

            <section className="fbw-section" aria-labelledby="fbw-brief-title" data-tour="facebook-brief">
              <div className="fbw-section-heading">
                <div><span className="fbw-step">02</span><h2 id="fbw-brief-title">Content Brief</h2></div>
                <span className="fbw-required-note">* จำเป็น</span>
              </div>
              <div className="fbw-field-grid">
                <label className="fbw-field fbw-field-wide">
                  <span>แคมเปญ / สินค้า / หัวข้อ *</span>
                  <input value={brief.topic} onChange={(event) => updateBrief('topic', event.target.value)} aria-invalid={Boolean(errors.topic)} aria-describedby={errors.topic ? 'fbw-topic-error' : undefined} maxLength={300} placeholder="เช่น คอร์สการตลาดสำหรับร้านท้องถิ่น" />
                  {errors.topic && <small className="fbw-field-error" id="fbw-topic-error">{errors.topic}</small>}
                </label>
                <label className="fbw-field">
                  <span>กลุ่มเป้าหมาย *</span>
                  <textarea value={brief.audience} onChange={(event) => updateBrief('audience', event.target.value)} aria-invalid={Boolean(errors.audience)} aria-describedby={errors.audience ? 'fbw-audience-error' : undefined} maxLength={1500} rows={3} placeholder="เขียนให้เห็นคนจริง ปัญหาจริง และระดับความรู้" />
                  {errors.audience && <small className="fbw-field-error" id="fbw-audience-error">{errors.audience}</small>}
                </label>
                <label className="fbw-field">
                  <span>เป้าหมายของโพสต์ *</span>
                  <textarea value={brief.objective} onChange={(event) => updateBrief('objective', event.target.value)} aria-invalid={Boolean(errors.objective)} aria-describedby={errors.objective ? 'fbw-objective-error' : undefined} maxLength={1500} rows={3} placeholder="เช่น ให้คนทักมาขอรายละเอียดคอร์ส" />
                  {errors.objective && <small className="fbw-field-error" id="fbw-objective-error">{errors.objective}</small>}
                </label>
                <label className="fbw-field">
                  <span>ข้อเสนอ</span>
                  <textarea value={brief.offer} onChange={(event) => updateBrief('offer', event.target.value)} maxLength={4000} rows={3} placeholder="ใส่เฉพาะสิ่งที่ยืนยันได้ รวมราคาและเงื่อนไข" />
                </label>
                <label className="fbw-field">
                  <span>น้ำเสียงแบรนด์</span>
                  <textarea value={brief.brandVoice} onChange={(event) => updateBrief('brandVoice', event.target.value)} maxLength={2000} rows={3} />
                </label>
                <label className="fbw-field fbw-field-compact">
                  <span>ภาษา *</span>
                  <select value={brief.language} onChange={(event) => updateBrief('language', event.target.value)} aria-invalid={Boolean(errors.language)}>
                    <option value="th">ไทย</option>
                    <option value="en">English</option>
                    <option value="th-en">ไทย + English</option>
                  </select>
                </label>
                <label className="fbw-field fbw-field-wide">
                  <span>รายละเอียดสินค้า / บริการที่ยืนยันแล้ว</span>
                  <textarea value={brief.productDetails} onChange={(event) => updateBrief('productDetails', event.target.value)} maxLength={12000} rows={5} placeholder="ข้อเท็จจริง จุดเด่น ข้อจำกัด วิธีสั่งซื้อ" />
                </label>
                <label className="fbw-field fbw-field-wide">
                  <span>ข้อกำหนดเพิ่มเติม</span>
                  <textarea value={brief.additionalInstructions} onChange={(event) => updateBrief('additionalInstructions', event.target.value)} maxLength={8000} rows={4} placeholder="คำที่ห้ามใช้ ข้อควรระวัง โครงสร้างที่ต้องการ" />
                </label>
              </div>
            </section>

            <section className="fbw-section" aria-labelledby="fbw-evidence-title" data-tour="facebook-evidence">
              <div className="fbw-section-heading">
                <div><span className="fbw-step">03</span><h2 id="fbw-evidence-title">หลักฐานอ้างอิง</h2></div>
                <button className="fbw-button fbw-button-quiet" type="button" onClick={addEvidence} disabled={isWorking || brief.evidence.length >= 30}>
                  <Plus size={17} aria-hidden="true" /> เพิ่มแหล่ง
                </button>
              </div>
              <p className="fbw-section-note">โมเดลจะถือว่าเฉพาข้อความใน “บันทึกข้อเท็จจริง” เป็นหลักฐาน URL อย่างเดียวไม่ได้ยืนยันข้ออ้าง</p>
              {brief.evidence.length === 0 ? (
                <button className="fbw-evidence-empty" type="button" onClick={addEvidence} disabled={isWorking}>
                  <FilePlus2 size={22} aria-hidden="true" />
                  <span><strong>ยังไม่มีแหล่งข้อมูล</strong><small>เพิ่มเอกสาร หน้าขาย หรือบันทึกจากเจ้าของสินค้า</small></span>
                </button>
              ) : (
                <div className="fbw-evidence-list">
                  {brief.evidence.map((source, index) => (
                    <fieldset className="fbw-evidence-row" key={index}>
                      <legend>แหล่งข้อมูล {index + 1}</legend>
                      <div className="fbw-evidence-fields">
                        <label className="fbw-field fbw-field-id">
                          <span>รหัส *</span>
                          <input value={source.id} onChange={(event) => updateEvidence(index, 'id', event.target.value)} aria-invalid={Boolean(errors[`evidence-id-${index}`])} aria-describedby={errors[`evidence-id-${index}`] ? `fbw-evidence-id-error-${index}` : undefined} maxLength={128} />
                          {errors[`evidence-id-${index}`] && <small className="fbw-field-error" id={`fbw-evidence-id-error-${index}`}>{errors[`evidence-id-${index}`]}</small>}
                        </label>
                        <label className="fbw-field">
                          <span>ชื่อแหล่ง *</span>
                          <input value={source.title} onChange={(event) => updateEvidence(index, 'title', event.target.value)} aria-invalid={Boolean(errors[`evidence-title-${index}`])} aria-describedby={errors[`evidence-title-${index}`] ? `fbw-evidence-title-error-${index}` : undefined} maxLength={500} />
                          {errors[`evidence-title-${index}`] && <small className="fbw-field-error" id={`fbw-evidence-title-error-${index}`}>{errors[`evidence-title-${index}`]}</small>}
                        </label>
                        <button className="fbw-icon-button fbw-delete-button" type="button" onClick={() => removeEvidence(index)} disabled={isWorking} aria-label={`ลบแหล่งข้อมูล ${index + 1}`}>
                          <Trash2 size={17} aria-hidden="true" />
                        </button>
                        <label className="fbw-field fbw-field-wide">
                          <span>URL</span>
                          <input type="url" value={source.url} onChange={(event) => updateEvidence(index, 'url', event.target.value)} aria-invalid={Boolean(errors[`evidence-url-${index}`])} aria-describedby={errors[`evidence-url-${index}`] ? `fbw-evidence-url-error-${index}` : undefined} placeholder="https://" maxLength={4096} />
                          {errors[`evidence-url-${index}`] && <small className="fbw-field-error" id={`fbw-evidence-url-error-${index}`}>{errors[`evidence-url-${index}`]}</small>}
                        </label>
                        <label className="fbw-field fbw-field-wide">
                          <span>บันทึกข้อเท็จจริง</span>
                          <textarea value={source.notes} onChange={(event) => updateEvidence(index, 'notes', event.target.value)} rows={3} maxLength={12000} placeholder="คัดเฉพาข้อมูลที่ตรวจสอบแล้วและอนุญาตให้นำมาใช้" />
                        </label>
                      </div>
                    </fieldset>
                  ))}
                </div>
              )}
            </section>

            <div className="fbw-action-bar">
              <div className="fbw-action-message" aria-live="polite" aria-atomic="true">
                {notice && (
                  <span className={`fbw-notice is-${notice.tone}`}>
                    {notice.tone === 'error' ? <AlertCircle size={18} aria-hidden="true" /> : notice.tone === 'success' ? <CheckCircle2 size={18} aria-hidden="true" /> : <FileText size={18} aria-hidden="true" />}
                    {notice.message}
                  </span>
                )}
              </div>
              <div className="fbw-action-buttons" data-tour="facebook-run">
                {busy === 'generate' && activeRunId && (
                  <button className="fbw-button fbw-button-danger" type="button" onClick={() => void cancelGeneration()}>
                    <Square size={16} aria-hidden="true" /> หยุดทีม AI
                  </button>
                )}
                {provider === 'mcp' && (
                  <button className="fbw-button fbw-button-secondary" type="button" onClick={fetchLatest} disabled={isWorking}>
                    {busy === 'fetch' ? <LoaderCircle className="spin" size={18} aria-hidden="true" /> : <Save size={18} aria-hidden="true" />}
                    ดึงผลลัพธ์ล่าสุด
                  </button>
                )}
                <button className="fbw-button fbw-button-primary" type="submit" disabled={isWorking || (provider !== 'mcp' && !providerMap.get(provider)?.available)}>
                  {busy === 'generate' || busy === 'sync' ? <LoaderCircle className="spin" size={18} aria-hidden="true" /> : provider === 'mcp' ? <Link2 size={18} aria-hidden="true" /> : <WandSparkles size={18} aria-hidden="true" />}
                  {provider === 'mcp' ? 'ซิงก์ Brief ไป MCP' : 'สร้าง Content Pack'}
                </button>
              </div>
            </div>
          </form>
        </main>

        <aside className="fbw-results-pane" ref={resultsRef} tabIndex={-1} aria-labelledby="fbw-results-title" data-tour="facebook-output">
          <header className="fbw-results-header">
            <div>
              <span className="fbw-step">04</span>
              <h2 id="fbw-results-title">Content Pack</h2>
            </div>
            {pack && <button className="fbw-button fbw-button-quiet" type="button" onClick={() => void copy(packText(pack), 'Content Pack ทั้งหมด')}><Clipboard size={17} aria-hidden="true" /> คัดลอกทั้งหมด</button>}
          </header>

          {outputStale && pack && (
            <div className="fbw-stale-warning" role="status">
              <TriangleAlert size={19} aria-hidden="true" />
              <span><strong>ผลลัพธ์อาจเก่ากว่า Brief</strong> ตรวจทานก่อนนำไปใช้ หรือสร้างใหม่จาก Brief ปัจจุบัน</span>
            </div>
          )}

          {!pack ? (
            <div className="fbw-results-empty">
              <div className="fbw-empty-icon"><FileText size={28} aria-hidden="true" /></div>
              <h3>ผลลัพธ์จะอยู่ตรงนี้</h3>
              <p>เติม Brief แล้วเลือกสร้างผ่าน CLI หรือซิงก์ไปยัง MCP Companion</p>
              <ul>
                <li>3 Hooks คนละมุม</li>
                <li>โพสต์ยาวและโพสต์สั้น</li>
                <li>Reel, Carousel, CTA และคลังคำตอบ</li>
                <li>ข้อควรระวังที่ต้องให้คนตรวจ</li>
              </ul>
            </div>
          ) : (
            <div className="fbw-output-grid">
              <OutputCard title="Hooks" number={1} copyValue={pack.hooks.join('\n')} onCopy={copy}>
                <ol className="fbw-hook-list">{pack.hooks.map((hook, index) => <li key={index}>{hook}</li>)}</ol>
              </OutputCard>
              <OutputCard title="โพสต์ยาว" number={2} copyValue={pack.longPost} onCopy={copy}><pre>{pack.longPost}</pre></OutputCard>
              <OutputCard title="โพสต์สั้น" number={3} copyValue={pack.shortPost} onCopy={copy}><pre>{pack.shortPost}</pre></OutputCard>
              <OutputCard title="Reel Script" number={4} copyValue={pack.reelScript} onCopy={copy}><pre>{pack.reelScript}</pre></OutputCard>
              <OutputCard title="Carousel" number={5} copyValue={pack.carouselSlides.map((slide, index) => `สไลด์ ${index + 1}: ${slide.headline}\n${slide.body}`).join('\n\n')} onCopy={copy}>
                <div className="fbw-slide-list">{pack.carouselSlides.map((slide, index) => <article key={index}><span>{index + 1}</span><div><h4>{slide.headline}</h4><p>{slide.body}</p></div></article>)}</div>
              </OutputCard>
              <OutputCard title="Call to action" number={6} copyValue={pack.cta} onCopy={copy}><pre>{pack.cta}</pre></OutputCard>
              <OutputCard title="คอมเมนต์แรก" number={7} copyValue={pack.firstComment} onCopy={copy}><pre>{pack.firstComment}</pre></OutputCard>
              <OutputCard title="คลังคำตอบ" number={8} copyValue={pack.replyBank.map((item) => `${item.intent}\n${item.reply}`).join('\n\n')} onCopy={copy}>
                <div className="fbw-reply-list">{pack.replyBank.map((item, index) => <article key={index}><h4>{item.intent}</h4><p>{item.reply}</p></article>)}</div>
              </OutputCard>
              <OutputCard title="ข้อควรระวัง" number={9} copyValue={pack.complianceNotes.join('\n')} onCopy={copy} caution>
                {pack.complianceNotes.length > 0
                  ? <ul className="fbw-compliance-list">{pack.complianceNotes.map((note, index) => <li key={index}>{note}</li>)}</ul>
                  : <p className="fbw-clear-state"><CheckCircle2 size={18} aria-hidden="true" /> โมเดลไม่ได้ระบุข้อควรระวังเพิ่มเติม แต่ผู้ดูแลยังควรตรวจทุกข้ออ้างก่อนโพสต์</p>}
              </OutputCard>

              {(snapshot.groundingSources?.length ?? 0) > 0 && (
                <section className="fbw-grounding" aria-labelledby="fbw-grounding-title">
                  <header><Link2 size={18} aria-hidden="true" /><h3 id="fbw-grounding-title">แหล่งข้อมูลที่โมเดลส่งกลับ</h3></header>
                  <ul>{snapshot.groundingSources?.map((source, index) => {
                    const href = safeExternalURL(source.url)
                    return <li key={`${source.url}-${index}`}>{href ? <a href={href} target="_blank" rel="noreferrer"><span>{source.title || href}</span><ExternalLink size={15} aria-hidden="true" /></a> : <span>{source.title || source.url}</span>}</li>
                  })}</ul>
                </section>
              )}
            </div>
          )}
        </aside>
      </div>
    </div>
  )
}

export default FacebookWorkspace
