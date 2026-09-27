import 'package:flutter/material.dart';

import '../../../../core/theme/bisik_theme.dart';
import '../providers/session_state.dart';

/// Kalibrasi dua suara, berurutan dan dipandu — kembaran dari
/// `SpeakerCalibration.tsx` di PWA.
///
/// Urutan bicara bukan identitas. Yang membuatnya sah di sini adalah
/// aplikasi yang MEMERINTAHKAN siapa bicara duluan, lalu manusia menekan
/// konfirmasi sebelum penilaian dimulai. Sampai itu terjadi, gateway
/// menahan seluruh scoring.
class SpeakerCalibration extends StatelessWidget {
  const SpeakerCalibration({
    super.key,
    required this.state,
    required this.onConfirm,
    required this.onRestart,
  });

  final SessionState state;
  final VoidCallback onConfirm;
  final VoidCallback onRestart;

  @override
  Widget build(BuildContext context) {
    // Kalau diarization merevisi label petugas menjadi label yang tidak lagi
    // ada, mundur ke langkah satu alih-alih berpegang pada label hantu.
    final officer = state.officerVoice != null &&
            state.recognisedVoices.contains(state.officerVoice)
        ? state.officerVoice
        : null;
    final customer = officer == null ? null : state.customerVoice;
    final step = officer == null ? 1 : (customer == null ? 2 : 3);

    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: BisikColors.accent.withValues(alpha: 0.07),
        border: Border.all(color: BisikColors.accent),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            step < 3
                ? 'LANGKAH KEAMANAN · BELUM DINILAI · LANGKAH $step DARI 2'
                : 'LANGKAH KEAMANAN · BELUM DINILAI',
            style: const TextStyle(
              color: BisikColors.accent,
              fontSize: 11,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.2,
            ),
          ),
          const SizedBox(height: 8),
          const Text(
            'Kenali dua suara',
            style: TextStyle(
              color: BisikColors.text,
              fontSize: 19,
              fontWeight: FontWeight.w700,
            ),
          ),

          if (step == 1) ...[
            const _Prompt(who: 'Petugas', trailing: ', ucapkan kalimat ini:'),
            const _Script('“Saya petugas yang menjalankan sesi ini.”'),
            const _Listening('Mendengarkan suara petugas…'),
          ] else
            _Registered(
              role: 'Petugas terdaftar',
              label: officer!,
              utterance: state.utteranceOf(officer),
            ),

          if (step == 2) ...[
            const _Prompt(who: 'Sekarang giliran nasabah', trailing: '. Ucapkan:'),
            const _Script('“Saya nasabah dan siap memulai.”'),
            if (state.duplicateVoice)
              const _Warning(
                'Suara itu sudah terdaftar sebagai petugas. Minta orang kedua '
                'yang berbicara — sistem perlu mendengar suara yang berbeda '
                'untuk bisa membedakan keduanya.',
              )
            else
              const _Listening('Mendengarkan suara nasabah…'),
          ],

          if (step == 3)
            _Registered(
              role: 'Nasabah terdaftar',
              label: customer!,
              utterance: state.utteranceOf(customer),
            ),

          if (state.calibrationError != null) ...[
            const SizedBox(height: 12),
            _Warning('Kalibrasi gagal: ${state.calibrationError}', bad: true),
          ],

          const SizedBox(height: 16),
          FilledButton(
            onPressed: step == 3 ? onConfirm : null,
            child: Padding(
              padding: const EdgeInsets.all(10),
              child: Text(step == 3
                  ? 'Konfirmasi dan mulai penilaian'
                  : 'Menunggu dua suara'),
            ),
          ),
          TextButton(
            onPressed: onRestart,
            child: const Text('Ulangi kalibrasi dari awal',
                style: TextStyle(color: BisikColors.accent)),
          ),
          const Text(
            'Ucapan kalibrasi tidak masuk laporan dan tidak dinilai.',
            textAlign: TextAlign.center,
            style: TextStyle(color: BisikColors.muted, fontSize: 12),
          ),
        ],
      ),
    );
  }
}

class _Prompt extends StatelessWidget {
  const _Prompt({required this.who, required this.trailing});
  final String who;
  final String trailing;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.only(top: 14, bottom: 6),
        child: RichText(
          text: TextSpan(
            style: const TextStyle(color: BisikColors.text, fontSize: 15),
            children: [
              TextSpan(
                  text: who,
                  style: const TextStyle(fontWeight: FontWeight.w700)),
              TextSpan(text: trailing),
            ],
          ),
        ),
      );
}

class _Script extends StatelessWidget {
  const _Script(this.text);
  final String text;

  @override
  Widget build(BuildContext context) => Container(
        width: double.infinity,
        margin: const EdgeInsets.only(bottom: 10),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        decoration: BoxDecoration(
          color: BisikColors.surface,
          border: Border.all(color: BisikColors.border),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(text,
            style: const TextStyle(
                color: BisikColors.accent,
                fontSize: 16,
                fontWeight: FontWeight.w600)),
      );
}

class _Listening extends StatelessWidget {
  const _Listening(this.text);
  final String text;

  @override
  Widget build(BuildContext context) =>
      Text(text, style: const TextStyle(color: BisikColors.warn, fontSize: 14));
}

class _Registered extends StatelessWidget {
  const _Registered({
    required this.role,
    required this.label,
    required this.utterance,
  });

  final String role;
  final String label;
  final String utterance;

  @override
  Widget build(BuildContext context) => Container(
        margin: const EdgeInsets.only(top: 12),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        decoration: BoxDecoration(
          color: BisikColors.good.withValues(alpha: 0.07),
          border: Border.all(color: BisikColors.good),
          borderRadius: BorderRadius.circular(10),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('✓',
                style: TextStyle(
                    color: BisikColors.good, fontWeight: FontWeight.w700)),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  RichText(
                    text: TextSpan(
                      style: const TextStyle(fontSize: 15),
                      children: [
                        TextSpan(
                          text: role,
                          style: const TextStyle(
                              color: BisikColors.good,
                              fontWeight: FontWeight.w700),
                        ),
                        TextSpan(
                            text: ' · Suara $label',
                            style: const TextStyle(color: BisikColors.text)),
                      ],
                    ),
                  ),
                  if (utterance.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 4),
                      child: Text(utterance,
                          style: const TextStyle(
                              color: BisikColors.muted, fontSize: 13)),
                    ),
                ],
              ),
            ),
          ],
        ),
      );
}

class _Warning extends StatelessWidget {
  const _Warning(this.text, {this.bad = false});
  final String text;
  final bool bad;

  @override
  Widget build(BuildContext context) {
    final color = bad ? BisikColors.bad : BisikColors.warn;
    return Container(
      margin: const EdgeInsets.only(top: 4),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        border: Border.all(color: color),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(text,
          style: TextStyle(color: color, fontSize: 13, height: 1.45)),
    );
  }
}
