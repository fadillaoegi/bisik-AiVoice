import 'dart:math' as math;
import 'dart:typed_data';

/// Alasan baku yang dikirim ke gateway — kembaran `AUDIO_REASONS` di
/// `bisik_frontend/src/domain/repositories/audio_repository.ts`.
///
/// Kalimatnya tetap Bahasa Indonesia apa pun bahasa antarmukanya: gateway
/// menyiarkannya ulang apa adanya dan mencatatnya di log. Aplikasi
/// menerjemahkannya saat menampilkan (`AppStrings.audioReason`).
abstract final class AudioReasons {
  static const tooLoud = 'Suara terlalu keras dan pecah';
  static const tooQuiet = 'Suara terlalu pelan dari mikrofon';
  static const noisy =
      'Kebisingan latar terlalu tinggi — matikan musik atau pindah ke tempat lebih tenang';

  /// Dikirim backend lewat `evidence_skipped`, bukan hasil pengukuran klien.
  static const echo =
      'terdengar seperti gema bisikan aplikasi — pakai earphone';
}

/// Pemantau kualitas audio — kembaran `verdictFor` di
/// `bisik_frontend/src/infrastructure/audio/worklet_audio_repository.ts`.
///
/// Ambangnya sengaja disamakan persis dengan web. Kalau salah satu diubah,
/// ubah keduanya: gateway memakai vonis ini untuk MENAHAN checklist, dan dua
/// klien dengan ambang berbeda akan menilai percakapan yang sama secara
/// berbeda pula.
class AudioQualityMonitor {
  // Di bawah ini dianggap tidak ada yang bicara — bukan audio buruk, jadi
  // jangan divonis. Tanpa ini setiap jeda dilaporkan sebagai terlalu pelan.
  static const _silencePeak = 0.02;
  // Lebih dari 1% sample menyentuh skala penuh: suara pecah.
  static const _clipRatio = 0.01;
  static const _quietRms = 0.008;
  // Crest factor rendah + energi tinggi = bunyi rata terus-menerus.
  static const _noisyRms = 0.02;
  static const _noisyCrest = 2.2;

  static const _windowSamples = 16000; // 1 detik pada 16 kHz

  int _count = 0;
  double _sumSquares = 0;
  double _peak = 0;
  int _clipped = 0;
  String _lastReason = '';

  /// Menyerap satu potongan PCM16 little-endian.
  ///
  /// Mengembalikan alasan baru hanya ketika vonisnya BERUBAH; null berarti
  /// tidak ada yang perlu dilaporkan. Mengirim vonis yang sama berulang kali
  /// hanya membanjiri gateway dan UI.
  String? add(List<int> pcm) {
    final bytes = Uint8List.fromList(pcm);
    final samples = bytes.buffer.asInt16List(
      bytes.offsetInBytes,
      bytes.lengthInBytes ~/ 2,
    );

    for (final raw in samples) {
      final value = raw / 32768.0;
      final magnitude = value.abs();
      _count++;
      _sumSquares += value * value;
      if (magnitude > _peak) _peak = magnitude;
      if (magnitude >= 0.98) _clipped++;
    }
    if (_count < _windowSamples) return null;

    final reason = _verdict(
      rms: math.sqrt(_sumSquares / _count),
      peak: _peak,
      clippedRatio: _clipped / _count,
    );
    _count = 0;
    _sumSquares = 0;
    _peak = 0;
    _clipped = 0;

    if (reason == _lastReason) return null;
    _lastReason = reason;
    return reason;
  }

  static String _verdict({
    required double rms,
    required double peak,
    required double clippedRatio,
  }) {
    if (peak < _silencePeak) return '';
    if (clippedRatio > _clipRatio) return AudioReasons.tooLoud;
    if (rms < _quietRms) return AudioReasons.tooQuiet;
    if (rms > _noisyRms && peak / rms < _noisyCrest) {
      return AudioReasons.noisy;
    }
    return '';
  }
}
