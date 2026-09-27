import 'package:flutter/material.dart';

import '../../../../core/theme/bisik_theme.dart';
import '../../domain/entities/compliance.dart';
import '../../domain/entities/session.dart';

/// Laporan berbukti setelah sesi berakhir.
///
/// Angka skor sendirian tidak berarti apa-apa bagi petugas maupun supervisor.
/// Yang membuatnya bisa dipertanggungjawabkan adalah kutipan di bawah tiap
/// butir: kalimat persis yang membuat butir itu dianggap terpenuhi.
class ReportView extends StatelessWidget {
  const ReportView({super.key, required this.report, required this.onReset});

  final ComplianceReport report;
  final VoidCallback onReset;

  @override
  Widget build(BuildContext context) {
    final score = report.session.score;
    final scoreColor = switch (score) {
      >= 80 => BisikColors.good,
      >= 50 => BisikColors.warn,
      _ => BisikColors.bad,
    };

    return ListView(
      padding: EdgeInsets.zero,
      children: [
        const Text(
          'LAPORAN KEPATUHAN',
          style: TextStyle(
            color: BisikColors.muted,
            fontSize: 11,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.6,
          ),
        ),
        const SizedBox(height: 12),
        Row(
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [
            Text(
              '$score',
              style: TextStyle(
                color: scoreColor,
                fontSize: 44,
                height: 1,
                fontWeight: FontWeight.w700,
              ),
            ),
            const Padding(
              padding: EdgeInsets.only(bottom: 6, left: 4),
              child: Text('/100',
                  style: TextStyle(color: BisikColors.muted, fontSize: 15)),
            ),
            const Spacer(),
            _Stat(
              value: '${report.satisfiedCount}/${report.obligations.length}',
              label: 'kewajiban',
            ),
            const SizedBox(width: 16),
            _Stat(
              value: '${report.violations.length}',
              label: 'pelanggaran',
              color: report.violations.isEmpty
                  ? BisikColors.good
                  : BisikColors.bad,
            ),
          ],
        ),
        const SizedBox(height: 24),

        const _SectionTitle('Bukti per kewajiban'),
        for (final o in report.obligations) _EvidenceTile(
          obligation: o,
          evidence: report.evidenceFor(o),
        ),

        if (report.violations.isNotEmpty) ...[
          const SizedBox(height: 20),
          const _SectionTitle('Pelanggaran'),
          for (final v in report.violations)
            Container(
              margin: const EdgeInsets.only(bottom: 6),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                border: Border.all(color: BisikColors.bad),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: Text('“${v.phrase}”',
                        style: const TextStyle(color: BisikColors.bad)),
                  ),
                  Text(v.severity,
                      style: const TextStyle(
                          color: BisikColors.muted, fontSize: 12)),
                ],
              ),
            ),
        ],

        const SizedBox(height: 20),
        const _SectionTitle('Transkrip'),
        if (report.transcript.isEmpty)
          const Text('Tidak ada ucapan yang tersimpan.',
              style: TextStyle(color: BisikColors.muted, fontSize: 14))
        else
          for (final u in report.transcript)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: RichText(
                text: TextSpan(
                  style: const TextStyle(
                      fontSize: 15, color: BisikColors.text, height: 1.4),
                  children: [
                    TextSpan(
                      text: '${_speaker(u.speaker)}  ',
                      style: TextStyle(
                        color: switch (u.speaker) {
                          Speaker.officer => BisikColors.officer,
                          Speaker.customer => BisikColors.customer,
                          Speaker.unknown => BisikColors.muted,
                        },
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    TextSpan(text: u.text),
                  ],
                ),
              ),
            ),

        const SizedBox(height: 24),
        FilledButton(
          onPressed: onReset,
          child: const Padding(
            padding: EdgeInsets.all(12),
            child: Text('Mulai sesi baru'),
          ),
        ),
        const SizedBox(height: 16),
      ],
    );
  }

  static String _speaker(Speaker s) => switch (s) {
        Speaker.officer => 'Petugas',
        Speaker.customer => 'Nasabah',
        Speaker.unknown => '—',
      };
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle(this.text);
  final String text;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.only(bottom: 10),
        child: Text(
          text,
          style: const TextStyle(
            color: BisikColors.muted,
            fontSize: 11,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.2,
          ),
        ),
      );
}

class _Stat extends StatelessWidget {
  const _Stat({required this.value, required this.label, this.color});

  final String value;
  final String label;
  final Color? color;

  @override
  Widget build(BuildContext context) => Column(
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          Text(value,
              style: TextStyle(
                  color: color ?? BisikColors.text,
                  fontSize: 18,
                  fontWeight: FontWeight.w700)),
          Text(label,
              style: const TextStyle(color: BisikColors.muted, fontSize: 12)),
        ],
      );
}

class _EvidenceTile extends StatelessWidget {
  const _EvidenceTile({required this.obligation, required this.evidence});

  final Obligation obligation;
  final Utterance? evidence;

  @override
  Widget build(BuildContext context) {
    final satisfied = obligation.status == ObligationStatus.satisfied;
    final color = satisfied ? BisikColors.good : BisikColors.warn;

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: BisikColors.surface,
        border: Border(left: BorderSide(color: color, width: 3)),
        borderRadius: const BorderRadius.horizontal(right: Radius.circular(8)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(satisfied ? Icons.check_circle : Icons.remove_circle_outline,
                  size: 18, color: color),
              const SizedBox(width: 10),
              Expanded(
                child: Text(obligation.label,
                    style: const TextStyle(
                        color: BisikColors.text, fontSize: 15)),
              ),
              if (satisfied)
                Text('${(obligation.confidence * 100).round()}%',
                    style: const TextStyle(
                        color: BisikColors.muted, fontSize: 12)),
            ],
          ),
          if (evidence != null) ...[
            const SizedBox(height: 10),
            Container(
              padding: const EdgeInsets.only(left: 12),
              decoration: const BoxDecoration(
                border: Border(
                    left: BorderSide(color: BisikColors.border, width: 2)),
              ),
              child: Text('“${evidence!.text}”',
                  style: const TextStyle(
                      color: BisikColors.text, fontSize: 14, height: 1.4)),
            ),
          ] else if (!satisfied) ...[
            const SizedBox(height: 8),
            const Text('Tidak pernah disampaikan.',
                style: TextStyle(color: BisikColors.muted, fontSize: 13)),
          ],
        ],
      ),
    );
  }
}
