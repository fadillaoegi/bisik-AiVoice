import 'package:flutter/material.dart';

import '../../../../core/theme/bisik_theme.dart';
import '../../domain/entities/compliance.dart';

/// Kalimat perintah per butir kewajiban.
///
/// Sengaja tidak memakai deskripsi dari backend: yang di sana adalah prompt
/// untuk semantic matcher, kalimat orang ketiga yang deskriptif. Petugas yang
/// sedang bicara butuh instruksi langsung dan pendek.
const _hint = <String, String>{
  'IDENTITY': 'Sebutkan nama kamu dan nama lembaga tempatmu bekerja.',
  'RATE': 'Sebutkan suku bunga atau total biaya yang harus dibayar.',
  'TENOR': 'Sebutkan jangka waktu dan besar cicilan per bulan.',
  'PENALTY': 'Jelaskan denda kalau nasabah telat membayar.',
  'RIGHT': 'Beri tahu nasabah berhak menolak atau membatalkan.',
};

/// Satu butir besar + titik progres — versi Flutter dari kartu fokus web.
///
/// Petugas menatap nasabah, bukan layar. Satu lirikan harus cukup untuk tahu
/// apa yang belum disampaikan, jadi hanya kewajiban berikutnya yang tampil
/// besar; sisanya cukup jadi titik.
class ObligationFocus extends StatelessWidget {
  const ObligationFocus({super.key, required this.obligations});

  final List<Obligation> obligations;

  @override
  Widget build(BuildContext context) {
    // Nol butir berarti daftarnya belum termuat, BUKAN semuanya terpenuhi.
    // "Lengkap" pada checklist kosong adalah klaim palsu — kebalikan dari
    // gunanya produk ini.
    if (obligations.isEmpty) {
      return Container(
        padding: const EdgeInsets.fromLTRB(20, 22, 20, 22),
        decoration: BoxDecoration(
          color: BisikColors.surface,
          border: Border.all(color: BisikColors.border),
          borderRadius: BorderRadius.circular(16),
        ),
        child: const Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('MEMUAT KEWAJIBAN',
                style: TextStyle(
                    color: BisikColors.muted,
                    fontSize: 11,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 1.6)),
            SizedBox(height: 10),
            Text('Menyiapkan…',
                style: TextStyle(
                    color: BisikColors.muted,
                    fontSize: 28,
                    fontWeight: FontWeight.w700)),
          ],
        ),
      );
    }

    final next = obligations
        .where((o) => o.status == ObligationStatus.pending)
        .firstOrNull;
    final satisfied = obligations
        .where((o) => o.status == ObligationStatus.satisfied)
        .length;
    final accent = next == null ? BisikColors.good : BisikColors.accent;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        AnimatedContainer(
          duration: const Duration(milliseconds: 250),
          padding: const EdgeInsets.fromLTRB(20, 22, 20, 22),
          decoration: BoxDecoration(
            color: next == null
                ? BisikColors.good.withValues(alpha: 0.07)
                : BisikColors.surface,
            border: Border.all(
              color: next == null ? BisikColors.good : BisikColors.border,
            ),
            borderRadius: BorderRadius.circular(16),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                next == null
                    ? 'SEMUA KEWAJIBAN TERPENUHI'
                    : 'BELUM DISAMPAIKAN',
                style: const TextStyle(
                  color: BisikColors.muted,
                  fontSize: 11,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 1.6,
                ),
              ),
              const SizedBox(height: 10),
              Text(
                next?.label ?? 'Lengkap',
                style: TextStyle(
                  color: accent,
                  fontSize: 28,
                  height: 1.15,
                  fontWeight: FontWeight.w700,
                ),
              ),
              if (next != null) ...[
                const SizedBox(height: 10),
                Text(
                  _hint[next.code] ?? '',
                  style: const TextStyle(
                    color: BisikColors.text,
                    fontSize: 15,
                  ),
                ),
              ],
            ],
          ),
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            for (final o in obligations)
              Padding(
                padding: const EdgeInsets.only(right: 7),
                child: _Dot(status: o.status),
              ),
            const SizedBox(width: 5),
            Text(
              '$satisfied dari ${obligations.length}',
              style: const TextStyle(color: BisikColors.muted, fontSize: 14),
            ),
          ],
        ),
      ],
    );
  }
}

class _Dot extends StatelessWidget {
  const _Dot({required this.status});

  final ObligationStatus status;

  @override
  Widget build(BuildContext context) {
    final color = switch (status) {
      ObligationStatus.satisfied => BisikColors.good,
      ObligationStatus.violated => BisikColors.bad,
      ObligationStatus.pending => BisikColors.border,
    };
    return Container(
      width: 13,
      height: 13,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: status == ObligationStatus.pending ? null : color,
        border: Border.all(color: color, width: 2),
      ),
    );
  }
}
