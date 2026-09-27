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

export interface ComplianceReport {
  session: Session
  obligations: Obligation[]
  violations: Violation[]
  transcript: Utterance[]
}
