import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/audio/audio_quality.dart';
import '../../domain/entities/compliance.dart';
import '../../domain/entities/session_event.dart';
import 'di_providers.dart';
import 'session_state.dart';

class SessionController extends Notifier<SessionState> {
  StreamSubscription<SessionEvent>? _events;
  StreamSubscription<List<int>>? _mic;
  final _quality = AudioQualityMonitor();

  @override
  SessionState build() {
    ref.onDispose(_teardown);
    return const SessionState();
  }

  Future<void> loadObligations() async {
    try {
      final obligations =
          await ref.read(sessionRepositoryProvider).obligations();
      state = state.copyWith(obligations: obligations);
    } catch (e) {
      state = state.copyWith(error: '$e');
    }
  }

  Future<void> start(String productId) async {
    if (state.starting) return; // cegah ketukan ganda membuat dua sesi
    state = state.copyWith(starting: true, clearError: true);
    try {
      final session = await ref.read(startSessionProvider)(productId);
      state = state.copyWith(session: session, clearError: true);
      await _listen(session.id);
    } catch (e) {
      state = state.copyWith(error: '$e');
    } finally {
      state = state.copyWith(starting: false);
    }
  }

  Future<void> end() async {
    final id = state.session?.id;
    if (id == null) return;
    try {
      final session = await ref.read(endSessionProvider)(id);
      await _teardown();
      state = state.copyWith(
        session: session,
        recording: false,
        connected: false,
        loadingReport: true,
      );

      // Laporan diambil setelah sesi ditutup, bukan sebelum: backend baru
      // menghitung skor akhir setelah revisi label terakhir dari diarization
      // selesai diproses.
      final report = await ref.read(getReportProvider)(id);
      state = state.copyWith(report: report, loadingReport: false);
    } catch (e) {
      state = state.copyWith(error: '$e', loadingReport: false);
    }
  }

  /// Mengunci peran dan memulai penilaian.
  void confirmSpeakerRoles() {
    final officer = state.officerVoice;
    final customer = state.customerVoice;
    if (officer == null || customer == null) return;
    ref.read(sessionRepositoryProvider).confirmSpeakerRoles(officer, customer);
  }

  /// Mulai ulang kalibrasi dari nol — gateway ikut membuang mapping lama.
  void restartCalibration() {
    state = state.copyWith(
      calibration: CalibrationStatus.collecting,
      calibrationSamples: const [],
      duplicateVoice: false,
      clearOfficerVoice: true,
      clearCalibrationError: true,
    );
    ref.read(sessionRepositoryProvider).beginSpeakerCalibration();
  }

  /// Membuang laporan dan kembali ke layar awal untuk sesi berikutnya.
  ///
  /// Daftar kewajiban dimuat ulang, bukan dipertahankan: statusnya masih
  /// membawa hasil sesi sebelumnya, dan kalau dibiarkan, sesi baru akan
  /// terbuka dengan checklist yang sudah hijau.
  void reset() {
    state = const SessionState();
    unawaited(loadObligations());
  }

  Future<void> _listen(String sessionId) async {
    final usecase = ref.read(streamSessionProvider);
    final repo = ref.read(sessionRepositoryProvider);

    _events = usecase.events(sessionId).listen(
          _onEvent,
          onError: (Object e) => state = state.copyWith(error: '$e'),
        );
    state = state.copyWith(connected: true);

    // Dikirim SEBELUM mikrofon menyala, supaya frame pertama pun sudah
    // diperlakukan sebagai kalibrasi — bukan otomatis dianggap petugas.
    ref.read(sessionRepositoryProvider).beginSpeakerCalibration();

    try {
      final mic = await usecase.microphone();
      _mic = mic.listen((pcm) {
        // Kualitas diukur dari frame yang SAMA dengan yang dikirim, dan
        // hanya dilaporkan saat vonisnya berubah.
        final reason = _quality.add(pcm);
        if (reason != null) repo.reportAudioQuality(reason);
        repo.pushAudio(pcm);
      });
      state = state.copyWith(recording: true);
    } catch (e) {
      state = state.copyWith(error: '$e', recording: false);
    }
  }

  void _onEvent(SessionEvent event) {
    switch (event) {
      case CalibrationStarted():
        state = state.copyWith(
          calibration: CalibrationStatus.collecting,
          calibrationSamples: const [],
          clearCalibrationError: true,
        );

      // Langkah 1 selesai saat suara PERTAMA dikenali; sesudah itu, suara
      // yang sama berarti orang kedua belum bicara.
      case CalibrationUtterance(
          :final utteranceId,
          :final sourceSpeaker,
          :final text
        ):
        final samples = [...state.calibrationSamples];
        final index = samples.indexWhere((s) => s.id == utteranceId);
        final merged = CalibrationSample(
          id: utteranceId,
          sourceSpeaker: sourceSpeaker,
          // Revisi tidak selalu membawa teks; pertahankan yang lama.
          text: text.isNotEmpty
              ? text
              : (index >= 0 ? samples[index].text : ''),
        );
        if (index >= 0) {
          samples[index] = merged;
        } else {
          samples.add(merged);
        }

        final officer = state.officerVoice;
        state = state.copyWith(
          calibrationSamples: samples,
          officerVoice: officer ?? (sourceSpeaker.isNotEmpty ? sourceSpeaker : null),
          duplicateVoice:
              officer != null && sourceSpeaker == officer,
        );

      case SpeakerRolesConfirmed():
        state = state.copyWith(
          calibration: CalibrationStatus.confirmed,
          clearCalibrationError: true,
        );

      case CalibrationError(:final message):
        state = state.copyWith(calibrationError: message);

      case PartialReceived(:final text):
        state = state.copyWith(partial: text);

      case UtteranceReceived(:final utterance):
        state = state.copyWith(
          utterances: [...state.utterances, utterance],
          clearPartial: true,
        );

      // Diarization merevisi label sebelumnya — ini fitur, tandai di UI.
      case SpeakerRevised(:final utteranceId, :final speaker):
        state = state.copyWith(
          utterances: [
            for (final u in state.utterances)
              if (u.id == utteranceId)
                u.copyWith(speaker: speaker, revised: true)
              else
                u,
          ],
        );

      case ObligationSatisfied(:final code, :final confidence, :final evidenceId):
        state = state.copyWith(
          obligations: [
            for (final o in state.obligations)
              if (o.code == code)
                o.copyWith(
                  status: ObligationStatus.satisfied,
                  confidence: confidence,
                  evidenceId: evidenceId,
                )
              else
                o,
          ],
        );

      case ViolationDetected(:final phrase, :final severity, :final evidenceId):
        state = state.copyWith(
          violations: [
            ...state.violations,
            Violation(
              phrase: phrase,
              severity: severity,
              evidenceId: evidenceId,
              detectedAt: DateTime.now(),
            ),
          ],
        );

      case NudgeReceived(:final text):
        state = state.copyWith(lastNudge: text);
        unawaited(ref.read(speechRepositoryProvider).whisper(text));

      // Pesan kosong = kondisi sudah pulih, bersihkan peringatannya.
      case SessionWarningReceived(:final message):
        state = state.copyWith(warning: message, clearWarning: message.isEmpty);

      // Jalur audio mati: hentikan indikator merekam supaya petugas tidak
      // mengira sesi masih disimak.
      case SessionErrorReceived(:final message):
        state = state.copyWith(error: message, recording: false);

      case UnknownEvent():
        break;
    }
  }

  Future<void> _teardown() async {
    await _mic?.cancel();
    await _events?.cancel();
    _mic = null;
    _events = null;
    await ref.read(streamSessionProvider).stop();
    await ref.read(speechRepositoryProvider).cancel();
  }
}

final sessionControllerProvider =
    NotifierProvider<SessionController, SessionState>(SessionController.new);
