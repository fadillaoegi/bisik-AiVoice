import '../entities/compliance.dart';
import '../repositories/session_repository.dart';

class GetReportUseCase {
  const GetReportUseCase(this._repo);
  final SessionRepository _repo;

  Future<ComplianceReport> call(String sessionId) => _repo.report(sessionId);
}
