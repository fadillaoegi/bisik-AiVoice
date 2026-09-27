enum Speaker { officer, customer, unknown }

enum SessionStatus { active, ended, aborted }

class Session {
  const Session({
    required this.id,
    required this.officerId,
    required this.productId,
    required this.status,
    required this.startedAt,
    this.score = 0,
  });

  final String id;
  final String officerId;
  final String productId;
  final SessionStatus status;
  final DateTime startedAt;
  final int score;

  bool get isRunning => status == SessionStatus.active;
}

class Utterance {
  const Utterance({
    required this.id,
    required this.speaker,
    required this.text,
    this.startMs = 0,
    this.revised = false,
  });

  final String id;
  final Speaker speaker;
  final String text;
  final int startMs;

  /// true kalau label pembicara pernah direvisi oleh diarization.
  final bool revised;

  Utterance copyWith({Speaker? speaker, bool? revised}) => Utterance(
        id: id,
        speaker: speaker ?? this.speaker,
        text: text,
        startMs: startMs,
        revised: revised ?? this.revised,
      );
}
