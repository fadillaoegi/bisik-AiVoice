import 'package:flutter_tts/flutter_tts.dart';

/// Bisikan diucapkan di perangkat petugas, bukan di-streaming dari server:
/// latensi lebih rendah dan suaranya keluar lewat earpiece —
/// nasabah tidak mendengar.
class TtsDataSource {
  TtsDataSource() {
    _tts
      ..setLanguage('id-ID')
      ..setSpeechRate(0.55)
      ..setVolume(1.0);
  }

  final FlutterTts _tts = FlutterTts();

  Future<void> speak(String text) => _tts.speak(text);

  Future<void> stop() => _tts.stop();
}
