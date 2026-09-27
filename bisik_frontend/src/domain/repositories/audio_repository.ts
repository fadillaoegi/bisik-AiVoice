/**
 * Alasan audio dianggap tidak layak dijadikan bukti kepatuhan.
 * String kosong berarti audio sehat.
 */
export type AudioQualityReason = string

/** Pemrosesan audio yang benar-benar aktif di perangkat, bukan yang diminta. */
export interface AudioProcessing {
  noiseSuppression: boolean
  echoCancellation: boolean
  autoGainControl: boolean
  voiceIsolation: boolean
}

/** Kontrak penangkapan mikrofon + pengiriman frame PCM16. */
export interface AudioRepository {
  start(
    onFrame: (pcm: ArrayBuffer) => void,
    onQuality?: (reason: AudioQualityReason) => void,
  ): Promise<void>
  stop(): Promise<void>
  isRunning(): boolean
  /** Terisi setelah start(); dibaca balik dari track mikrofon. */
  readonly processing: AudioProcessing
}

/** Kontrak bisikan suara ke earpiece petugas. */
export interface SpeechRepository {
  speak(text: string): void
  cancel(): void
}
