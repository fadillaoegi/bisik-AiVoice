import '../../../core/i18n/app_strings.dart';
import '../domain/entities/compliance.dart';
import '../domain/entities/session.dart';
import '../domain/entities/session_event.dart';

/// Menyusun kalimat dari entitas sesi — kembaran `i18n/localize.ts` di PWA.
///
/// Sengaja di sini, bukan di `core/i18n`: kamus di `core` tidak boleh
/// bergantung pada entitas fitur.
extension SessionStrings on AppStrings {
  /// Label menurut kode; kode baru dari backend memakai label aslinya.
  String obligation(Obligation o) => obligationLabel(o.code) ?? o.label;

  String speaker(Speaker s) => switch (s) {
    Speaker.officer => officer,
    Speaker.customer => customer,
    Speaker.unknown => unknownSpeaker,
  };

  /// Bisikan yang tampil. Yang terdengar selalu [Nudge.text].
  String nudge(Nudge n) => switch (n.kind) {
    NudgeKind.avoidPhrase when n.phrase.isNotEmpty => nudgeAvoidPhrase(
      n.phrase,
    ),
    NudgeKind.pendingObligation when n.code.isNotEmpty => nudgePending(
      obligationLabel(n.code) ?? n.text,
    ),
    _ => n.text,
  };

  String warning(SessionWarning w) => switch (w.kind) {
    SessionWarningKind.unknownSpeaker => warningUnknownSpeaker,
    SessionWarningKind.evidenceSkipped => warningEvidenceSkipped(w.reason),
    SessionWarningKind.audioDegraded => warningAudioDegraded(w.reason),
  };
}
