import type { SessionRepository } from '../../domain/repositories/session_repository'
import type { Session } from '../../domain/entities/session'

export class EndSessionUseCase {
  private readonly repo: SessionRepository

  constructor(repo: SessionRepository) {
    this.repo = repo
  }

  execute(sessionId: string): Promise<Session> {
    return this.repo.end(sessionId)
  }
}
