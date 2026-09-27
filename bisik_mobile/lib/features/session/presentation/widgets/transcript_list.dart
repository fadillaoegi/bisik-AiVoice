import 'package:flutter/material.dart';

import '../../../../core/theme/bisik_theme.dart';
import '../../domain/entities/session.dart';

class TranscriptList extends StatelessWidget {
  const TranscriptList({super.key, required this.utterances, this.partial});

  final List<Utterance> utterances;
  final String? partial;

  static const _labels = {
    Speaker.officer: 'Petugas',
    Speaker.customer: 'Nasabah',
    Speaker.unknown: '—',
  };

  static const _colors = {
    Speaker.officer: BisikColors.officer,
    Speaker.customer: BisikColors.customer,
    Speaker.unknown: BisikColors.muted,
  };

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: BisikColors.surface,
        border: Border.all(color: BisikColors.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: ListView.builder(
        reverse: true,
        itemCount: utterances.length + (partial == null ? 0 : 1),
        itemBuilder: (context, index) {
          if (partial != null && index == 0) {
            return Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                partial!,
                style: const TextStyle(
                  color: BisikColors.muted,
                  fontStyle: FontStyle.italic,
                  fontSize: 13,
                ),
              ),
            );
          }
          final offset = partial == null ? 0 : 1;
          final u = utterances[utterances.length - 1 - (index - offset)];

          return Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: RichText(
              text: TextSpan(
                style: const TextStyle(fontSize: 15, color: BisikColors.text),
                children: [
                  TextSpan(
                    text: '${_labels[u.speaker]}  ',
                    style: TextStyle(
                      color: _colors[u.speaker],
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  TextSpan(text: u.text),
                  if (u.revised)
                    const TextSpan(
                      text: '  (label direvisi)',
                      style: TextStyle(color: BisikColors.warn, fontSize: 11),
                    ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}
