/**
 * Alasan audio dianggap tidak layak dijadikan bukti kepatuhan.
 * String kosong berarti audio sehat.
 */
export type AudioQualityReason = string

/**
 * Alasan baku yang dikirim ke gateway.
 *
 * Kalimatnya tetap Bahasa Indonesia apa pun bahasa antarmukanya: gateway
 * menyiarkannya ulang apa adanya ke semua klien dan mencatatnya di log.
 * Klien menerjemahkannya saat menampilkan. Kembarannya ada di
 * `bisik_mobile/lib/core/audio/audio_quality.dart` — ubah keduanya.
 */
export const AUDIO_REASONS = {
  tooLoud: 'Suara terlalu keras dan pecah',
  tooQuiet: 'Suara terlalu pelan dari mikrofon',
  noisy: 'Kebisingan latar terlalu tinggi — matikan musik atau pindah ke tempat lebih tenang',
} as const

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
  /** `lang` BCP 47; bawaan `id-ID`, bahasa semua bisikan sesi sungguhan. */
  speak(text: string, lang?: string): void
  cancel(): void
}
