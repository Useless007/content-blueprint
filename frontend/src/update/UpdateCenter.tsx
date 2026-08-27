import {
  BadgeCheck,
  CheckCircle2,
  CloudDownload,
  Download,
  ExternalLink,
  Info,
  LoaderCircle,
  RefreshCw,
  Rocket,
  ShieldAlert,
  ShieldCheck,
  X,
} from 'lucide-react'
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import {wailsApi} from '../lib/wails'
import {
  isAutomaticUpdateCheckDue,
  isTrustedReleaseURL,
  openReleaseURL,
  recordAutomaticUpdateCheck,
  subscribeToUpdateProgress,
  UPDATE_AUTOMATIC_DELAY_MS,
} from './runtime'
import type {
  UpdateFailureAction,
  UpdateInfo,
  UpdatePhase,
  UpdateProgress,
} from './types'
import './update-center.css'

const VERSION_PATTERN = /^\d+\.\d+\.\d+$/

function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim()) return error.message.trim()
  if (typeof error === 'string' && error.trim()) return error.trim()
  return 'ดำเนินการไม่สำเร็จ กรุณาลองอีกครั้ง'
}

function phaseFromInfo(info: UpdateInfo): UpdatePhase {
  if (info.state === 'up_to_date') return 'up_to_date'
  if (info.state === 'ready') return 'ready'
  if (info.state === 'downloading') return 'downloading'
  return 'available'
}

function formatReleaseDate(value?: string): string {
  if (!value) return 'ไม่ระบุวันที่เผยแพร่'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'ไม่ระบุวันที่เผยแพร่'
  return new Intl.DateTimeFormat('th-TH', {dateStyle: 'medium'}).format(date)
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 MB'
  return `${(value / (1024 * 1024)).toLocaleString('th-TH', {maximumFractionDigits: 1})} MB`
}

interface UpdateDialogFrameProps {
  busy: boolean
  confirming: boolean
  onCancelConfirmation: () => void
  onClose: () => void
  children: ReactNode
}

function UpdateDialogFrame({
  busy,
  confirming,
  onCancelConfirmation,
  onClose,
  children,
}: UpdateDialogFrameProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const busyRef = useRef(busy)
  const confirmingRef = useRef(confirming)
  const closeRef = useRef(onClose)
  const cancelConfirmationRef = useRef(onCancelConfirmation)

  useEffect(() => { busyRef.current = busy }, [busy])
  useEffect(() => { confirmingRef.current = confirming }, [confirming])
  useEffect(() => { closeRef.current = onClose }, [onClose])
  useEffect(() => { cancelConfirmationRef.current = onCancelConfirmation }, [onCancelConfirmation])

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    window.requestAnimationFrame(() => dialogRef.current?.focus())

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (busyRef.current) return
        event.preventDefault()
        if (confirmingRef.current) cancelConfirmationRef.current()
        else closeRef.current()
        return
      }
      if (event.key !== 'Tab' || !dialogRef.current) return
      const focusable = Array.from(dialogRef.current.querySelectorAll<HTMLElement>(
        'button:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])',
      )).filter((element) => element.getAttribute('aria-hidden') !== 'true')
      if (!focusable.length) {
        event.preventDefault()
        dialogRef.current.focus()
        return
      }
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && (document.activeElement === first || document.activeElement === dialogRef.current)) {
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
  }, [])

  useEffect(() => {
    if (!confirming) return
    window.requestAnimationFrame(() => {
      dialogRef.current?.querySelector<HTMLElement>('[data-update-confirm-cancel]')?.focus()
    })
  }, [confirming])

  return (
    <div
      className="update-modal-overlay"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !busy) onClose()
      }}
    >
      <div
        className="update-dialog"
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="update-dialog-title"
        aria-describedby="update-dialog-description"
        aria-busy={busy}
        tabIndex={-1}
      >
        {children}
      </div>
    </div>
  )
}

export function UpdateCenter() {
  const [phase, setPhase] = useState<UpdatePhase>('idle')
  const [info, setInfo] = useState<UpdateInfo | null>(null)
  const [progress, setProgress] = useState<UpdateProgress | null>(null)
  const [error, setError] = useState('')
  const [failureAction, setFailureAction] = useState<UpdateFailureAction>('check')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [dismissedVersion, setDismissedVersion] = useState('')
  const [announcement, setAnnouncement] = useState('')
  const automaticCheckStarted = useRef(false)

  const busy = phase === 'checking' || phase === 'downloading' || phase === 'installing'
  const trustedReleaseURL = isTrustedReleaseURL(info?.releaseUrl) ? info.releaseUrl : ''
  const hasUpdate = Boolean(info && (phase === 'available' || phase === 'downloading' || phase === 'ready'))
  const showBanner = Boolean(
    hasUpdate
    && info?.latestVersion
    && dismissedVersion !== info.latestVersion,
  )

  const applyInfo = useCallback((next: UpdateInfo) => {
    setInfo(next)
    const nextPhase = phaseFromInfo(next)
    setPhase(nextPhase)
    setError('')
    if (nextPhase === 'up_to_date') {
      setAnnouncement(`Content Blueprint ${next.currentVersion} เป็นเวอร์ชันล่าสุดแล้ว`)
    } else if (nextPhase === 'ready') {
      setAnnouncement(`ดาวน์โหลดเวอร์ชัน ${next.latestVersion} และตรวจ SHA-256 เรียบร้อยแล้ว`)
    } else if (nextPhase === 'available') {
      setDismissedVersion('')
      setAnnouncement(`มี Content Blueprint เวอร์ชัน ${next.latestVersion} พร้อมให้อัปเดต`)
    }
  }, [])

  const checkForUpdates = useCallback(async (automatic = false) => {
    if (!wailsApi.isAvailable()) {
      if (!automatic) {
        setError('เปิดผ่านแอป Content Blueprint บน Windows เพื่อเช็กอัปเดตจาก GitHub')
        setFailureAction('check')
        setPhase('error')
      }
      return
    }

    // Any user-initiated check satisfies the pending startup check for this
    // mount, preventing the delayed automatic request from racing it.
    automaticCheckStarted.current = true
    if (automatic) {
      // Throttle attempts, not only successful responses. Otherwise an offline
      // machine would retry on every launch and violate the once-per-day promise.
      recordAutomaticUpdateCheck()
    }
    setPhase('checking')
    setError('')
    setConfirming(false)
    setAnnouncement('กำลังตรวจหาอัปเดตจาก GitHub Releases')
    try {
      const next = await wailsApi.checkForUpdates()
      applyInfo(next)
    } catch (caught) {
      if (automatic) {
        setPhase('idle')
        setAnnouncement('')
        return
      }
      setFailureAction('check')
      setError(errorMessage(caught))
      setPhase('error')
      setAnnouncement('ตรวจอัปเดตไม่สำเร็จ กรุณาลองอีกครั้ง')
    }
  }, [applyInfo])

  useEffect(() => {
    if (automaticCheckStarted.current || !wailsApi.isAvailable() || !isAutomaticUpdateCheckDue()) return
    const timer = window.setTimeout(() => {
      if (automaticCheckStarted.current || !isAutomaticUpdateCheckDue()) return
      void checkForUpdates(true)
    }, UPDATE_AUTOMATIC_DELAY_MS)
    return () => window.clearTimeout(timer)
  }, [checkForUpdates])

  useEffect(() => subscribeToUpdateProgress((next) => {
    setProgress(next)
    setPhase('downloading')
    const progressText = next.totalBytes > 0
      ? `${next.percent}% หรือ ${formatBytes(next.downloadedBytes)} จาก ${formatBytes(next.totalBytes)}`
      : formatBytes(next.downloadedBytes)
    setAnnouncement(`กำลังดาวน์โหลดเวอร์ชัน ${next.version}: ${progressText}`)
  }), [])

  const openCenter = () => {
    setDialogOpen(true)
    setConfirming(false)
    if (phase === 'idle' || phase === 'up_to_date' || phase === 'error') {
      void checkForUpdates(false)
    }
  }

  const closeCenter = useCallback(() => {
    if (busy) return
    setDialogOpen(false)
    setConfirming(false)
  }, [busy])

  const downloadUpdate = useCallback(async () => {
    const version = info?.latestVersion ?? ''
    if (!VERSION_PATTERN.test(version)) {
      setFailureAction('download')
      setError('เลขเวอร์ชันจาก release ไม่ถูกต้อง จึงยังไม่เริ่มดาวน์โหลด')
      setPhase('error')
      return
    }
    setFailureAction('download')
    setError('')
    setProgress({version, downloadedBytes: 0, totalBytes: 0, percent: 0})
    setPhase('downloading')
    setAnnouncement(`เริ่มดาวน์โหลดเวอร์ชัน ${version}`)
    try {
      const next = await wailsApi.downloadUpdate(version)
      setProgress((current) => ({
        version,
        downloadedBytes: current?.totalBytes || current?.downloadedBytes || 0,
        totalBytes: current?.totalBytes || 0,
        percent: 100,
      }))
      applyInfo(next)
    } catch (caught) {
      setError(errorMessage(caught))
      setPhase('error')
      setAnnouncement('ดาวน์โหลดอัปเดตไม่สำเร็จ กรุณาลองอีกครั้ง')
    }
  }, [applyInfo, info?.latestVersion])

  const launchInstaller = useCallback(async () => {
    const version = info?.latestVersion ?? ''
    if (!VERSION_PATTERN.test(version)) return
    setFailureAction('launch')
    setError('')
    setPhase('installing')
    setAnnouncement(`กำลังเปิดตัวติดตั้งเวอร์ชัน ${version}`)
    try {
      await wailsApi.launchDownloadedUpdate(version)
      setAnnouncement('เปิดตัวติดตั้งที่ตรวจสอบแล้วเรียบร้อย Content Blueprint กำลังปิด')
    } catch (caught) {
      setError(errorMessage(caught))
      setConfirming(false)
      setPhase('error')
      setAnnouncement('เปิดตัวติดตั้งไม่สำเร็จ กรุณาลองอีกครั้ง')
    }
  }, [info?.latestVersion])

  const retry = () => {
    if (failureAction === 'download') void downloadUpdate()
    else if (failureAction === 'launch') void launchInstaller()
    else void checkForUpdates(false)
  }

  const launcherLabel = useMemo(() => {
    if (phase === 'checking') return 'กำลังตรวจอัปเดต…'
    if (phase === 'available' && info) return `มีอัปเดต v${info.latestVersion}`
    if (phase === 'downloading') return 'กำลังดาวน์โหลด…'
    if (phase === 'ready' && info) return `พร้อมติดตั้ง v${info.latestVersion}`
    if (phase === 'installing') return 'กำลังเปิดตัวติดตั้ง…'
    return 'ตรวจอัปเดต'
  }, [info, phase])

  const progressPercent = Math.min(100, Math.max(0, progress?.percent ?? 0))
  const progressText = progress?.totalBytes
    ? `${progressPercent}% · ${formatBytes(progress.downloadedBytes)} จาก ${formatBytes(progress.totalBytes)}`
    : progress && progress.downloadedBytes > 0
      ? `ดาวน์โหลดแล้ว ${formatBytes(progress.downloadedBytes)}`
      : 'กำลังเตรียมการดาวน์โหลดที่ปลอดภัย…'

  return (
    <div className="update-center-root">
      <div className="update-live-region sr-only" role="status" aria-live="polite" aria-atomic="true">
        {announcement}
      </div>

      <button
        className={`update-launcher update-launcher--${phase}`}
        type="button"
        onClick={openCenter}
        aria-haspopup="dialog"
        aria-expanded={dialogOpen}
        disabled={phase === 'installing'}
      >
        {phase === 'checking' || phase === 'downloading' || phase === 'installing'
          ? <LoaderCircle className="spin" size={17} aria-hidden="true" />
          : phase === 'ready'
            ? <BadgeCheck size={17} aria-hidden="true" />
            : phase === 'available'
              ? <CloudDownload size={17} aria-hidden="true" />
              : <RefreshCw size={17} aria-hidden="true" />}
        <span>{launcherLabel}</span>
        {(phase === 'available' || phase === 'ready') && <span className="update-launcher-dot" aria-hidden="true" />}
      </button>

      {showBanner && info && (
        <aside className={`update-banner ${phase === 'ready' ? 'is-ready' : ''}`} aria-label="อัปเดต Content Blueprint">
          <div className="update-banner-icon" aria-hidden="true">
            {phase === 'ready' ? <BadgeCheck size={20} /> : <CloudDownload size={20} />}
          </div>
          <div className="update-banner-copy">
            <strong>{phase === 'ready' ? `เวอร์ชัน ${info.latestVersion} พร้อมติดตั้ง` : `มีเวอร์ชัน ${info.latestVersion}`}</strong>
            <span>{phase === 'ready' ? 'ดาวน์โหลดและตรวจ SHA-256 แล้ว' : 'กดดูรายละเอียดก่อนดาวน์โหลด'}</span>
          </div>
          <button className="update-banner-action" type="button" onClick={() => setDialogOpen(true)}>ดูรายละเอียด</button>
          <button
            className="update-icon-button"
            type="button"
            onClick={() => setDismissedVersion(info.latestVersion)}
            aria-label={`ซ่อนข้อความอัปเดตเวอร์ชัน ${info.latestVersion}`}
          >
            <X size={17} aria-hidden="true" />
          </button>
        </aside>
      )}

      {dialogOpen && (
        <UpdateDialogFrame
          busy={busy}
          confirming={confirming}
          onCancelConfirmation={() => setConfirming(false)}
          onClose={closeCenter}
        >
          <header className="update-dialog-heading">
            <div className="update-dialog-icon" aria-hidden="true"><Rocket size={22} /></div>
            <div>
              <span className="update-eyebrow">GitHub Releases</span>
              <h2 id="update-dialog-title">อัปเดต Content Blueprint</h2>
              <p id="update-dialog-description">คุณเป็นคนเลือกว่าจะดาวน์โหลดและเปิดตัวติดตั้งเมื่อใด</p>
            </div>
            <button className="update-icon-button update-dialog-close" type="button" onClick={closeCenter} disabled={busy} aria-label="ปิดหน้าต่างอัปเดต">
              <X size={19} aria-hidden="true" />
            </button>
          </header>

          {info && (
            <div className="update-version-strip" aria-label="เวอร์ชันปัจจุบันและเวอร์ชันล่าสุด">
              <div><span>ใช้อยู่</span><strong>v{info.currentVersion}</strong></div>
              <span className="update-version-line" aria-hidden="true" />
              <div><span>ล่าสุด</span><strong>v{info.latestVersion}</strong></div>
            </div>
          )}

          {phase === 'checking' && (
            <div className="update-state-card" role="status">
              <LoaderCircle className="spin" size={24} aria-hidden="true" />
              <div><strong>กำลังตรวจ GitHub Releases</strong><p>รอสักครู่ แอปจะไม่ดาวน์โหลดไฟล์เอง</p></div>
            </div>
          )}

          {phase === 'up_to_date' && info && (
            <div className="update-state-card is-success" role="status">
              <CheckCircle2 size={24} aria-hidden="true" />
              <div><strong>เป็นเวอร์ชันล่าสุดแล้ว</strong><p>Content Blueprint v{info.currentVersion} พร้อมใช้งานต่อได้เลย</p></div>
            </div>
          )}

          {info && (phase === 'available' || phase === 'downloading' || phase === 'ready' || phase === 'installing') && (
            <section className="update-release" aria-labelledby="update-release-title">
              <div className="update-release-heading">
                <div>
                  <span className="update-release-date">เผยแพร่ {formatReleaseDate(info.publishedAt)}</span>
                  <h3 id="update-release-title">มีอะไรใหม่ใน v{info.latestVersion}</h3>
                </div>
                {trustedReleaseURL && (
                  <button className="update-text-button" type="button" onClick={() => openReleaseURL(trustedReleaseURL)}>
                    ดูบน GitHub <ExternalLink size={15} aria-hidden="true" />
                  </button>
                )}
              </div>
              <div className="update-release-notes">
                {info.releaseNotes?.trim()
                  ? info.releaseNotes.trim()
                  : 'ดูรายการเปลี่ยนแปลงฉบับเต็มได้จากหน้า GitHub Release'}
              </div>
            </section>
          )}

          {phase === 'downloading' && (
            <div className="update-progress-card" role="status" aria-live="polite" aria-atomic="true">
              <div className="update-progress-heading"><span>กำลังดาวน์โหลดตัวติดตั้ง</span><strong>{progressText}</strong></div>
              <div
                className={`update-progress-track ${progress?.totalBytes ? '' : 'is-indeterminate'}`}
                role="progressbar"
                aria-label="ความคืบหน้าการดาวน์โหลด"
                aria-valuemin={0}
                aria-valuemax={progress?.totalBytes ? 100 : undefined}
                aria-valuenow={progress?.totalBytes ? progressPercent : undefined}
                aria-valuetext={progressText}
              >
                <span style={progress?.totalBytes ? {transform: `scaleX(${progressPercent / 100})`} : undefined} />
              </div>
              <p>แอปจะตรวจ SHA-256 ก่อนเปิดปุ่มติดตั้ง</p>
            </div>
          )}

          {phase === 'ready' && !confirming && info && (
            <div className="update-verified-card" role="status">
              <BadgeCheck size={24} aria-hidden="true" />
              <div><strong>SHA-256 verified</strong><p>ไฟล์ v{info.latestVersion} ตรงกับ checksum ที่เผยแพร่ใน GitHub Release</p></div>
            </div>
          )}

          {confirming && info && (
            <section className="update-confirmation" aria-labelledby="update-confirmation-title">
              <div className="update-confirmation-heading">
                <ShieldAlert size={24} aria-hidden="true" />
                <div><span className="update-eyebrow">ยืนยันอีกครั้ง</span><h3 id="update-confirmation-title">เปิดตัวติดตั้ง v{info.latestVersion}?</h3></div>
              </div>
              <p>Windows อาจแสดง Microsoft Defender SmartScreen ว่า <strong>Unknown publisher</strong> เพราะรุ่นนี้ยังไม่มีลายเซ็นโค้ด ตรวจว่าไฟล์มาจาก GitHub ของ Useless007 และหน้าจอนี้แสดง SHA-256 verified ก่อนเลือก Run anyway</p>
              <div className="update-confirm-check"><ShieldCheck size={18} aria-hidden="true" /><span>แอปตรวจ checksum แล้ว แต่ยังไม่สามารถยืนยันผู้เผยแพร่ด้วยลายเซ็น Windows ได้</span></div>
            </section>
          )}

          {phase === 'installing' && (
            <div className="update-state-card" role="status">
              <LoaderCircle className="spin" size={24} aria-hidden="true" />
              <div><strong>กำลังเปิดตัวติดตั้งที่ตรวจสอบแล้ว</strong><p>ระบบจะปิด Content Blueprint หลัง Windows รับคำสั่งเปิดสำเร็จ</p></div>
            </div>
          )}

          {phase === 'error' && (
            <div className="update-error" role="alert">
              <ShieldAlert size={22} aria-hidden="true" />
              <div><strong>{failureAction === 'check' ? 'ตรวจอัปเดตไม่สำเร็จ' : failureAction === 'download' ? 'ดาวน์โหลดไม่สำเร็จ' : 'เปิดตัวติดตั้งไม่สำเร็จ'}</strong><p>{error}</p></div>
            </div>
          )}

          <footer className="update-dialog-actions">
            {confirming ? (
              <>
                <button className="update-button update-button-quiet" type="button" onClick={() => setConfirming(false)} disabled={busy} data-update-confirm-cancel>ย้อนกลับ</button>
                <span className="update-action-spacer" />
                <button className="update-button update-button-primary" type="button" onClick={() => void launchInstaller()} disabled={busy}>
                  {phase === 'installing' ? <LoaderCircle className="spin" size={17} aria-hidden="true" /> : <Rocket size={17} aria-hidden="true" />}
                  {phase === 'installing' ? 'กำลังเปิด…' : 'เปิดตัวติดตั้ง Windows'}
                </button>
              </>
            ) : (
              <>
                <button className="update-button update-button-quiet" type="button" onClick={() => void checkForUpdates(false)} disabled={busy}>
                  {phase === 'checking' ? <LoaderCircle className="spin" size={17} aria-hidden="true" /> : <RefreshCw size={17} aria-hidden="true" />}
                  ตรวจอีกครั้ง
                </button>
                <span className="update-action-spacer" />
                {phase === 'available' && (
                  <button className="update-button update-button-primary" type="button" onClick={() => void downloadUpdate()}>
                    <Download size={17} aria-hidden="true" /> ดาวน์โหลดอัปเดต
                  </button>
                )}
                {phase === 'ready' && (
                  <button className="update-button update-button-primary" type="button" onClick={() => setConfirming(true)}>
                    <ShieldCheck size={17} aria-hidden="true" /> ติดตั้งอัปเดต
                  </button>
                )}
                {phase === 'error' && (
                  <button className="update-button update-button-primary" type="button" onClick={retry}>
                    <RefreshCw size={17} aria-hidden="true" /> ลองอีกครั้ง
                  </button>
                )}
                {(phase === 'idle' || phase === 'up_to_date') && (
                  <button className="update-button update-button-secondary" type="button" onClick={closeCenter}>ปิด</button>
                )}
              </>
            )}
          </footer>

          <div className="update-privacy-note"><Info size={15} aria-hidden="true" /><span>ตรวจจาก repository สาธารณะเท่านั้น ไม่มี token ฝังในแอป และไม่มีการติดตั้งแบบเงียบ</span></div>
        </UpdateDialogFrame>
      )}
    </div>
  )
}
