import '../../domain/entities/compliance.dart';
import '../../domain/entities/session.dart';
import 'session_model.dart';

class ObligationModel {
  const ObligationModel._();

  static Obligation fromJson(Map<String, dynamic> json) => Obligation(
        code: json['code'] as String,
        label: json['label'] as String? ?? '',
        status: switch (json['status'] as String?) {
          'satisfied' => ObligationStatus.satisfied,
          'violated' => ObligationStatus.violated,
          _ => ObligationStatus.pending,
        },
        confidence: (json['confidence'] as num?)?.toDouble() ?? 0,
        evidenceId: json['evidence_id'] as String?,
      );
}

class ViolationModel {
  const ViolationModel._();

  static Violation fromJson(Map<String, dynamic> json) => Violation(
        phrase: json['phrase'] as String? ?? '',
        severity: json['severity'] as String? ?? 'medium',
        evidenceId: json['evidence_id'] as String? ?? '',
        detectedAt:
            DateTime.tryParse(json['detected_at'] as String? ?? '') ??
                DateTime.now(),
      );
}

class UtteranceModel {
  const UtteranceModel._();

  static Utterance fromJson(Map<String, dynamic> json) => Utterance(
        id: json['id'] as String? ?? '',
        speaker: speakerFromString(json['speaker'] as String?),
        text: json['text'] as String? ?? '',
        startMs: json['start_ms'] as int? ?? 0,
        revised: json['revised'] as bool? ?? false,
      );
}

class ComplianceReportModel {
  const ComplianceReportModel._();

  static ComplianceReport fromJson(Map<String, dynamic> json) {
    List<Map<String, dynamic>> list(String key) =>
        ((json[key] as List<dynamic>?) ?? const [])
            .cast<Map<String, dynamic>>();

    return ComplianceReport(
      session: SessionModel.fromJson(json['session'] as Map<String, dynamic>),
      obligations: list('obligations').map(ObligationModel.fromJson).toList(),
      violations: list('violations').map(ViolationModel.fromJson).toList(),
      transcript: list('transcript').map(UtteranceModel.fromJson).toList(),
    );
  }
}
