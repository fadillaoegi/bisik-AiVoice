import { useEffect, useRef, useState } from 'react'
import { container } from '../../../infrastructure/di/container'
import { useAppDispatch, useAppSelector } from '../../hooks/redux'
import { useSessionStream } from '../../hooks/useSessionStream'
import {
  sessionEnded,
  sessionFailed,
  sessionReset,
  sessionStarted,
  connectionChanged,
  recordingChanged,
} from '../../../application/store/slices/sessionSlice'
import {
  complianceReset,
  nudgeReceived,
  obligationSatisfied,
  obligationsLoaded,
  violationDetected,
} from '../../../application/store/slices/complianceSlice'
import {
  speakerRevised,
  transcriptCleared,
  utteranceAppended,
} from '../../../application/store/slices/transcriptSlice'
import type { ComplianceReport } from '../../../domain/entities/compliance'
import {
  createDemoReport,
  createDemoSession,
  demoObligations,
  demoMislabeledUtterance,
  demoUtterances,
  demoViolation,
} from '../../../application/demo/demo_scenario'
import {
  selectConnected,
  selectLastNudge,
  selectLiveScore,
  selectObligations,
  selectPartial,
  selectRecording,
  selectAuthRestoring,
  selectAuthUser,
  selectSession,
  selectSessionError,
  selectUtterances,
} from '../../../application/store/selectors'
import { ObligationList } from '../../components/ObligationList'
import { ObligationFocus } from '../../components/ObligationFocus'
import { BisikWave } from '../../components/BisikWave'
import { LoginForm } from '../../components/LoginForm'
import { loggedOut } from '../../../application/store/slices/authSlice'
import { TranscriptView } from '../../components/TranscriptView'
import { ScoreBadge } from '../../components/ScoreBadge'
import { ReportView } from '../../components/ReportView'
import { SpeakerCalibration } from '../../components/SpeakerCalibration'

export function OfficerPage() {
  const dispatch = useAppDispatch()
  const session = useAppSelector(selectSession)
  const connected = useAppSelector(selectConnected)
  const recording = useAppSelector(selectRecording)
  const sessionError = useAppSelector(selectSessionError)
  const obligations = useAppSelector(selectObligations)
  const utterances = useAppSelector(selectUtterances)
  const partial = useAppSelector(selectPartial)
  const nudge = useAppSelector(selectLastNudge)
  const score = useAppSelector(selectLiveScore)

  const authUser = useAppSelector(selectAuthUser)
  const authRestoring = useAppSelector(selectAuthRestoring)
  // Membuka sesi butuh HTTP lalu WebSocket lalu izin mikrofon. Tanpa penanda
  // ini tombolnya terasa mati selama proses berjalan.
  const [starting, setStarting] = useState(false)
  // Gagal memuat daftar kewajiban BUKAN kegagalan sesi. Demo terarah tetap
  // bisa diputar tanpa backend, jadi jangan sambut pengunjung dengan error
  // merah yang membuat mereka mengira aplikasinya rusak.
  const [gatewayDown, setGatewayDown] = useState(false)
  const [report, setReport] = useState<ComplianceReport | null>(null)
  const [reportError, setReportError] = useState<string | null>(null)
  const [demoMode, setDemoMode] = useState(false)
  const demoTimers = useRef<number[]>([])

  // Endpoint ini butuh login, jadi jangan dipanggil sebelum ada sesi login —
  // kalau tidak, 401 akan dilaporkan sebagai "gateway tidak bisa dihubungi".
  useEffect(() => {
    if (!authUser) return
    container.repositories.session
      .obligations()
      .then((o) => {
        dispatch(obligationsLoaded(o))
        setGatewayDown(false)
      })
      .catch(() => setGatewayDown(true))
  }, [dispatch, authUser])

  const stream = useSessionStream(
    session?.status === 'active' && !demoMode ? session.id : null,
    'officer',
  )

  function clearDemoTimers() {
    demoTimers.current.forEach((timer) => window.clearTimeout(timer))
    demoTimers.current = []
    container.repositories.speech.cancel()
  }

  useEffect(() => () => clearDemoTimers(), [])

  function scheduleDemo(delayMs: number, task: () => void) {
    demoTimers.current.push(window.setTimeout(task, delayMs))
  }

  async function handleStart() {
    if (starting) return // ketukan ganda akan membuat sesi kedua yang terbuang
    setStarting(true)
    try {
      clearDemoTimers()
      setDemoMode(false)
      dispatch(complianceReset())
      dispatch(transcriptCleared())
      setReport(null)
      setReportError(null)
      // Daftar kewajiban WAJIB dimuat ulang. `complianceReset()` di atas
      // mengosongkannya, dan efek pemuatan awal hanya berjalan sekali saat
      // login — jadi tanpa ini seluruh sesi berjalan dengan checklist kosong,
      // dan kartu fokus menyatakan "Lengkap" padahal belum ada yang dinilai.
      const obligations = await container.repositories.session.obligations()
      dispatch(obligationsLoaded(obligations))

      const s = await container.usecases.startSession.execute('KREDIT-MULTIGUNA')
      dispatch(sessionStarted(s))
    } catch (e) {
      dispatch(sessionFailed((e as Error).message))
    } finally {
      setStarting(false)
    }
  }

  function handleDemo() {
    clearDemoTimers()
    dispatch(complianceReset())
    dispatch(transcriptCleared())
    setReport(null)
    setReportError(null)
    setDemoMode(true)

    const demoSession = createDemoSession()
    dispatch(obligationsLoaded(demoObligations))
    dispatch(sessionStarted(demoSession))
    dispatch(connectionChanged(true))
    dispatch(recordingChanged(true))

    scheduleDemo(600, () => {
      dispatch(utteranceAppended(demoUtterances[0]))
      dispatch(obligationSatisfied({ code: 'IDENTITY', confidence: 0.99, evidenceId: 'demo-1' }))
    })
    scheduleDemo(1_800, () => dispatch(utteranceAppended(demoUtterances[1])))
    scheduleDemo(3_000, () => {
      dispatch(utteranceAppended(demoUtterances[2]))
      dispatch(violationDetected(demoViolation))
      const warning = 'Hindari janji pasti disetujui. Koreksi dan jelaskan bunga serta cicilan.'
      dispatch(nudgeReceived(warning))
      container.repositories.speech.speak(warning)
    })
    scheduleDemo(4_800, () => {
      dispatch(utteranceAppended(demoUtterances[3]))
      dispatch(obligationSatisfied({ code: 'RATE', confidence: 0.96, evidenceId: 'demo-4' }))
      dispatch(obligationSatisfied({ code: 'TENOR', confidence: 0.95, evidenceId: 'demo-4' }))
    })
    scheduleDemo(6_200, () => {
      const reminder = 'Sampaikan denda keterlambatan dan hak nasabah untuk membatalkan.'
      dispatch(nudgeReceived(reminder))
      container.repositories.speech.speak(reminder)
    })
    // Petugas menyampaikan dua kewajiban terakhir, TETAPI diarization salah
    // melabelinya sebagai nasabah. Checklist sengaja tetap diam di sini —
    // hanya ucapan petugas yang boleh memenuhi kewajiban.
    scheduleDemo(7_600, () => dispatch(utteranceAppended(demoMislabeledUtterance)))

    // AssemblyAI mengoreksi labelnya. Ucapan itu belum pernah dinilai, jadi
    // dinilai sekarang — dan dua kewajiban terakhir baru berubah hijau.
    // Inilah bagian terdalam dari pipeline ini, dan tanpa babak ini penonton
    // tidak akan pernah tahu bahwa sistemnya menanganinya.
    scheduleDemo(8_900, () => {
      dispatch(speakerRevised({ id: 'demo-5', speaker: 'officer' }))
      dispatch(obligationSatisfied({ code: 'PENALTY', confidence: 0.94, evidenceId: 'demo-5' }))
      dispatch(obligationSatisfied({ code: 'RIGHT', confidence: 0.98, evidenceId: 'demo-5' }))
    })
    scheduleDemo(10_400, () => {
      const ended = createDemoSession('ended')
      dispatch(sessionEnded(ended))
      dispatch(connectionChanged(false))
      setReport(createDemoReport(ended))
    })
  }

  async function handleEnd() {
    if (!session) return
    if (demoMode) {
      clearDemoTimers()
      const ended = { ...session, status: 'ended' as const, score: 90 }
      dispatch(sessionEnded(ended))
      dispatch(connectionChanged(false))
      dispatch(recordingChanged(false))
      setReport(createDemoReport(ended))
      return
    }
    try {
      const ended = await container.usecases.endSession.execute(session.id)
      dispatch(sessionEnded(ended))
      setReport(await container.usecases.getReport.execute(session.id))
    } catch (e) {
      const message = (e as Error).message
      setReportError(message)
      dispatch(sessionFailed(message))
    }
  }

  function handleNewSession() {
    clearDemoTimers()
    dispatch(sessionReset())
    dispatch(complianceReset())
    dispatch(transcriptCleared())
    setReport(null)
    setReportError(null)
    setDemoMode(false)
  }



  return (
    <main className="page page--officer">
      <header className="page__head">
        <div>
          <h1>Bisik</h1>
          <p className="brandline">Kopilot kepatuhan petugas lapangan</p>
        </div>
        <ScoreBadge score={report?.session.score ?? score} />
      </header>

      {!session ? (
        <section className="start">
          {/* Login hanya menjaga SESI SUNGGUHAN. Demo terarah di bawah tetap
              terbuka untuk siapa pun — dia tidak menyentuh backend sama
              sekali, dan juri harus bisa mencobanya tanpa kredensial. */}
          {authRestoring ? (
            <p className="loading">
              <BisikWave />
              Memeriksa sesi login…
            </p>
          ) : !authUser ? (
            <LoginForm
              expects="officer"
              title="Masuk sebagai petugas"
              hint="Sesi dan laporannya akan tercatat atas nama akun ini."
            />
          ) : (
            <>
              <p className="whoami">
                <span>
                  Masuk sebagai <strong>{authUser.name}</strong>
                </span>
                <button
                  className="btn--link"
                  onClick={() => {
                    container.repositories.auth.logout()
                    dispatch(loggedOut())
                  }}
                >
                  Keluar
                </button>
              </p>
              <button className="btn btn--primary" onClick={handleStart} disabled={starting}>
                {starting ? (
                  <span className="btn__busy">
                    <BisikWave />
                    Membuka sesi…
                  </span>
                ) : (
                  'Mulai sesi'
                )}
              </button>
            </>
          )}

          {/* Kegagalan di sini dulu tidak pernah terlihat: error tersimpan di
              store tetapi layar mulai tidak pernah menampilkannya, jadi
              tombolnya tampak mati padahal gateway tidak bisa dihubungi. */}
          {sessionError && <p className="error-box">{sessionError}</p>}

          {gatewayDown && !sessionError && (
            <p className="warn-box">
              Gateway tidak bisa dihubungi, jadi sesi sungguhan belum bisa
              dimulai. Demo terarah di bawah tetap berjalan penuh — dia tidak
              memakai backend maupun API eksternal.
            </p>
          )}
          <div className="demo-entry">
            <span>atau</span>
            <button className="btn btn--secondary" onClick={handleDemo}>
              Putar demo terarah · 11 detik
            </button>
            <small>Simulasi lokal tanpa mikrofon atau API eksternal.</small>
          </div>
        </section>
      ) : session.status === 'ended' ? (
        <>
          {report ? (
            <ReportView report={report} simulated={demoMode} />
          ) : reportError ? (
            <p className="error-box">Laporan gagal dimuat: {reportError}</p>
          ) : (
            <p className="loading">
              <BisikWave />
              Menyiapkan laporan berbukti…
            </p>
          )}
          <button className="btn btn--primary btn--wide" onClick={handleNewSession}>
            Mulai sesi baru
          </button>
        </>
      ) : (
        <>
          {demoMode && <p className="demo-banner">Simulasi lokal · alur dipercepat</p>}
          <p className="status">
            <span className={`dot ${connected ? 'dot--on' : 'dot--off'}`} />
            {demoMode ? 'Demo aktif' : connected ? 'Terhubung' : 'Menyambung…'} ·{' '}
            {demoMode ? 'tanpa mikrofon' : recording ? 'merekam' : 'mic mati'}
            {!demoMode && stream.micProcessing && (
              <> · {stream.micProcessing}</>
            )}
          </p>

          {/* Kegagalan jalur audio harus terlihat: status "merekam" saja
              pernah menutupi sesi yang sebenarnya sudah mati. */}
          {sessionError && <p className="error-box">{sessionError}</p>}

          {gatewayDown && !sessionError && (
            <p className="warn-box">
              Gateway tidak bisa dihubungi, jadi sesi sungguhan belum bisa
              dimulai. Demo terarah di bawah tetap berjalan penuh — dia tidak
              memakai backend maupun API eksternal.
            </p>
          )}

          {/* Audio tidak layak: checklist sengaja ditahan agar tidak ada
              centang hijau palsu. Petugas harus tahu sebabnya dan bisa
              memperbaikinya saat itu juga. */}
          {!demoMode && stream.audioWarning && (
            <p className="warn-box">
              ⚠️ {stream.audioWarning} · penilaian ditahan sampai audio membaik
            </p>
          )}

          {!demoMode && stream.excludedCount > 0 && (
            <p className="warn-box warn-box--muted">
              {stream.excludedCount} ucapan tidak dihitung sebagai bukti — suara
              tidak dikenali atau audio tidak layak. Ulangi bagian itu.
            </p>
          )}

          {!demoMode && stream.calibrationStatus !== 'confirmed' ? (
            <SpeakerCalibration
              samples={stream.calibrationSamples}
              officerVoice={stream.officerVoice}
              duplicateVoice={stream.duplicateVoice}
              error={stream.calibrationError}
              onConfirm={stream.confirmSpeakerRoles}
              onRestart={stream.restartCalibration}
            />
          ) : (
            <>
              {/* Bisikan ditampilkan sekaligus diucapkan ke earpiece */}
              {nudge && <div className="nudge">🔈 {nudge}</div>}

              <ObligationFocus items={obligations} />

              {/* Rincian dan transkrip diturunkan ke balik disclosure: saat
                  sesi berjalan keduanya mengganggu, saat meninjau berguna.
                  Di mode demo transkrip dibuka supaya juri melihat buktinya. */}
              <details className="disclosure">
                <summary>Rincian kewajiban</summary>
                <ObligationList items={obligations} />
              </details>
              <details className="disclosure" open={demoMode}>
                <summary>Transkrip</summary>
                <TranscriptView utterances={utterances} partial={partial} />
              </details>
            </>
          )}

          <button className="btn btn--danger" onClick={handleEnd}>
            Akhiri sesi
          </button>
        </>
      )}
    </main>
  )
}
