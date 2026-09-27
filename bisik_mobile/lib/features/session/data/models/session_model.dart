import '../../domain/entities/session.dart';

class SessionModel {
  const SessionModel._();

  static Session fromJson(Map<String, dynamic> json) => Session(
        id: json['id'] as String,
        officerId: json['officer_id'] as String? ?? '',
        productId: json['product_id'] as String? ?? '',
        status: _status(json['status'] as String?),
        startedAt:
            DateTime.tryParse(json['started_at'] as String? ?? '') ?? DateTime.now(),
        score: json['score'] as int? ?? 0,
      );

  static SessionStatus _status(String? raw) => switch (raw) {
        'ended' => SessionStatus.ended,
        'aborted' => SessionStatus.aborted,
        _ => SessionStatus.active,
      };
}

Speaker speakerFromString(String? raw) => switch (raw) {
      'officer' => Speaker.officer,
      'customer' => Speaker.customer,
      _ => Speaker.unknown,
    };
