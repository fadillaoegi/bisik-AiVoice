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
import { LanguageSwitch } from '../../components/LanguageSwitch'
import { AccountBar } from '../../components/AccountBar'
import { useI18n } from '../../i18n/useI18n'
import { localizeAudioReason, localizeError, nudgeText } from '../../i18n/localize'
import type { Nudge } from '../../../domain/entities/compliance'

export function OfficerPage() {
  const dispatch = useAppDispatch()
  const { t } = useI18n()
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
  // Mengakhiri sesi menunggu AssemblyAI menuntaskan ucapan terakhir dan
  // backend menghitung skor akhir — bisa beberapa detik. Tanpa penanda,
  // tombolnya terasa mati dan petugas cenderung mengetuk berulang kali.
  const [ending, setEnding] = useState(false)
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

  function speakDemoNudge(nudge: Nudge) {
    dispatch(nudgeReceived(nudge))
    container.repositories.speech.speak(nudge.text, t.locale)
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
      // Demo ditonton juri, jadi bisikannya MENGIKUTI bahasa antarmuka —
      // beda dengan sesi sungguhan yang selalu berbisik dalam Bahasa
      // Indonesia agar echo guard backend tetap bekerja.
      speakDemoNudge({ text: t.demo.correction, kind: 'demo_correction' })
    })
    scheduleDemo(4_800, () => {
      dispatch(utteranceAppended(demoUtterances[3]))
      dispatch(obligationSatisfied({ code: 'RATE', confidence: 0.96, evidenceId: 'demo-4' }))
      dispatch(obligationSatisfied({ code: 'TENOR', confidence: 0.95, evidenceId: 'demo-4' }))
    })
    scheduleDemo(6_200, () => {
      speakDemoNudge({ text: t.demo.reminder, kind: 'demo_reminder' })
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
    if (!session || ending) return
    if (demoMode) {
      clearDemoTimers()
      const ended = { ...session, status: 'ended' as const, score: 90 }
      dispatch(sessionEnded(ended))
      dispatch(connectionChanged(false))
      dispatch(recordingChanged(false))
      setReport(createDemoReport(ended))
      return
    }
    setEnding(true)
    try {
      const ended = await container.usecases.endSession.execute(session.id)
      dispatch(sessionEnded(ended))
      setReport(await container.usecases.getReport.execute(session.id))
    } catch (e) {
      const message = (e as Error).message
      setReportError(message)
      dispatch(sessionFailed(message))
    } finally {
      setEnding(false)
    }
  }

  /**
   * Keluar sekaligus membuang sisa sesi di layar. Tanpa ini laporan petugas
   * sebelumnya masih terpampang untuk siapa pun yang masuk berikutnya di
   * perangkat yang sama.
   */
  function handleLogout() {
    handleNewSession()
    container.repositories.auth.logout()
    dispatch(loggedOut())
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
          <p className="brandline">{t.brand.tagline}</p>
        </div>
        <div className="page__tools">
          <LanguageSwitch />
          <ScoreBadge score={report?.session.score ?? score} />
        </div>
      </header>

      {!session ? (
        <section className="start">
          {/* Login hanya menjaga SESI SUNGGUHAN. Demo terarah di bawah tetap
              terbuka untuk siapa pun — dia tidak menyentuh backend sama
              sekali, dan juri harus bisa mencobanya tanpa kredensial. */}
          {authRestoring ? (
            <p className="loading">
              <BisikWave />
              {t.auth.checkingSession}
            </p>
          ) : !authUser ? (
            <LoginForm
              expects="officer"
              title={t.auth.officerTitle}
              hint={t.auth.officerHint}
            />
          ) : (
            <>
              <AccountBar name={authUser.name} onLogout={handleLogout} />
              <button
                className="btn btn--primary"
                onClick={handleStart}
                disabled={starting}
                aria-busy={starting}
              >
                {starting ? (
                  <span className="btn__busy">
                    <BisikWave />
                    {t.officer.starting}
                  </span>
                ) : (
                  t.officer.start
                )}
              </button>
            </>
          )}

          {/* Kegagalan di sini dulu tidak pernah terlihat: error tersimpan di
              store tetapi layar mulai tidak pernah menampilkannya, jadi
              tombolnya tampak mati padahal gateway tidak bisa dihubungi. */}
          {sessionError && <p className="error-box">{localizeError(t, sessionError)}</p>}

          {gatewayDown && !sessionError && (
            <p className="warn-box">{t.officer.gatewayDown}</p>
          )}
          <div className="demo-entry">
            <span>{t.officer.or}</span>
            <button className="btn btn--secondary" onClick={handleDemo}>
              {t.officer.demo}
            </button>
            <small>{t.officer.demoNote}</small>
          </div>
        </section>
      ) : session.status === 'ended' ? (
        <>
          {authUser && !demoMode && (
            <AccountBar name={authUser.name} onLogout={handleLogout} />
          )}
          {report ? (
            <ReportView report={report} simulated={demoMode} />
          ) : reportError ? (
            <p className="error-box">{t.officer.reportFailed(localizeError(t, reportError))}</p>
          ) : (
            <p className="loading">
              <BisikWave />
              {t.officer.preparingReport}
            </p>
          )}
          <button className="btn btn--primary btn--wide" onClick={handleNewSession}>
            {t.officer.newSession}
          </button>
        </>
      ) : (
        <>
          {demoMode && <p className="demo-banner">{t.officer.demoBanner}</p>}
          <p className="status">
            <span className={`dot ${connected ? 'dot--on' : 'dot--off'}`} />
            {demoMode ? t.officer.demoActive : connected ? t.officer.connected : t.officer.connecting} ·{' '}
            {demoMode ? t.officer.noMic : recording ? t.officer.recording : t.officer.micOff}
            {!demoMode && stream.micProcessing.length > 0 && (
              <> · {stream.micProcessing.map((key) => t.officer.micProcessing[key as keyof typeof t.officer.micProcessing]).join(' · ')}</>
            )}
          </p>

          {/* Kegagalan jalur audio harus terlihat: status "merekam" saja
              pernah menutupi sesi yang sebenarnya sudah mati. */}
          {sessionError && <p className="error-box">{localizeError(t, sessionError)}</p>}

          {gatewayDown && !sessionError && (
            <p className="warn-box">{t.officer.gatewayDown}</p>
          )}

          {/* Penilai mati: tanpa ini gejalanya hanya checklist yang tak pernah
              hijau dan pengingat tanpa henti — persis yang terjadi saat akun
              kehilangan akses LLM Gateway pada 28 Sep 2026. */}
          {!demoMode && stream.scoringError && (
            <p className="error-box">
              {t.officer.scoringUnavailable}
              <small className="error-box__detail">{stream.scoringError}</small>
            </p>
          )}

          {/* Audio tidak layak: checklist sengaja ditahan agar tidak ada
              centang hijau palsu. Petugas harus tahu sebabnya dan bisa
              memperbaikinya saat itu juga. */}
          {!demoMode && stream.audioWarning && (
            <p className="warn-box">
              ⚠️ {localizeAudioReason(t, stream.audioWarning)} · {t.officer.audioHeld}
            </p>
          )}

          {!demoMode && stream.excludedCount > 0 && (
            <p className="warn-box warn-box--muted">{t.officer.excluded(stream.excludedCount)}</p>
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
              {nudge && <div className="nudge">🔈 {nudgeText(t, nudge)}</div>}

              <ObligationFocus items={obligations} />

              {/* Rincian dan transkrip diturunkan ke balik disclosure: saat
                  sesi berjalan keduanya mengganggu, saat meninjau berguna.
                  Di mode demo transkrip dibuka supaya juri melihat buktinya. */}
              <details className="disclosure">
                <summary>{t.officer.obligationDetails}</summary>
                <ObligationList items={obligations} />
              </details>
              <details className="disclosure" open={demoMode}>
                <summary>{t.officer.transcript}</summary>
                <TranscriptView utterances={utterances} partial={partial} />
              </details>
            </>
          )}

          <button
            className="btn btn--danger"
            onClick={handleEnd}
            disabled={ending}
            aria-busy={ending}
          >
            {ending ? (
              <span className="btn__busy">
                <BisikWave />
                {t.officer.ending}
              </span>
            ) : (
              t.officer.end
            )}
          </button>
        </>
      )}
    </main>
  )
}
