import 'package:record/record.dart';

import '../../../../core/config/app_config.dart';
import '../../../../core/error/failure.dart';

/// Menangkap mikrofon sebagai PCM16 mono 16 kHz — format yang sama
/// dengan AudioWorklet di web, supaya gateway tidak perlu bercabang.
class AudioDataSource {
  final AudioRecorder _recorder = AudioRecorder();

  Future<Stream<List<int>>> stream() async {
    if (!await _recorder.hasPermission()) {
      throw const PermissionFailure('Izin mikrofon ditolak');
    }
    try {
      return await _recorder.startStream(
        const RecordConfig(
          encoder: AudioEncoder.pcm16bits,
          sampleRate: AppConfig.sampleRate,
          numChannels: 1,
          echoCancel: true,
          noiseSuppress: true,
        ),
      );
    } catch (e) {
      throw AudioFailure('Gagal membuka mikrofon: $e');
    }
  }

  Future<void> stop() async {
    if (await _recorder.isRecording()) {
      await _recorder.stop();
    }
  }

  void dispose() => _recorder.dispose();
}
