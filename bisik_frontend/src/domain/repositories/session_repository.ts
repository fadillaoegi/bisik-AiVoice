import type { Session } from '../entities/session'
import type { ComplianceReport, Obligation } from '../entities/compliance'

/** Kontrak yang dipakai use case. Implementasinya ada di infrastructure. */
export interface SessionRepository {
  /** Pemilik sesi ditentukan token, bukan parameter. */
  start(productId: string): Promise<Session>
  end(sessionId: string): Promise<Session>
  report(sessionId: string): Promise<ComplianceReport>
  obligations(): Promise<Obligation[]>
  /** Hanya untuk supervisor: daftar sesi terbaru yang bisa dipantau. */
  sessions(): Promise<Session[]>
}
