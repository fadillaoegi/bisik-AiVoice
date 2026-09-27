import { useCallback, useEffect, useRef, useState } from 'react'
import { container } from '../../infrastructure/di/container'
import type { SessionEvent } from '../../domain/entities/events'
import type { SocketRole } from '../../infrastructure/ws/session_socket'
import { useAppDispatch } from './redux'
import { connectionChanged, recordingChanged, sessionFailed } from '../../application/store/slices/sessionSlice'
import {
  partialReceived,
  speakerRevised,
  utteranceAppended,
} from '../../application/store/slices/transcriptSlice'
import {
  nudgeReceived,
  obligationSatisfied,
  violationDetected,
} from '../../application/store/slices/complianceSlice'

/**
 * Menyambungkan WebSocket ke store, dan — khusus role officer —
 * mengalirkan frame mikrofon ke gateway.
 */
export function useSessionStream(sessionId: string | null, role: SocketRole) {
  const dispatch = useAppDispatch()
  const startedRef = useRef(false)
  const [calibrationStatus, setCalibrationStatus] = useState<'idle' | 'collecting' | 'confirmed'>(
    'idle',
  )
  // Disimpan per utterance, bukan per label. Diarization bisa MENGOREKSI
  // label sebuah ucapan belakangan — dan di detik-detik awal sesi itu justru
  // hal yang lumrah. Kalau kunci penyimpanannya label, koreksi tidak bisa
  // dipetakan ke contoh mana pun dan suara kedua tidak pernah muncul.
  const [calibrationSamples, setCalibrationSamples] = useState<
    Array<{ id: string; sourceSpeaker: string; text: string }>
  >([])
  const [calibrationError, setCalibrationError] = useState<string | null>(null)
  // Label suara yang sudah didaftarkan sebagai petugas. Diisi oleh suara
  // PERTAMA yang dikenali, karena kalibrasi kini dipandu langkah demi
  // langkah: aplikasi menyuruh petugas bicara dulu, baru nasabah.
  const [officerVoice, setOfficerVoice] = useState<string | null>(null)
  // Menyala ketika suara yang sama bicara lagi di langkah kedua. Tanpa
  // penanda ini petugas menunggu tanpa tahu kenapa layarnya diam.
  const [duplicateVoice, setDuplicateVoice] = useState(false)
  const [audioWarning, setAudioWarning] = useState<string | null>(null)
  // Berapa ucapan yang sengaja tidak dihitung sebagai bukti. Angka ini harus
  // terlihat petugas: checklist yang diam bukan berarti sistemnya rusak.
  const [excludedCount, setExcludedCount] = useState(0)
  // Apa yang benar-benar aktif di mikrofon. Dipakai saat menguji di lapangan:
  // kalau peredam bawaan ternyata mati, itu penjelasan pertama kenapa
  // transkrip berantakan — dan tanpa ditampilkan, tidak ada yang tahu.
  const [micProcessing, setMicProcessing] = useState<string | null>(null)

  const handleEvent = useCallback(
    (e: SessionEvent) => {
      switch (e.type) {
        case 'partial':
          dispatch(partialReceived({ speaker: e.speaker, text: e.text }))
          break
        case 'utterance':
          dispatch(
            utteranceAppended({
              id: e.id, speaker: e.speaker, text: e.text, startMs: 0, revised: false,
              sourceSpeaker: e.source_speaker,
            }),
          )
          break
        case 'speaker_revised':
          dispatch(speakerRevised({ id: e.utterance_id, speaker: e.speaker }))
          break
        case 'speaker_calibration_started':
          setCalibrationStatus('collecting')
          setCalibrationSamples([])
          setCalibrationError(null)
          setOfficerVoice(null)
          setDuplicateVoice(false)
          break
        case 'speaker_calibration_partial':
          break
        case 'speaker_calibration_utterance':
          // Langkah 1 selesai saat suara pertama dikenali; sesudah itu, suara
          // yang sama berarti orang kedua belum bicara.
          if (e.source_speaker) {
            setOfficerVoice((registered) => {
              if (registered === null) {
                setDuplicateVoice(false)
                return e.source_speaker
              }
              setDuplicateVoice(e.source_speaker === registered)
              return registered
            })
          }
          setCalibrationSamples((current) => {
            const existing = current.find((sample) => sample.id === e.utterance_id)
            // Revisi tidak selalu membawa teks; pertahankan teks lama.
            const merged = {
              id: e.utterance_id,
              sourceSpeaker: e.source_speaker,
              text: e.text || existing?.text || '',
            }
            return existing
              ? current.map((s) => (s.id === e.utterance_id ? merged : s))
              : [...current, merged]
          })
          break
        case 'speaker_roles_confirmed':
          setCalibrationStatus('confirmed')
          setCalibrationError(null)
          break
        case 'speaker_calibration_error':
          setCalibrationError(e.message)
          break
        case 'obligation_satisfied':
          dispatch(
            obligationSatisfied({
              code: e.code, confidence: e.confidence, evidenceId: e.evidence_id,
            }),
          )
          break
        case 'violation':
          dispatch(
            violationDetected({
              phrase: e.phrase, severity: e.severity,
              evidenceId: e.evidence_id, detectedAt: new Date().toISOString(),
            }),
          )
          break
        case 'audio_quality':
          setAudioWarning(e.degraded ? e.reason : null)
          break
        case 'speaker_unknown':
        case 'evidence_skipped':
          setExcludedCount((n) => n + 1)
          break
        case 'session_error':
          // Jalur audio mati: hentikan indikator merekam dan tampilkan
          // sebabnya, jangan biarkan UI terlihat masih menyimak.
          dispatch(sessionFailed(e.message))
          break
        case 'nudge':
          dispatch(nudgeReceived(e.text))
          // Hanya petugas yang mendengar bisikan.
          if (role === 'officer') container.repositories.speech.speak(e.text)
          break
      }
    },
    [dispatch, role],
  )

  useEffect(() => {
    if (!sessionId || startedRef.current) return
    startedRef.current = true

    container.socket.connect(sessionId, role, {
      onEvent: handleEvent,
      onOpen: () => {
        dispatch(connectionChanged(true))
        if (role !== 'officer') return
        // Perintah dikirim sebelum mikrofon mulai agar frame pertama pun
        // diperlakukan sebagai kalibrasi, bukan otomatis sebagai petugas.
        container.socket.beginSpeakerCalibration()
        container.usecases.streamAudio
          .start(
            (pcm) => container.socket.sendAudio(pcm),
            (reason) => container.socket.reportAudioQuality(reason),
          )
          .then(() => {
            dispatch(recordingChanged(true))
            const p = container.repositories.audio.processing
            const on = [
              p.noiseSuppression && 'peredam bising',
              p.voiceIsolation && 'isolasi suara',
              p.echoCancellation && 'peredam gema',
            ].filter(Boolean)
            setMicProcessing(on.length > 0 ? on.join(' · ') : null)
          })
          .catch((err: Error) => dispatch(sessionFailed(err.message)))
      },
      onClose: () => dispatch(connectionChanged(false)),
      onError: () => dispatch(sessionFailed('Koneksi gateway terputus')),
    })

    return () => {
      startedRef.current = false
      container.repositories.speech.cancel()
      // Socket ditutup SETELAH mikrofon berhenti: stop() masih mengirim
      // sisa buffer worklet, dan frame itu ikut hilang kalau soket sudah mati.
      void container.usecases.streamAudio
        .stop()
        .finally(() => container.socket.close())
    }
  }, [sessionId, role, handleEvent, dispatch])

  const confirmSpeakerRoles = useCallback((officerLabel: string, customerLabel: string) => {
    setCalibrationError(null)
    container.socket.confirmSpeakerRoles(officerLabel, customerLabel)
  }, [])

  /** Mulai ulang kalibrasi dari nol — backend ikut membuang mapping lama. */
  const restartCalibration = useCallback(() => {
    setCalibrationSamples([])
    setCalibrationError(null)
    setOfficerVoice(null)
    setDuplicateVoice(false)
    container.socket.beginSpeakerCalibration()
  }, [])

  return {
    calibrationStatus,
    calibrationSamples,
    calibrationError,
    confirmSpeakerRoles,
    restartCalibration,
    officerVoice,
    duplicateVoice,
    audioWarning,
    excludedCount,
    micProcessing,
  }
}
