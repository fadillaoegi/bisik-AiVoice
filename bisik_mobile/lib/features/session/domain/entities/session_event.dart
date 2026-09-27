import 'session.dart';

/// Event realtime dari gateway Go.
sealed class SessionEvent {
  const SessionEvent();
}

class PartialReceived extends SessionEvent {
  const PartialReceived(this.speaker, this.text);
  final Speaker speaker;
  final String text;
}

class UtteranceReceived extends SessionEvent {
  const UtteranceReceived(this.utterance);
  final Utterance utterance;
}

class SpeakerRevised extends SessionEvent {
  const SpeakerRevised(this.utteranceId, this.speaker);
  final String utteranceId;
  final Speaker speaker;
}

class ObligationSatisfied extends SessionEvent {
  const ObligationSatisfied(this.code, this.confidence, this.evidenceId);
  final String code;
  final double confidence;
  final String evidenceId;
}

class ViolationDetected extends SessionEvent {
  const ViolationDetected(this.phrase, this.severity, this.evidenceId);
  final String phrase;
  final String severity;
  final String evidenceId;
}

class NudgeReceived extends SessionEvent {
  const NudgeReceived(this.text);
  final String text;
}

/// Kalibrasi dimulai: gateway menahan penilaian sampai dua suara dikenali.
class CalibrationStarted extends SessionEvent {
  const CalibrationStarted();
}

/// Satu contoh suara saat kalibrasi. `sourceSpeaker` adalah label mentah
/// diarization (mis. "A"/"B") — belum berarti petugas atau nasabah.
class CalibrationUtterance extends SessionEvent {
  const CalibrationUtterance(this.utteranceId, this.sourceSpeaker, this.text);
  final String utteranceId;
  final String sourceSpeaker;
  final String text;
}

/// Mapping peran sudah dikunci; penilaian boleh dimulai.
class SpeakerRolesConfirmed extends SessionEvent {
  const SpeakerRolesConfirmed();
}

class CalibrationError extends SessionEvent {
  const CalibrationError(this.message);
  final String message;
}

/// Bagian percakapan sengaja tidak dihitung sebagai bukti kepatuhan:
/// pembicara tidak dikenal, atau audio tidak layak. Flutter belum punya
/// panel khusus, jadi ketiganya masuk ke satu slot peringatan.
class SessionWarningReceived extends SessionEvent {
  const SessionWarningReceived(this.message);
  final String message;
}

/// Jalur audio upstream berhenti sebelum petugas mengakhiri sesi.
class SessionErrorReceived extends SessionEvent {
  const SessionErrorReceived(this.message);
  final String message;
}

class UnknownEvent extends SessionEvent {
  const UnknownEvent();
}
