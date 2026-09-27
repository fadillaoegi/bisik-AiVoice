import 'package:flutter/material.dart';

import '../../../../core/theme/bisik_theme.dart';
import '../../domain/entities/compliance.dart';

class ObligationTile extends StatelessWidget {
  const ObligationTile({super.key, required this.obligation});

  final Obligation obligation;

  @override
  Widget build(BuildContext context) {
    final (color, icon) = switch (obligation.status) {
      ObligationStatus.satisfied => (BisikColors.good, Icons.check_circle),
      ObligationStatus.violated => (BisikColors.bad, Icons.cancel),
      ObligationStatus.pending => (BisikColors.muted, Icons.circle_outlined),
    };

    return AnimatedContainer(
      duration: const Duration(milliseconds: 250),
      margin: const EdgeInsets.only(bottom: 6),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: BisikColors.surface,
        border: Border.all(color: color.withValues(alpha: 0.6)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(icon, size: 18, color: color),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              obligation.label,
              style: TextStyle(color: color, fontSize: 15),
            ),
          ),
          if (obligation.status == ObligationStatus.satisfied)
            Text(
              '${(obligation.confidence * 100).round()}%',
              style: const TextStyle(color: BisikColors.muted, fontSize: 12),
            ),
        ],
      ),
    );
  }
}
