import type { RootState } from './index'

export const selectAuthUser = (s: RootState) => s.auth.user
export const selectAuthRestoring = (s: RootState) => s.auth.restoring
export const selectAuthError = (s: RootState) => s.auth.error
export const selectSession = (s: RootState) => s.session.current
export const selectConnected = (s: RootState) => s.session.connected
export const selectRecording = (s: RootState) => s.session.recording
export const selectSessionError = (s: RootState) => s.session.error
export const selectUtterances = (s: RootState) => s.transcript.utterances
export const selectPartial = (s: RootState) => s.transcript.partial
export const selectObligations = (s: RootState) => s.compliance.obligations
export const selectViolations = (s: RootState) => s.compliance.violations
export const selectLastNudge = (s: RootState) => s.compliance.lastNudge

export const selectPendingObligations = (s: RootState) =>
  s.compliance.obligations.filter((o) => o.status === 'pending')

/** Skor sementara: % butir terpenuhi dikurangi 10 per pelanggaran. */
export const selectLiveScore = (s: RootState) => {
  const total = s.compliance.obligations.length
  if (total === 0) return 0
  const done = s.compliance.obligations.filter((o) => o.status === 'satisfied').length
  return Math.max(0, Math.round((done / total) * 100) - s.compliance.violations.length * 10)
}
