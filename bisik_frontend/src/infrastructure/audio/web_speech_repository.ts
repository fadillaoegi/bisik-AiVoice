import type { SpeechRepository } from '../../domain/repositories/audio_repository'

/**
 * Bisikan dieksekusi di sisi klien lewat Web Speech API, bukan streaming
 * audio dari server. Latensi lebih rendah, tidak ada bandwidth audio balik,
 * dan suaranya keluar lewat earpiece perangkat petugas — nasabah tidak dengar.
 */
export class WebSpeechRepository implements SpeechRepository {
  speak(text: string): void {
    if (!('speechSynthesis' in window)) return
    const u = new SpeechSynthesisUtterance(text)
    u.lang = 'id-ID'
    u.rate = 1.15
    window.speechSynthesis.speak(u)
  }

  cancel(): void {
    if ('speechSynthesis' in window) window.speechSynthesis.cancel()
  }
}
