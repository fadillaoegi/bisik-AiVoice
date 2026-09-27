export type Speaker = 'officer' | 'customer' | 'unknown'
export type SessionStatus = 'active' | 'ended' | 'aborted'

export interface Session {
  id: string
  officerId: string
  productId: string
  status: SessionStatus
  startedAt: string
  score: number
}

export interface Utterance {
  id: string
  speaker: Speaker
  text: string
  startMs: number
  /** true kalau label speaker pernah direvisi oleh diarization */
  revised: boolean
  /** Label mentah diarization (mis. "A"/"B") — untuk verifikasi, bukan identitas. */
  sourceSpeaker?: string
}
