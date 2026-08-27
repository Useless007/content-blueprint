import {BookOpenCheck, CheckCircle2, CircleHelp, Clock3, RotateCcw, X} from 'lucide-react'
import {
  EVENTS,
  STATUS,
  useJoyride,
  type EventData,
  type Step,
} from 'react-joyride'
import {useCallback, useEffect, useMemo, useRef, useState} from 'react'
import {
  getMission,
  ONBOARDING_MISSIONS,
  ONBOARDING_VERSION,
  type TourDestination,
} from './missions'
import './onboarding.css'

const STORAGE_KEY = `content-blueprint:onboarding:v${ONBOARDING_VERSION}`

interface StoredOnboardingState {
  version: number
  welcomed: boolean
  activeMissionId: string
  lastStepByMission: Record<string, number>
  completedMissionIds: string[]
}

const EMPTY_STATE: StoredOnboardingState = {
  version: ONBOARDING_VERSION,
  welcomed: false,
  activeMissionId: '',
  lastStepByMission: {},
  completedMissionIds: [],
}

function readStoredState(): StoredOnboardingState {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return EMPTY_STATE
    const parsed = JSON.parse(raw) as Partial<StoredOnboardingState>
    if (parsed.version !== ONBOARDING_VERSION) return EMPTY_STATE
    return {
      version: ONBOARDING_VERSION,
      welcomed: Boolean(parsed.welcomed),
      activeMissionId: typeof parsed.activeMissionId === 'string' ? parsed.activeMissionId : '',
      lastStepByMission: parsed.lastStepByMission && typeof parsed.lastStepByMission === 'object'
        ? parsed.lastStepByMission
        : {},
      completedMissionIds: Array.isArray(parsed.completedMissionIds)
        ? parsed.completedMissionIds.filter((id): id is string => typeof id === 'string')
        : [],
    }
  } catch {
    return EMPTY_STATE
  }
}

function writeStoredState(value: StoredOnboardingState) {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
  } catch {
    // The guide remains usable even when local persistence is unavailable.
  }
}

function waitForPaint(): Promise<void> {
  return new Promise((resolve) => {
    window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve()))
  })
}

function focusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(
    'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
  )).filter((element) => !element.hasAttribute('hidden'))
}

export interface OnboardingProviderProps {
  onNavigate: (destination: TourDestination) => void | Promise<void>
}

export function OnboardingProvider({onNavigate}: OnboardingProviderProps) {
  const initialStateRef = useRef(readStoredState())
  const helpButtonRef = useRef<HTMLButtonElement>(null)
  const dialogRef = useRef<HTMLDivElement>(null)
  const pendingStartRef = useRef<number | null>(null)
  const [storedState, setStoredState] = useState(initialStateRef.current)
  const [centerOpen, setCenterOpen] = useState(!initialStateRef.current.welcomed)
  const [activeMissionId, setActiveMissionId] = useState('')
  const [tourMessage, setTourMessage] = useState('')
  const [reducedMotion, setReducedMotion] = useState(false)

  const persist = useCallback((updater: (current: StoredOnboardingState) => StoredOnboardingState) => {
    setStoredState((current) => {
      const next = updater(current)
      writeStoredState(next)
      return next
    })
  }, [])

  const activeMission = useMemo(() => getMission(activeMissionId), [activeMissionId])
  const steps = useMemo<Step[]>(() => activeMission?.steps.map((step) => ({
    id: step.id,
    target: step.target,
    title: step.title,
    content: <p className="onboarding-step-copy">{step.body}</p>,
    placement: step.placement ?? 'bottom',
    blockTargetInteraction: step.blockTargetInteraction ?? false,
    before: async () => {
      await onNavigate(step.destination)
      await waitForPaint()
    },
    data: {missionId: activeMission.id, stepId: step.id},
  })) ?? [], [activeMission, onNavigate])

  const handleTourEvent = useCallback((event: EventData) => {
    const missionId = typeof event.step?.data?.missionId === 'string'
      ? event.step.data.missionId
      : activeMissionId
    if (!missionId) return

    if (event.type === EVENTS.STEP_BEFORE || event.type === EVENTS.TOOLTIP) {
      persist((current) => ({
        ...current,
        activeMissionId: missionId,
        lastStepByMission: {...current.lastStepByMission, [missionId]: event.index},
      }))
    }

    if (event.type === EVENTS.TARGET_NOT_FOUND) {
      setTourMessage('บางจุดยังไม่พร้อมแสดง ระบบข้ามจุดนั้นให้แล้ว คุณเริ่มภารกิจนี้ซ้ำได้จากศูนย์ช่วยเหลือ')
    }

    if (event.type === EVENTS.TOUR_END) {
      const completed = event.status === STATUS.FINISHED
      persist((current) => ({
        ...current,
        activeMissionId: '',
        lastStepByMission: completed
          ? {...current.lastStepByMission, [missionId]: 0}
          : {...current.lastStepByMission, [missionId]: event.index},
        completedMissionIds: completed && !current.completedMissionIds.includes(missionId)
          ? [...current.completedMissionIds, missionId]
          : current.completedMissionIds,
      }))
      setActiveMissionId('')
      if (completed) {
        setTourMessage('เรียนภารกิจนี้ครบแล้ว คุณเปิดทบทวนได้ทุกเมื่อ')
        setCenterOpen(true)
      }
    }
  }, [activeMissionId, persist])

  const {Tour, controls} = useJoyride({
    steps,
    continuous: true,
    scrollToFirstStep: true,
    onEvent: handleTourEvent,
    locale: {
      back: 'ย้อนกลับ',
      close: 'ปิด',
      last: 'จบภารกิจ',
      next: 'ถัดไป',
      nextWithProgress: 'ถัดไป ({current}/{total})',
      open: 'เปิดคำแนะนำ',
      skip: 'จบภายหลัง',
    },
    options: {
      buttons: ['back', 'skip', 'primary'],
      closeButtonAction: 'skip',
      dismissKeyAction: false,
      overlayClickAction: false,
      showProgress: true,
      skipBeacon: true,
      targetWaitTimeout: 4500,
      beforeTimeout: 5000,
      loaderDelay: 200,
      scrollDuration: reducedMotion ? 0 : 260,
      scrollOffset: 82,
      spotlightPadding: 8,
      spotlightRadius: 12,
      primaryColor: '#0f766e',
      backgroundColor: '#fffdf8',
      textColor: '#17313a',
      overlayColor: 'rgba(10, 24, 32, 0.68)',
      zIndex: 5000,
    },
    styles: {
      tooltip: {
        border: '1px solid #bdd4d2',
        borderRadius: 18,
        boxShadow: '0 20px 55px rgba(10, 24, 32, 0.24)',
        fontFamily: 'Nunito, system-ui, sans-serif',
      },
      tooltipTitle: {fontSize: 18, fontWeight: 850, lineHeight: 1.3},
      tooltipContent: {fontSize: 14, lineHeight: 1.65, padding: '8px 4px 16px'},
      buttonPrimary: {borderRadius: 10, fontWeight: 800, minHeight: 40, padding: '8px 14px'},
      buttonBack: {color: '#31545c', fontWeight: 800, minHeight: 40},
      buttonSkip: {color: '#526b72', fontWeight: 750, minHeight: 40},
      beaconInner: reducedMotion ? {animation: 'none'} : {},
      beaconOuter: reducedMotion ? {animation: 'none'} : {},
    },
  })

  useEffect(() => {
    const media = window.matchMedia('(prefers-reduced-motion: reduce)')
    const update = () => setReducedMotion(media.matches)
    update()
    media.addEventListener?.('change', update)
    return () => media.removeEventListener?.('change', update)
  }, [])

  useEffect(() => {
    if (!centerOpen) return
    const dialog = dialogRef.current
    if (!dialog) return
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const first = focusableElements(dialog)[0]
    first?.focus()

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        setCenterOpen(false)
        persist((current) => ({...current, welcomed: true}))
        window.requestAnimationFrame(() => helpButtonRef.current?.focus())
        return
      }
      if (event.key !== 'Tab') return
      const focusable = focusableElements(dialog)
      if (focusable.length === 0) return
      const firstElement = focusable[0]
      const lastElement = focusable[focusable.length - 1]
      if (event.shiftKey && document.activeElement === firstElement) {
        event.preventDefault()
        lastElement.focus()
      } else if (!event.shiftKey && document.activeElement === lastElement) {
        event.preventDefault()
        firstElement.focus()
      }
    }
    dialog.addEventListener('keydown', handleKeyDown)
    return () => {
      dialog.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = previousOverflow
      if (previousFocus && document.contains(previousFocus)) previousFocus.focus()
    }
  }, [centerOpen, persist])

  useEffect(() => {
    const startIndex = pendingStartRef.current
    if (!activeMission || startIndex === null || steps.length === 0) return
    pendingStartRef.current = null
    let cancelled = false
    void (async () => {
      await onNavigate(activeMission.steps[Math.min(startIndex, activeMission.steps.length - 1)].destination)
      await waitForPaint()
      if (!cancelled) controls.start(Math.min(startIndex, activeMission.steps.length - 1))
    })()
    return () => { cancelled = true }
  }, [activeMission, controls, onNavigate, steps.length])

  const closeCenter = useCallback(() => {
    setCenterOpen(false)
    persist((current) => ({...current, welcomed: true}))
  }, [persist])

  const startMission = useCallback((missionId: string, restart = false) => {
    const mission = getMission(missionId)
    if (!mission) return
    const savedIndex = storedState.lastStepByMission[missionId] ?? 0
    const startIndex = restart ? 0 : Math.max(0, Math.min(savedIndex, mission.steps.length - 1))
    pendingStartRef.current = startIndex
    setTourMessage('')
    setActiveMissionId(missionId)
    setCenterOpen(false)
    persist((current) => ({...current, welcomed: true, activeMissionId: missionId}))
  }, [persist, storedState.lastStepByMission])

  return (
    <>
      {Tour}
      <button
        ref={helpButtonRef}
        className="onboarding-help-button"
        type="button"
        data-tour="onboarding-help"
        onClick={() => setCenterOpen(true)}
        aria-haspopup="dialog"
        aria-expanded={centerOpen}
      >
        <CircleHelp size={19} aria-hidden="true" />
        วิธีใช้
      </button>

      {centerOpen && (
        <div className="onboarding-center-overlay" onMouseDown={(event) => { if (event.target === event.currentTarget) closeCenter() }}>
          <div
            ref={dialogRef}
            className="onboarding-center"
            role="dialog"
            aria-modal="true"
            aria-labelledby="onboarding-center-title"
            aria-describedby="onboarding-center-description"
          >
            <header className="onboarding-center-header">
              <div className="onboarding-center-mark" aria-hidden="true"><BookOpenCheck size={23} /></div>
              <div>
                <span className="onboarding-eyebrow">คู่มือในแอป</span>
                <h2 id="onboarding-center-title">เลือกงานที่อยากลองทำ</h2>
                <p id="onboarding-center-description">แต่ละภารกิจพาไปยังหน้าที่เกี่ยวข้อง คุณจบภายหลังแล้วกลับมาต่อได้</p>
              </div>
              <button className="onboarding-close" type="button" onClick={closeCenter} aria-label="ปิดศูนย์วิธีใช้"><X size={19} aria-hidden="true" /></button>
            </header>

            {tourMessage && <div className="onboarding-message" role="status">{tourMessage}</div>}

            <div className="onboarding-mission-list">
              {ONBOARDING_MISSIONS.map((mission) => {
                const completed = storedState.completedMissionIds.includes(mission.id)
                const savedIndex = storedState.lastStepByMission[mission.id] ?? 0
                const hasProgress = !completed && savedIndex > 0
                return (
                  <article className="onboarding-mission" key={mission.id}>
                    <div className="onboarding-mission-copy">
                      <div className="onboarding-mission-title-row">
                        <h3>{mission.title}</h3>
                        {completed && <span className="onboarding-complete"><CheckCircle2 size={14} aria-hidden="true" />เรียนแล้ว</span>}
                      </div>
                      <p>{mission.description}</p>
                      <span className="onboarding-duration"><Clock3 size={14} aria-hidden="true" />{mission.duration} · {mission.steps.length} ขั้น</span>
                    </div>
                    <div className="onboarding-mission-actions">
                      <button className="onboarding-start" type="button" onClick={() => startMission(mission.id, completed)}>
                        {completed ? <><RotateCcw size={15} aria-hidden="true" />ทบทวน</> : hasProgress ? 'เรียนต่อ' : 'เริ่ม'}
                      </button>
                    </div>
                  </article>
                )
              })}
            </div>
          </div>
        </div>
      )}
    </>
  )
}
