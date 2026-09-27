import type {
  AudioQualityReason,
  AudioRepository,
} from '../../domain/repositories/audio_repository'

/**
 * Menyalurkan frame PCM16 dari mikrofon ke gateway.
 * Use case ini tidak tahu apa-apa soal WebSocket — hanya menerima sink.
 */
export class StreamAudioUseCase {
  private readonly audio: AudioRepository

  constructor(audio: AudioRepository) {
    this.audio = audio
  }

  start(
    sink: (pcm: ArrayBuffer) => void,
    onQuality?: (reason: AudioQualityReason) => void,
  ): Promise<void> {
    return this.audio.start(sink, onQuality)
  }

  stop(): Promise<void> {
    return this.audio.stop()
  }
}
