import type { SessionRepository } from '../../domain/repositories/session_repository'
import type { Session } from '../../domain/entities/session'
import type { ComplianceReport, Obligation } from '../../domain/entities/compliance'
import { tokenStore } from './token_store'

// `||`, bukan `??`: Vite mengisi variabel yang sengaja dikosongkan
// dengan string kosong, dan string kosong berarti "pakai origin halaman".
const API = import.meta.env.VITE_API_URL || window.location.origin

/** Menyisipkan token bearer bila ada. */
function authHeaders(extra?: Record<string, string>): Record<string, string> {
  const token = tokenStore.read()
  return token ? { ...extra, Authorization: `Bearer ${token}` } : { ...extra }
}

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string }
    throw new Error(body.error ?? `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

interface SessionDTO {
  id: string
  officer_id: string
  product_id: string
  status: Session['status']
  started_at: string
  score: number
}

const toSession = (d: SessionDTO): Session => ({
  id: d.id,
  officerId: d.officer_id,
  productId: d.product_id,
  status: d.status,
  startedAt: d.started_at,
  score: d.score,
})

export class HttpSessionRepository implements SessionRepository {
  // Pemilik sesi tidak dikirim klien: backend mengambilnya dari token supaya
  // atribusi laporan tidak bisa diklaim sendiri oleh klien.
  async start(productId: string): Promise<Session> {
    const res = await fetch(`${API}/api/sessions`, {
      method: 'POST',
      headers: authHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ product_id: productId }),
    })
    return toSession(await json<SessionDTO>(res))
  }

  async end(sessionId: string): Promise<Session> {
    const res = await fetch(`${API}/api/sessions/${sessionId}/end`, {
      method: 'POST',
      headers: authHeaders(),
    })
    return toSession(await json<SessionDTO>(res))
  }

  async report(sessionId: string): Promise<ComplianceReport> {
    const res = await fetch(`${API}/api/sessions/${sessionId}/report`, {
      headers: authHeaders(),
    })
    const raw = await json<{
      session: SessionDTO
      obligations: Array<{ code: string; label: string; status: Obligation['status']; confidence: number; evidence_id?: string }>
      violations: Array<{ phrase: string; severity: string; evidence_id: string; detected_at: string }>
      transcript: Array<{ id: string; speaker: string; text: string; start_ms: number; revised: boolean }>
    }>(res)

    return {
      session: toSession(raw.session),
      obligations: (raw.obligations ?? []).map((o) => ({
        code: o.code, label: o.label, status: o.status,
        confidence: o.confidence, evidenceId: o.evidence_id,
      })),
      violations: (raw.violations ?? []).map((v) => ({
        phrase: v.phrase, severity: v.severity,
        evidenceId: v.evidence_id, detectedAt: v.detected_at,
      })),
      transcript: (raw.transcript ?? []).map((u) => ({
        id: u.id, speaker: u.speaker as never, text: u.text,
        startMs: u.start_ms, revised: u.revised,
      })),
    }
  }

  /** Daftar sesi terbaru untuk supervisor. */
  async sessions(): Promise<Session[]> {
    const res = await fetch(`${API}/api/sessions`, { headers: authHeaders() })
    const raw = await json<SessionDTO[]>(res)
    return (raw ?? []).map(toSession)
  }

  async obligations(): Promise<Obligation[]> {
    const res = await fetch(`${API}/api/obligations`, { headers: authHeaders() })
    const raw = await json<Array<{ code: string; label: string; status: Obligation['status']; confidence: number }>>(res)
    return raw.map((o) => ({ code: o.code, label: o.label, status: o.status, confidence: o.confidence }))
  }
}
