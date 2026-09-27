import type { Speaker } from './session'

/** Event realtime yang dikirim gateway Go lewat WebSocket. */
export type SessionEvent =
  | { type: 'partial'; speaker: Speaker; text: string }
  | { type: 'utterance'; id: string; speaker: Speaker; source_speaker?: string; text: string }
  | { type: 'speaker_revised'; utterance_id: string; speaker: Speaker }
  | { type: 'speaker_calibration_started' }
  | { type: 'speaker_calibration_partial'; utterance_id: string; source_speaker: string; text: string }
  | { type: 'speaker_calibration_utterance'; utterance_id: string; source_speaker: string; text: string }
  | { type: 'speaker_roles_confirmed'; officer_label: string; customer_label: string }
  | { type: 'speaker_calibration_error'; message: string }
  | { type: 'obligation_satisfied'; code: string; confidence: number; evidence_id: string }
  | { type: 'violation'; phrase: string; severity: string; evidence_id: string }
  | { type: 'nudge'; text: string }
  | { type: 'session_error'; message: string }
  | { type: 'speaker_unknown'; utterance_id: string; text: string }
  | { type: 'evidence_skipped'; utterance_id: string; reason: string }
  | { type: 'audio_quality'; degraded: boolean; reason: string }
