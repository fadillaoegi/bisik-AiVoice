import '../entities/session.dart';
import '../repositories/session_repository.dart';

class EndSessionUseCase {
  const EndSessionUseCase(this._repo);
  final SessionRepository _repo;

  Future<Session> call(String sessionId) => _repo.end(sessionId);
}
