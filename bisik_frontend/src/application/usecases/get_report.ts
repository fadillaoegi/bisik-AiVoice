import type { SessionRepository } from '../../domain/repositories/session_repository'
import type { ComplianceReport } from '../../domain/entities/compliance'

export class GetReportUseCase {
  private readonly repo: SessionRepository

  constructor(repo: SessionRepository) {
    this.repo = repo
  }

  execute(sessionId: string): Promise<ComplianceReport> {
    return this.repo.report(sessionId)
  }
}
