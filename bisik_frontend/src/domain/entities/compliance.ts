import type { Session, Utterance } from './session'

export type ObligationStatus = 'pending' | 'satisfied' | 'violated'

export interface Obligation {
  code: string
  label: string
  status: ObligationStatus
  confidence: number
  /** id utterance yang membuktikan butir ini terpenuhi */
  evidenceId?: string
}

export interface Violation {
  phrase: string
  severity: string
  evidenceId: string
  detectedAt: string
}

/**
 * Satu bisikan ke earpiece petugas.
 *
 * `text` adalah kalimat Bahasa Indonesia yang diucapkan — backend mencocokkan
 * kalimat itu untuk mengenali gemanya sendiri. `kind` beserta `code`/`phrase`
 * dipakai untuk MENAMPILKAN bisikan dalam bahasa antarmuka lain.
 */
export interface Nudge {
  text: string
  kind?: 'avoid_phrase' | 'pending_obligation' | 'demo_correction' | 'demo_reminder'
  code?: string
  phrase?: string
}

export interface ComplianceReport {
  session: Session
  obligations: Obligation[]
  violations: Violation[]
  transcript: Utterance[]
}
