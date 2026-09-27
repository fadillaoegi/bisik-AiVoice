import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/bisik_theme.dart';
import '../../../../core/theme/bisik_wave.dart';
import '../../../auth/presentation/providers/auth_providers.dart';
import '../providers/session_controller.dart';
import '../providers/session_state.dart';
import '../widgets/obligation_focus.dart';
import '../widgets/obligation_tile.dart';
import '../widgets/report_view.dart';
import '../widgets/speaker_calibration.dart';
import '../widgets/transcript_list.dart';

class OfficerPage extends ConsumerStatefulWidget {
  const OfficerPage({super.key});

  @override
  ConsumerState<OfficerPage> createState() => _OfficerPageState();
}

class _OfficerPageState extends ConsumerState<OfficerPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(sessionControllerProvider.notifier).loadObligations();
    });
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(sessionControllerProvider);
    final controller = ref.read(sessionControllerProvider.notifier);
    final running = state.session?.isRunning ?? false;
    // Selama sesi berjalan skornya perkiraan; begitu laporan tiba, angka
    // resmi dari backend yang dipakai — itu yang dihitung setelah revisi
    // label terakhir diproses.
    final score = state.report?.session.score ?? state.liveScore;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Bisik'),
        actions: [
          // Keluar hanya saat tidak ada sesi berjalan: menutup sesi di tengah
          // percakapan akan membuang skor yang belum sempat dihitung.
          if (!running)
            IconButton(
              tooltip: 'Keluar',
              icon: const Icon(Icons.logout, size: 20),
              color: BisikColors.muted,
              onPressed: ref.read(authControllerProvider.notifier).logout,
            ),
          Padding(
            padding: const EdgeInsets.only(right: 16),
            child: Center(
              child: Text(
                '$score/100',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.bold,
                  color: switch (score) {
                    >= 80 => BisikColors.good,
                    >= 50 => BisikColors.warn,
                    _ => BisikColors.bad,
                  },
                ),
              ),
            ),
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        // Tiga layar berurutan: belum mulai → sesi berjalan → laporan.
        // Sebelumnya laporan tidak ada sama sekali, jadi setelah sesi
        // berakhir aplikasi langsung kembali ke layar awal dan payoff
        // produknya — bukti kepatuhan — hilang di mobile.
        child: running
            ? _buildActive(state, controller)
            : state.loadingReport
            ? _buildLoadingReport()
            : state.report != null
            ? ReportView(report: state.report!, onReset: controller.reset)
            : _buildIdle(state, controller),
      ),
    );
  }

  Widget _buildLoadingReport() => const Center(
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        BisikWave(height: 26),
        SizedBox(height: 16),
        Text(
          'Menyiapkan laporan berbukti…',
          style: TextStyle(color: BisikColors.muted, fontSize: 15),
        ),
      ],
    ),
  );

  Widget _buildIdle(SessionState state, SessionController controller) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const SizedBox(height: 32),
        // Identitas petugas datang dari akun yang masuk, bukan dari isian.
        // Kolom "ID Petugas" yang lama dihapus: kalau petugas boleh mengetik
        // ID siapa pun, laporan kepatuhan tidak membuktikan siapa pelakunya.
        Consumer(
          builder: (context, ref, _) {
            final name = ref.watch(
              authControllerProvider.select((auth) => auth.user?.name),
            );
            return Text(
              name == null ? '' : 'Masuk sebagai $name',
              style: const TextStyle(color: BisikColors.muted, fontSize: 15),
            );
          },
        ),
        const SizedBox(height: 20),
        FilledButton(
          // Dinonaktifkan selama proses berjalan: membuka sesi butuh HTTP,
          // WebSocket, dan izin mikrofon, dan ketukan kedua akan membuat
          // sesi kedua yang tidak pernah dipakai.
          onPressed: state.starting
              ? null
              : () => controller.start('KREDIT-MULTIGUNA'),
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: state.starting
                ? const Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      BisikWave(height: 18, color: BisikColors.accentInk),
                      SizedBox(width: 12),
                      Text('Membuka sesi…'),
                    ],
                  )
                : const Text('Mulai sesi'),
          ),
        ),
        // Kegagalan di sini dulu tidak pernah terlihat: error tersimpan di
        // state tetapi layar idle tidak pernah menampilkannya, jadi tombolnya
        // tampak mati padahal sebenarnya gagal menghubungi gateway.
        if (state.error != null) ...[
          const SizedBox(height: 16),
          _Banner(text: state.error!, color: BisikColors.bad),
        ],
      ],
    );
  }

  Widget _buildActive(SessionState state, SessionController controller) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Icon(
              Icons.circle,
              size: 8,
              color: state.connected ? BisikColors.good : BisikColors.muted,
            ),
            const SizedBox(width: 8),
            Text(
              '${state.connected ? "Terhubung" : "Menyambung…"} · '
              '${state.recording ? "merekam" : "mic mati"}',
              style: const TextStyle(color: BisikColors.muted, fontSize: 13),
            ),
          ],
        ),
        if (state.error != null) ...[
          const SizedBox(height: 12),
          _Banner(text: state.error!, color: BisikColors.bad),
        ],
        // Bagian yang tidak dihitung sebagai bukti harus terlihat: checklist
        // yang diam tanpa penjelasan membuat petugas mengira sistemnya rusak.
        if (state.warning != null) ...[
          const SizedBox(height: 12),
          _Banner(text: state.warning!, color: BisikColors.warn),
        ],
        if (state.lastNudge != null) ...[
          const SizedBox(height: 12),
          _Banner(
            text: state.lastNudge!,
            color: BisikColors.warn,
            icon: Icons.volume_up,
          ),
        ],
        const SizedBox(height: 16),

        // Sebelum peran dikunci, kalibrasi menggantikan checklist. Menampilkan
        // checklist lebih dulu memberi kesan penilaian sudah berjalan,
        // padahal gateway masih menahan seluruh scoring.
        if (state.calibration != CalibrationStatus.confirmed)
          Expanded(
            child: SingleChildScrollView(
              child: SpeakerCalibration(
                state: state,
                onConfirm: controller.confirmSpeakerRoles,
                onRestart: controller.restartCalibration,
              ),
            ),
          )
        else ...[
          ObligationFocus(obligations: state.obligations),
          const SizedBox(height: 8),
          // Rincian dan transkrip diturunkan ke balik disclosure: saat sesi
          // berjalan keduanya mengganggu, saat meninjau baru berguna.
          Expanded(
            child: ListView(
              padding: EdgeInsets.zero,
              children: [
                _Disclosure(
                  title: 'Rincian kewajiban',
                  child: Column(
                    children: [
                      for (final o in state.obligations)
                        ObligationTile(obligation: o),
                    ],
                  ),
                ),
                _Disclosure(
                  title: 'Transkrip',
                  child: SizedBox(
                    height: 260,
                    child: TranscriptList(
                      utterances: state.utterances,
                      partial: state.partial,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
        const SizedBox(height: 12),
        OutlinedButton(
          onPressed: controller.end,
          style: OutlinedButton.styleFrom(
            foregroundColor: BisikColors.bad,
            side: const BorderSide(color: BisikColors.bad),
          ),
          child: const Padding(
            padding: EdgeInsets.all(12),
            child: Text('Akhiri sesi'),
          ),
        ),
      ],
    );
  }
}

class _Banner extends StatelessWidget {
  const _Banner({required this.text, required this.color, this.icon});

  final String text;
  final Color color;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        border: Border.all(color: color),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          if (icon != null) ...[
            Icon(icon, size: 18, color: color),
            const SizedBox(width: 10),
          ],
          Expanded(
            child: Text(
              text,
              style: TextStyle(color: color, fontWeight: FontWeight.w600),
            ),
          ),
        ],
      ),
    );
  }
}

class _Disclosure extends StatelessWidget {
  const _Disclosure({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Theme(
      data: Theme.of(context).copyWith(dividerColor: BisikColors.border),
      child: ExpansionTile(
        title: Text(
          title,
          style: const TextStyle(color: BisikColors.muted, fontSize: 14),
        ),
        iconColor: BisikColors.muted,
        collapsedIconColor: BisikColors.muted,
        tilePadding: EdgeInsets.zero,
        childrenPadding: const EdgeInsets.only(bottom: 12),
        children: [child],
      ),
    );
  }
}
