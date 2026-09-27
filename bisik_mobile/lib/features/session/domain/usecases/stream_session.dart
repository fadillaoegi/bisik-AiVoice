import '../entities/session_event.dart';
import '../repositories/session_repository.dart';

/// Menyatukan dua arah: audio keluar ke gateway, event masuk ke UI.
class StreamSessionUseCase {
  const StreamSessionUseCase(this._session, this._audio);
  final SessionRepository _session;
  final AudioRepository _audio;

  Stream<SessionEvent> events(String sessionId) => _session.connect(sessionId);

  Future<Stream<List<int>>> microphone() => _audio.stream();

  Future<void> stop() async {
    await _audio.stop();
    await _session.disconnect();
  }
}
