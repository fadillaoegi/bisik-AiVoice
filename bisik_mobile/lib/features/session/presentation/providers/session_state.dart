import '../../domain/entities/compliance.dart';
import '../../domain/entities/session.dart';

/// Contoh suara saat kalibrasi, disimpan per ucapan.
///
/// Per ucapan, bukan per label: diarization bisa MENGOREKSI label sebuah
/// ucapan belakangan, dan kalau kuncinya label, koreksi itu tidak bisa
/// dipetakan ke contoh mana pun.
class CalibrationSample {
  const CalibrationSample({
    required this.id,
    required this.sourceSpeaker,
    required this.text,
  });

  final String id;
  final String sourceSpeaker;
  final String text;
}

enum CalibrationStatus { idle, collecting, confirmed }

class SessionState {
  const SessionState({
    this.session,
    this.obligations = const [],
    this.utterances = const [],
    this.violations = const [],
    this.partial,
    this.lastNudge,
    this.connected = false,
    this.recording = false,
    this.starting = false,
    this.report,
    this.loadingReport = false,
    this.error,
    this.warning,
    this.calibration = CalibrationStatus.idle,
    this.calibrationSamples = const [],
    this.officerVoice,
    this.duplicateVoice = false,
    this.calibrationError,
  });

  final Session? session;
  final List<Obligation> obligations;
  final List<Utterance> utterances;
  final List<Violation> violations;
  final String? partial;
  final String? lastNudge;
  final bool connected;
  final bool recording;

  /// Sesi sedang dibuka: HTTP, WebSocket, dan mikrofon belum selesai.
  /// Tanpa ini tombol "Mulai sesi" terasa mati selama beberapa detik.
  final bool starting;

  /// Laporan berbukti setelah sesi berakhir. Ini payoff produknya:
  /// skor tidak berarti apa-apa tanpa kutipan yang mendasarinya.
  final ComplianceReport? report;
  final bool loadingReport;

  final String? error;

  /// Peringatan non-fatal: bagian percakapan yang tidak dihitung sebagai bukti.
  final String? warning;

  final CalibrationStatus calibration;
  final List<CalibrationSample> calibrationSamples;

  /// Suara yang sudah terdaftar sebagai petugas; null = langkah 1 belum usai.
  final String? officerVoice;

  /// Suara yang sama terdengar lagi saat menunggu orang kedua.
  final bool duplicateVoice;

  final String? calibrationError;

  /// Suara yang benar-benar dikenali, terurut. Label kosong disaring di
  /// gateway, jadi di sini cukup menyaring yang tidak membawa teks.
  List<String> get recognisedVoices {
    final labels = <String>{};
    for (final s in calibrationSamples) {
      if (s.sourceSpeaker.isNotEmpty) labels.add(s.sourceSpeaker);
    }
    final sorted = labels.toList()..sort();
    return sorted;
  }

  String? get customerVoice {
    final officer = officerVoice;
    if (officer == null) return null;
    for (final label in recognisedVoices) {
      if (label != officer) return label;
    }
    return null;
  }

  /// Ucapan terakhir per suara — satu orang bisa bicara berkali-kali.
  String utteranceOf(String label) {
    var text = '';
    for (final s in calibrationSamples) {
      if (s.sourceSpeaker == label && s.text.isNotEmpty) text = s.text;
    }
    return text;
  }

  /// Skor sementara: % butir terpenuhi dikurangi 10 per pelanggaran.
  int get liveScore {
    if (obligations.isEmpty) return 0;
    final done = obligations
        .where((o) => o.status == ObligationStatus.satisfied)
        .length;
    final raw = (done / obligations.length * 100).round() - violations.length * 10;
    return raw < 0 ? 0 : raw;
  }

  SessionState copyWith({
    Session? session,
    List<Obligation>? obligations,
    List<Utterance>? utterances,
    List<Violation>? violations,
    String? partial,
    String? lastNudge,
    bool? connected,
    bool? recording,
    bool? starting,
    ComplianceReport? report,
    bool? loadingReport,
    String? error,
    String? warning,
    CalibrationStatus? calibration,
    List<CalibrationSample>? calibrationSamples,
    String? officerVoice,
    bool? duplicateVoice,
    String? calibrationError,
    bool clearPartial = false,
    bool clearError = false,
    bool clearWarning = false,
    bool clearCalibrationError = false,
    bool clearOfficerVoice = false,
  }) =>
      SessionState(
        session: session ?? this.session,
        obligations: obligations ?? this.obligations,
        utterances: utterances ?? this.utterances,
        violations: violations ?? this.violations,
        partial: clearPartial ? null : (partial ?? this.partial),
        lastNudge: lastNudge ?? this.lastNudge,
        connected: connected ?? this.connected,
        recording: recording ?? this.recording,
        starting: starting ?? this.starting,
        report: report ?? this.report,
        loadingReport: loadingReport ?? this.loadingReport,
        error: clearError ? null : (error ?? this.error),
        warning: clearWarning ? null : (warning ?? this.warning),
        calibration: calibration ?? this.calibration,
        calibrationSamples: calibrationSamples ?? this.calibrationSamples,
        officerVoice: clearOfficerVoice ? null : (officerVoice ?? this.officerVoice),
        duplicateVoice: duplicateVoice ?? this.duplicateVoice,
        calibrationError: clearCalibrationError
            ? null
            : (calibrationError ?? this.calibrationError),
      );
}
