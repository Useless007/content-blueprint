import {
  Activity,
  Bot,
  CheckCircle2,
  Circle,
  CircleAlert,
  Clock3,
  Radio,
  TriangleAlert,
  Unplug,
  Workflow,
  type LucideIcon,
} from 'lucide-react'
import {useEffect, useId, useState} from 'react'
import './AIStudio.css'

export type AIStageId = 'strategist' | 'copywriter' | 'reviewer' | 'browserCourier'
export type AIStageStatus = 'idle' | 'queued' | 'working' | 'done' | 'error'
export type AIConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'error'
export type AIStudioProvider = 'claude' | 'codex' | 'gemini' | 'local' | 'none' | (string & {})

export interface AIStageState {
  status: AIStageStatus
  summary?: string
  updatedAt?: string
}

export type AIStudioStages = Readonly<
  Partial<Record<AIStageId, AIStageState | AIStageStatus>>
>

export interface AIActivityMessage {
  id: string
  message: string
  stage?: AIStageId
  timestamp?: string
  tone?: 'info' | 'success' | 'warning' | 'error'
}

export interface AIStudioProps {
  provider: AIStudioProvider
  workflow: string
  stages: AIStudioStages
  connectionStatus: AIConnectionStatus
  activityMessages: ReadonlyArray<AIActivityMessage | string>
  className?: string
}

interface StageDefinition {
  id: AIStageId
  name: string
  shortName: string
  deskLabel: string
}

interface AIHandoff {
  id: string
  from: StageDefinition
  to: StageDefinition
}

const STAGE_DEFINITIONS: ReadonlyArray<StageDefinition> = [
  {
    id: 'strategist',
    name: 'นักวางกลยุทธ์',
    shortName: 'กลยุทธ์',
    deskLabel: 'โต๊ะวางแผน',
  },
  {
    id: 'copywriter',
    name: 'นักเขียนคอนเทนต์',
    shortName: 'นักเขียน',
    deskLabel: 'โต๊ะเขียน',
  },
  {
    id: 'reviewer',
    name: 'ผู้ตรวจทาน',
    shortName: 'ตรวจทาน',
    deskLabel: 'โต๊ะตรวจคุณภาพ',
  },
  {
    id: 'browserCourier',
    name: 'ผู้ส่งงานไปเบราว์เซอร์',
    shortName: 'Browser Courier',
    deskLabel: 'โต๊ะจัดส่ง',
  },
]

const STATUS_META: Record<
  AIStageStatus,
  {label: string; description: string; Icon: LucideIcon}
> = {
  idle: {
    label: 'ว่าง',
    description: 'รอรับงานใหม่ที่ล็อบบี้',
    Icon: Circle,
  },
  queued: {
    label: 'รอคิว',
    description: 'รับงานแล้วและกำลังรอขั้นตอนก่อนหน้า',
    Icon: Clock3,
  },
  working: {
    label: 'กำลังทำงาน',
    description: 'กำลังประมวลผลงานที่โต๊ะประจำ',
    Icon: Activity,
  },
  done: {
    label: 'เสร็จแล้ว',
    description: 'ส่งมอบผลลัพธ์ไปยังจุดรับงานแล้ว',
    Icon: CheckCircle2,
  },
  error: {
    label: 'ต้องตรวจสอบ',
    description: 'หยุดที่จุดช่วยเหลือเพื่อรอการแก้ไข',
    Icon: CircleAlert,
  },
}

const CONNECTION_META: Record<
  AIConnectionStatus,
  {label: string; detail: string; Icon: LucideIcon}
> = {
  disconnected: {
    label: 'ยังไม่เชื่อมต่อ',
    detail: 'ยังไม่พบ Wails event bridge',
    Icon: Unplug,
  },
  connecting: {
    label: 'กำลังเชื่อมต่อ',
    detail: 'กำลังเริ่มงานผ่าน Wails',
    Icon: Radio,
  },
  connected: {
    label: 'เชื่อมต่อแล้ว',
    detail: 'พร้อมรับ event การทำงานจาก Wails',
    Icon: Radio,
  },
  error: {
    label: 'การเชื่อมต่อมีปัญหา',
    detail: 'ตรวจสอบ Wails event bridge',
    Icon: TriangleAlert,
  },
}

const PROVIDER_LABELS: Record<string, string> = {
  claude: 'Claude',
  codex: 'Codex',
  gemini: 'Gemini API',
  local: 'Local Companion',
  none: 'ยังไม่เลือกผู้ให้บริการ',
}

const ACTIVITY_TONE_META: Record<
  NonNullable<AIActivityMessage['tone']>,
  {label: string; Icon: LucideIcon}
> = {
  info: {label: 'ข้อมูล', Icon: Activity},
  success: {label: 'สำเร็จ', Icon: CheckCircle2},
  warning: {label: 'คำเตือน', Icon: TriangleAlert},
  error: {label: 'ผิดพลาด', Icon: CircleAlert},
}

function normalizeStage(stages: AIStudioStages, id: AIStageId): AIStageState {
  const value = stages[id]
  if (!value) return {status: 'idle'}
  return typeof value === 'string' ? {status: value} : value
}

function providerLabel(provider: AIStudioProvider): string {
  const key = String(provider).trim()
  return (PROVIDER_LABELS[key.toLowerCase()] ?? key) || PROVIDER_LABELS.none
}

function formatActivityTime(timestamp?: string): string | null {
  if (!timestamp) return null
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) return timestamp
  return new Intl.DateTimeFormat('th-TH', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(date)
}

function AgentSprite({definition, state}: {definition: StageDefinition; state: AIStageState}) {
  const meta = STATUS_META[state.status]
  const accessibleStatus = state.summary?.trim() || meta.description

  return (
    <div
      className="ai-agent"
      data-agent={definition.id}
      data-status={state.status}
      role="img"
      aria-label={`${definition.name}: ${meta.label}. ${accessibleStatus}`}
    >
      <span className="ai-agent__name" aria-hidden="true">
        {definition.shortName}
      </span>
      <svg
        className="ai-agent__sprite"
        viewBox="0 0 40 48"
        focusable="false"
        aria-hidden="true"
        shapeRendering="crispEdges"
      >
        <rect className="ai-agent__shadow" x="8" y="43" width="24" height="3" rx="1" />
        <rect className="ai-agent__leg" x="12" y="35" width="6" height="8" />
        <rect className="ai-agent__leg" x="22" y="35" width="6" height="8" />
        <rect className="ai-agent__shoe" x="9" y="41" width="9" height="4" />
        <rect className="ai-agent__shoe" x="22" y="41" width="9" height="4" />
        <rect className="ai-agent__body" x="9" y="20" width="22" height="17" rx="2" />
        <rect className="ai-agent__arm" x="5" y="23" width="5" height="12" rx="1" />
        <rect className="ai-agent__arm" x="30" y="23" width="5" height="12" rx="1" />
        <rect className="ai-agent__skin" x="11" y="7" width="18" height="15" rx="2" />
        <rect className="ai-agent__hair" x="10" y="5" width="20" height="6" rx="1" />
        <rect className="ai-agent__hair" x="10" y="9" width="4" height="6" />
        <rect className="ai-agent__eye" x="15" y="13" width="2" height="2" />
        <rect className="ai-agent__eye" x="23" y="13" width="2" height="2" />

        {definition.id === 'strategist' && (
          <>
            <rect className="ai-agent__accessory" x="13" y="25" width="14" height="9" rx="1" />
            <rect className="ai-agent__paper" x="15" y="27" width="10" height="5" />
            <rect className="ai-agent__ink" x="17" y="28" width="6" height="1" />
            <rect className="ai-agent__ink" x="17" y="30" width="4" height="1" />
          </>
        )}

        {definition.id === 'copywriter' && (
          <>
            <rect className="ai-agent__paper" x="13" y="25" width="12" height="10" rx="1" />
            <path className="ai-agent__accessory" d="M25 31l7-7 2 2-7 7-3 1z" />
            <rect className="ai-agent__ink" x="15" y="28" width="7" height="1" />
            <rect className="ai-agent__ink" x="15" y="31" width="5" height="1" />
          </>
        )}

        {definition.id === 'reviewer' && (
          <>
            <rect className="ai-agent__glasses" x="13" y="11" width="6" height="5" rx="1" />
            <rect className="ai-agent__glasses" x="21" y="11" width="6" height="5" rx="1" />
            <rect className="ai-agent__glasses" x="19" y="13" width="2" height="1" />
            <path className="ai-agent__accessory" d="M13 27h11v7H13zM24 31l5 5 2-2-5-5z" />
          </>
        )}

        {definition.id === 'browserCourier' && (
          <>
            <rect className="ai-agent__parcel" x="13" y="25" width="15" height="11" rx="1" />
            <rect className="ai-agent__parcel-tape" x="19" y="25" width="3" height="11" />
            <path className="ai-agent__parcel-mark" d="M15 29h3v-2l4 3-4 3v-2h-3z" />
          </>
        )}
      </svg>
      {state.status === 'done' && (
        <span className="ai-agent__handoff-file" aria-hidden="true">
          <i />
        </span>
      )}
      <span className="ai-agent__state" aria-hidden="true">
        {meta.label}
      </span>
    </div>
  )
}

function StatusListItem({definition, state}: {definition: StageDefinition; state: AIStageState}) {
  const meta = STATUS_META[state.status]
  const Icon = meta.Icon

  return (
    <li className="ai-studio__agent-row" data-status={state.status}>
      <span className="ai-studio__agent-icon" data-agent={definition.id} aria-hidden="true">
        <Bot size={17} strokeWidth={2} />
      </span>
      <span className="ai-studio__agent-copy">
        <strong>{definition.name}</strong>
        <small>{state.summary?.trim() || meta.description}</small>
      </span>
      <span className="ai-studio__status-pill">
        <Icon size={15} strokeWidth={2.2} aria-hidden="true" />
        {meta.label}
      </span>
    </li>
  )
}

export function AIStudio({
  provider,
  workflow,
  stages,
  connectionStatus,
  activityMessages,
  className = '',
}: AIStudioProps) {
  const titleId = useId()
  const teamTitleId = useId()
  const activityTitleId = useId()
  const connection = CONNECTION_META[connectionStatus]
  const ConnectionIcon = connection.Icon
  const normalizedStages = STAGE_DEFINITIONS.map((definition) => ({
    definition,
    state: normalizeStage(stages, definition.id),
  }))
  const activeAgents = normalizedStages.filter(({state}) => state.status === 'working')
  const completeCount = normalizedStages.filter(({state}) => state.status === 'done').length
  const studioClassName = ['ai-studio', className].filter(Boolean).join(' ')
  const liveSummary = normalizedStages
    .map(({definition, state}) => `${definition.name} ${STATUS_META[state.status].label}`)
    .join(', ')
  const stageSignature = normalizedStages
    .map(({definition, state}) => `${definition.id}:${state.status}:${state.updatedAt ?? ''}`)
    .join('|')
  const [handoff, setHandoff] = useState<AIHandoff | null>(null)

  useEffect(() => {
    const workingIndex = normalizedStages.findIndex(({state}) => state.status === 'working')
    if (workingIndex < 1) {
      setHandoff(null)
      return
    }
    const previous = normalizedStages
      .slice(0, workingIndex)
      .reverse()
      .find(({state}) => state.status === 'done')
    if (!previous) {
      setHandoff(null)
      return
    }
    const next = normalizedStages[workingIndex]
    const nextHandoff = {
      id: `${previous.definition.id}-${next.definition.id}-${next.state.updatedAt ?? stageSignature}`,
      from: previous.definition,
      to: next.definition,
    }
    setHandoff(nextHandoff)
    const timer = window.setTimeout(() => setHandoff(null), 2_200)
    return () => window.clearTimeout(timer)
  }, [stageSignature])

  return (
    <section className={studioClassName} aria-labelledby={titleId}>
      <header className="ai-studio__header">
        <div className="ai-studio__title-block">
          <span className="ai-studio__eyebrow">SUBAGENT WORKSPACE</span>
          <h2 id={titleId}>AI Studio</h2>
          <p>ดูทีม AI รับช่วงงาน วางแผน ตรวจ และส่งต่อไปยังเบราว์เซอร์แบบเป็นขั้นตอน</p>
        </div>
        <div
          className="ai-studio__connection"
          data-connection={connectionStatus}
          role="status"
          aria-atomic="true"
        >
          <ConnectionIcon size={18} strokeWidth={2.2} aria-hidden="true" />
          <span>
            <strong>{connection.label}</strong>
            <small>{connection.detail}</small>
          </span>
        </div>
      </header>

      <div className="ai-studio__run-strip" aria-label="งานที่กำลังติดตาม">
        <div>
          <Bot size={18} aria-hidden="true" />
          <span>ผู้ให้บริการ</span>
          <strong>{providerLabel(provider)}</strong>
        </div>
        <div>
          <Workflow size={18} aria-hidden="true" />
          <span>เวิร์กโฟลว์</span>
          <strong title={workflow}>{workflow.trim() || 'ยังไม่มีงานที่กำลังทำ'}</strong>
        </div>
        <div className="ai-studio__run-metric">
          <CheckCircle2 size={18} aria-hidden="true" />
          <span>ความคืบหน้า</span>
          <strong>{completeCount}/4 ขั้น</strong>
        </div>
      </div>

      <div className="ai-studio__body">
        <div className="ai-studio__office-column">
          <div className="ai-studio__section-heading">
            <div>
              <h3>ห้องทำงานของทีม</h3>
              <p>
                {activeAgents.length > 0
                  ? `${activeAgents.map(({definition}) => definition.shortName).join(', ')} กำลังทำงาน`
                  : 'ทีมพร้อมรับคำสั่งถัดไป'}
              </p>
            </div>
            <span className="ai-studio__active-count">
              <Activity size={15} aria-hidden="true" />
              ทำงาน {activeAgents.length} คน
            </span>
          </div>

          <div className="ai-studio__floor" aria-label="แผนผังตำแหน่งของ AI subagent">
            <div className="ai-studio__tiles" aria-hidden="true">
              {Array.from({length: 96}, (_, index) => (
                <span key={index} />
              ))}
            </div>

            <div className="ai-office-zone ai-office-zone--lobby" aria-hidden="true">
              <span>LOBBY</span>
              <i />
              <i />
              <i />
            </div>

            {STAGE_DEFINITIONS.map((definition) => (
              <div
                key={definition.id}
                className="ai-office-desk"
                data-agent={definition.id}
                aria-hidden="true"
              >
                <span className="ai-office-desk__screen" />
                <span className="ai-office-desk__top" />
                <small>{definition.deskLabel}</small>
              </div>
            ))}

            <div className="ai-office-zone ai-office-zone--delivery" aria-hidden="true">
              <span>ส่งมอบ</span>
              <svg viewBox="0 0 36 36" shapeRendering="crispEdges">
                <rect x="6" y="8" width="24" height="21" rx="2" />
                <path d="M10 17h9v-4l8 6-8 6v-4h-9z" />
              </svg>
            </div>

            <div className="ai-office-zone ai-office-zone--help" aria-hidden="true">
              <CircleAlert size={18} />
              <span>HELP</span>
            </div>

            {handoff && (
              <div
                key={handoff.id}
                className="ai-handoff"
                data-from={handoff.from.id}
                data-to={handoff.to.id}
                role="status"
                aria-label={`${handoff.from.name} กำลังส่งงานให้ ${handoff.to.name}`}
              >
                <span className="ai-handoff__file" aria-hidden="true">
                  <i />
                </span>
                <small aria-hidden="true">
                  {handoff.from.shortName} → {handoff.to.shortName}
                </small>
              </div>
            )}

            {normalizedStages.map(({definition, state}) => (
              <AgentSprite key={definition.id} definition={definition} state={state} />
            ))}

            <p className="sr-only" role="status" aria-live="polite" aria-atomic="true">
              สถานะทีม AI: {liveSummary}
            </p>
          </div>
        </div>

        <aside className="ai-studio__team-panel" aria-labelledby={teamTitleId}>
          <div className="ai-studio__section-heading ai-studio__section-heading--compact">
            <div>
              <h3 id={teamTitleId}>ทีม Subagent</h3>
              <p>สถานะและหน้าที่ล่าสุดของแต่ละคน</p>
            </div>
          </div>
          <ol className="ai-studio__agent-list">
            {normalizedStages.map(({definition, state}) => (
              <StatusListItem key={definition.id} definition={definition} state={state} />
            ))}
          </ol>
        </aside>
      </div>

      <section className="ai-studio__activity" aria-labelledby={activityTitleId}>
        <div className="ai-studio__section-heading ai-studio__section-heading--compact">
          <div>
            <h3 id={activityTitleId}>บันทึกการทำงาน</h3>
            <p>ข้อความล่าสุดจาก companion และ AI CLI</p>
          </div>
          <span className="ai-studio__message-count">{activityMessages.length} รายการ</span>
        </div>

        {activityMessages.length > 0 ? (
          <ol className="ai-studio__timeline" aria-label="ลำดับเหตุการณ์ล่าสุด">
            {activityMessages.map((item, index) => {
              const activity: AIActivityMessage =
                typeof item === 'string'
                  ? {id: `message-${index}`, message: item}
                  : item
              const stageDefinition = activity.stage
                ? STAGE_DEFINITIONS.find((definition) => definition.id === activity.stage)
                : undefined
              const formattedTime = formatActivityTime(activity.timestamp)
              const tone = activity.tone ?? 'info'
              const ToneIcon = ACTIVITY_TONE_META[tone].Icon

              return (
                <li key={activity.id || `message-${index}`} data-tone={tone}>
                  <span className="ai-studio__timeline-marker" aria-hidden="true">
                    <ToneIcon size={12} strokeWidth={2.4} />
                  </span>
                  <div>
                    <p>{activity.message}</p>
                    <span>
                      <small className="ai-studio__timeline-tone">
                        {ACTIVITY_TONE_META[tone].label}
                      </small>
                      {stageDefinition && <strong>{stageDefinition.name}</strong>}
                      {formattedTime && (
                        <time dateTime={activity.timestamp}>{formattedTime}</time>
                      )}
                    </span>
                  </div>
                </li>
              )
            })}
          </ol>
        ) : (
          <div className="ai-studio__empty-activity">
            <Clock3 size={20} aria-hidden="true" />
            <div>
              <strong>ยังไม่มีข้อความจากทีม</strong>
              <p>เมื่อเริ่มเวิร์กโฟลว์ เหตุการณ์สำคัญจะปรากฏที่นี่</p>
            </div>
          </div>
        )}
      </section>
    </section>
  )
}
