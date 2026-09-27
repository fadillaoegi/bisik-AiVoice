import type { SessionRepository } from '../../domain/repositories/session_repository'
import type { Session } from '../../domain/entities/session'

export class StartSessionUseCase {
  private readonly repo: SessionRepository

  constructor(repo: SessionRepository) {
    this.repo = repo
  }

  /**
   * Identitas petugas TIDAK lagi menjadi parameter: backend mengambilnya
   * dari token. Validasi "ID petugas wajib diisi" sengaja dihapus — dulu
   * wajar ketika ID diketik manusia, sekarang justru menolak permintaan
   * yang sah sebelum sempat dikirim.
   */
  async execute(productId: string): Promise<Session> {
    return this.repo.start(productId)
  }
}
