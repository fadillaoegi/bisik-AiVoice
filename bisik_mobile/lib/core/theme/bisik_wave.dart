import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'bisik_theme.dart';

/// Indikator memuat berbentuk gelombang suara lima batang.
///
/// Bentuknya sengaja diambil dari ikon Bisik: lima batang dengan tinggi
/// berbeda. Kebetulan yang berguna — lima batang juga jumlah butir kewajiban,
/// jadi indikatornya terbaca sebagai "sedang menyimak", bukan spinner generik
/// yang bisa dipakai aplikasi mana pun.
class BisikWave extends StatefulWidget {
  const BisikWave({
    super.key,
    this.height = 22,
    this.color = BisikColors.accent,
  });

  final double height;
  final Color color;

  @override
  State<BisikWave> createState() => _BisikWaveState();
}

class _BisikWaveState extends State<BisikWave>
    with SingleTickerProviderStateMixin {
  static const _bars = 5;

  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1100),
  )..repeat();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final barWidth = widget.height * 0.18;

    return SizedBox(
      height: widget.height,
      child: AnimatedBuilder(
        animation: _controller,
        builder: (context, _) {
          return Row(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              for (var i = 0; i < _bars; i++) ...[
                if (i > 0) SizedBox(width: barWidth * 0.7),
                Container(
                  width: barWidth,
                  height: widget.height * _scaleFor(i),
                  decoration: BoxDecoration(
                    color: widget.color,
                    borderRadius: BorderRadius.circular(barWidth),
                  ),
                ),
              ],
            ],
          );
        },
      ),
    );
  }

  /// Tiap batang bergerak dengan beda fase, jadi gelombangnya mengalir
  /// dari kiri ke kanan dan tidak berdenyut serempak.
  double _scaleFor(int index) {
    final phase = _controller.value * 2 * math.pi - index * 0.6;
    return 0.3 + 0.7 * (0.5 + 0.5 * math.sin(phase));
  }
}
