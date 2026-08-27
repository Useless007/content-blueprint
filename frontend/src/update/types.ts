export type UpdateState =
  | 'up_to_date'
  | 'update_available'
  | 'downloading'
  | 'ready'

export interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  state: UpdateState
  releaseUrl?: string
  publishedAt?: string
  releaseNotes?: string
}

export interface UpdateProgress {
  version: string
  downloadedBytes: number
  totalBytes: number
  percent: number
}

export type UpdatePhase =
  | 'idle'
  | 'checking'
  | 'up_to_date'
  | 'available'
  | 'downloading'
  | 'ready'
  | 'installing'
  | 'error'

export type UpdateFailureAction = 'check' | 'download' | 'launch'
