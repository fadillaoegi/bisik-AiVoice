import '../../domain/entities/compliance.dart';
import '../../domain/entities/session.dart';
import '../../domain/entities/session_event.dart';
import '../../domain/repositories/session_repository.dart';
import '../datasources/audio_datasource.dart';
import '../datasources/session_remote_datasource.dart';
import '../datasources/session_ws_datasource.dart';
import '../datasources/tts_datasource.dart';
import '../models/compliance_model.dart';
import '../models/session_model.dart';

class SessionRepositoryImpl implements SessionRepository {
  const SessionRepositoryImpl(this._remote, this._ws);

  final SessionRemoteDataSource _remote;
  final SessionWsDataSource _ws;

  @override
  Future<Session> start(String productId) async =>
      SessionModel.fromJson(await _remote.startSession(productId));

  @override
  Future<Session> end(String sessionId) async =>
      SessionModel.fromJson(await _remote.endSession(sessionId));

  @override
  Future<ComplianceReport> report(String sessionId) async =>
      ComplianceReportModel.fromJson(await _remote.report(sessionId));

  @override
  Future<List<Obligation>> obligations() async {
    final raw = await _remote.obligations();
    return raw
        .cast<Map<String, dynamic>>()
        .map(ObligationModel.fromJson)
        .toList(growable: false);
  }

  @override
  Stream<SessionEvent> connect(String sessionId) => _ws.connect(sessionId);

  @override
  Future<void> disconnect() => _ws.disconnect();

  /// Dipakai controller untuk meneruskan frame mikrofon ke gateway.
  void pushAudio(List<int> pcm) => _ws.sendAudio(pcm);

  void beginSpeakerCalibration() => _ws.beginSpeakerCalibration();

  void confirmSpeakerRoles(String officerLabel, String customerLabel) =>
      _ws.confirmSpeakerRoles(officerLabel, customerLabel);

  void reportAudioQuality(String reason) => _ws.reportAudioQuality(reason);
}

class AudioRepositoryImpl implements AudioRepository {
  const AudioRepositoryImpl(this._source);
  final AudioDataSource _source;

  @override
  Future<Stream<List<int>>> stream() => _source.stream();

  @override
  Future<void> stop() => _source.stop();
}

class SpeechRepositoryImpl implements SpeechRepository {
  const SpeechRepositoryImpl(this._source);
  final TtsDataSource _source;

  @override
  Future<void> whisper(String text) => _source.speak(text);

  @override
  Future<void> cancel() => _source.stop();
}
