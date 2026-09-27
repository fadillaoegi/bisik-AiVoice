import type { SessionEvent } from '../../domain/entities/events'
import { tokenStore } from '../api/token_store'

// Lihat catatan di http_session_repository.ts soal `||` vs `??`.
const WS_BASE = import.meta.env.VITE_WS_URL ||
  `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}`

export type SocketRole = 'officer' | 'supervisor'

/**
 * Satu koneksi WebSocket ke gateway Go.
 * Arah keluar: frame PCM16 biner (hanya untuk role officer).
 * Arah masuk: event JSON (partial, utterance, obligation, violation, nudge).
 */
export class SessionSocket {
  private ws: WebSocket | null = null

  connect(
    sessionId: string,
    /**
     * Sisa kontrak untuk klien: peran dipakai hook untuk memutuskan siapa
     * mendengar bisikan. Server tidak memakainya sama sekali dan menentukan
     * peran dari token, supaya klien tidak bisa mengaku petugas sesi lain.
     */
    _role: SocketRole,
    handlers: {
      onEvent: (e: SessionEvent) => void
      onOpen?: () => void
      onClose?: () => void
      onError?: (e: Event) => void
    },
  ): void {
    // Token lewat query karena browser tidak mengizinkan header kustom pada
    // handshake WebSocket. Peran TIDAK dikirim lagi: server menentukannya
    // dari token, supaya klien tidak bisa mengaku petugas sesi orang lain.
    const token = tokenStore.read() ?? ''
    const ws = new WebSocket(
      `${WS_BASE}/ws?session_id=${sessionId}&token=${encodeURIComponent(token)}`,
    )
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => handlers.onOpen?.()
    ws.onclose = () => handlers.onClose?.()
    ws.onerror = (e) => handlers.onError?.(e)
    ws.onmessage = (msg) => {
      if (typeof msg.data !== 'string') return
      try {
        handlers.onEvent(JSON.parse(msg.data) as SessionEvent)
      } catch {
        // abaikan frame yang tidak bisa diparse
      }
    }

    this.ws = ws
  }

  sendAudio(pcm: ArrayBuffer): void {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(pcm)
  }

  beginSpeakerCalibration(): void {
    this.sendCommand({ type: 'begin_speaker_calibration' })
  }

  confirmSpeakerRoles(officerLabel: string, customerLabel: string): void {
    this.sendCommand({
      type: 'confirm_speaker_roles',
      officer_label: officerLabel,
      customer_label: customerLabel,
    })
  }

  /** Melaporkan kualitas audio; alasan kosong berarti audio kembali sehat. */
  reportAudioQuality(reason: string): void {
    this.sendCommand({ type: 'audio_quality', reason })
  }

  private sendCommand(command: Record<string, string>): void {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify(command))
  }

  close(): void {
    this.ws?.close()
    this.ws = null
  }

  get isOpen(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }
}
