import type {UpdateProgress} from './types'

const RELEASE_PATH_PREFIX = '/Useless007/content-blueprint/releases/'

interface RuntimeBridge {
  BrowserOpenURL?: (url: string) => void
  EventsOnMultiple?: (
    eventName: string,
    callback: (payload: unknown) => void,
    maxCallbacks: number,
  ) => (() => void) | void
}

type RuntimeWindow = Window & {runtime?: RuntimeBridge}

export const UPDATE_LAST_CHECK_KEY = 'content-blueprint:update:last-automatic-check'
export const UPDATE_AUTOMATIC_INTERVAL_MS = 24 * 60 * 60 * 1000
export const UPDATE_AUTOMATIC_DELAY_MS = 2200

export function isAutomaticUpdateCheckDue(now = Date.now()): boolean {
  try {
    const stored = Number(window.localStorage.getItem(UPDATE_LAST_CHECK_KEY))
    if (!Number.isFinite(stored) || stored <= 0) return true
    if (stored > now + 5 * 60 * 1000) return true
    return now - stored >= UPDATE_AUTOMATIC_INTERVAL_MS
  } catch {
    return true
  }
}

export function recordAutomaticUpdateCheck(now = Date.now()): void {
  try {
    window.localStorage.setItem(UPDATE_LAST_CHECK_KEY, String(now))
  } catch {
    // A blocked localStorage must not prevent a manual or automatic check.
  }
}

export function isTrustedReleaseURL(value?: string): value is string {
  if (!value) return false
  try {
    const parsed = new URL(value)
    return parsed.protocol === 'https:'
      && parsed.hostname === 'github.com'
      && parsed.pathname.startsWith(RELEASE_PATH_PREFIX)
  } catch {
    return false
  }
}

export function openReleaseURL(url: string): void {
  if (!isTrustedReleaseURL(url)) return
  const runtime = (window as RuntimeWindow).runtime
  if (typeof runtime?.BrowserOpenURL === 'function') {
    runtime.BrowserOpenURL(url)
    return
  }
  window.open(url, '_blank', 'noopener,noreferrer')
}

export function subscribeToUpdateProgress(
  listener: (progress: UpdateProgress) => void,
): () => void {
  const runtime = (window as RuntimeWindow).runtime
  if (typeof runtime?.EventsOnMultiple !== 'function') return () => undefined

  const cancel = runtime.EventsOnMultiple('app:update-progress', (payload) => {
    if (!payload || typeof payload !== 'object') return
    const candidate = payload as Partial<UpdateProgress>
    if (
      typeof candidate.version !== 'string'
      || typeof candidate.downloadedBytes !== 'number'
      || typeof candidate.totalBytes !== 'number'
      || typeof candidate.percent !== 'number'
    ) return
    listener({
      version: candidate.version,
      downloadedBytes: Math.max(0, candidate.downloadedBytes),
      totalBytes: Math.max(0, candidate.totalBytes),
      percent: Math.min(100, Math.max(0, Math.round(candidate.percent))),
    })
  }, -1)

  return typeof cancel === 'function' ? cancel : () => undefined
}
