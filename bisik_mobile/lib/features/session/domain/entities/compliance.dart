import 'session.dart';

enum ObligationStatus { pending, satisfied, violated }

enum NudgeKind { avoidPhrase, pendingObligation, other }

/// Satu bisikan ke earpiece petugas.
///
/// [text] adalah kalimat Bahasa Indonesia yang DIUCAPKAN — backend
/// mencocokkan kalimat itu untuk mengenali gemanya sendiri. [kind] beserta
/// [code]/[phrase] dipakai untuk MENAMPILKANNYA dalam bahasa antarmuka.
class Nudge {
  const Nudge({
    required this.text,
    this.kind = NudgeKind.other,
    this.code = '',
    this.phrase = '',
  });

  final String text;
  final NudgeKind kind;
  final String code;
  final String phrase;
}

class Obligation {
  const Obligation({
    required this.code,
    required this.label,
    this.status = ObligationStatus.pending,
    this.confidence = 0,
    this.evidenceId,
  });

  final String code;
  final String label;
  final ObligationStatus status;
  final double confidence;
  final String? evidenceId;

  Obligation copyWith({
    ObligationStatus? status,
    double? confidence,
    String? evidenceId,
  }) => Obligation(
    code: code,
    label: label,
    status: status ?? this.status,
    confidence: confidence ?? this.confidence,
    evidenceId: evidenceId ?? this.evidenceId,
  );
}

class Violation {
  const Violation({
    required this.phrase,
    required this.severity,
    required this.evidenceId,
    required this.detectedAt,
  });

  final String phrase;
  final String severity;
  final String evidenceId;
  final DateTime detectedAt;
}

/// Laporan kepatuhan berbukti: tiap butir menunjuk ucapan mana yang
/// memenuhinya, sehingga skornya bisa ditelusuri balik ke percakapan.
class ComplianceReport {
  const ComplianceReport({
    required this.session,
    required this.obligations,
    required this.violations,
    required this.transcript,
  });

  final Session session;
  final List<Obligation> obligations;
  final List<Violation> violations;
  final List<Utterance> transcript;

  int get satisfiedCount =>
      obligations.where((o) => o.status == ObligationStatus.satisfied).length;

  /// Ucapan yang menjadi bukti sebuah butir, kalau masih ada di transkrip.
  Utterance? evidenceFor(Obligation obligation) {
    if (obligation.evidenceId == null) return null;
    for (final u in transcript) {
      if (u.id == obligation.evidenceId) return u;
    }
    return null;
  }
}
