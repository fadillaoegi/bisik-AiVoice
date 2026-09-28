import { useCallback, useEffect, useState } from 'react'
import { container } from '../../../infrastructure/di/container'
import { useAppDispatch, useAppSelector } from '../../hooks/redux'
import { loggedOut } from '../../../application/store/slices/authSlice'
import {
  complianceReset,
  obligationsLoaded,
  violationsLoaded,
} from '../../../application/store/slices/complianceSlice'
import {
  transcriptCleared,
  transcriptLoaded,
} from '../../../application/store/slices/transcriptSlice'
import type { Session } from '../../../domain/entities/session'
import { useSessionStream } from '../../hooks/useSessionStream'
import {
  selectAuthRestoring,
  selectAuthUser,
  selectConnected,
  selectLiveScore,
  selectObligations,
  selectPartial,
  selectUtterances,
  selectViolations,
} from '../../../application/store/selectors'
import { ObligationList } from '../../components/ObligationList'
import { TranscriptView } from '../../components/TranscriptView'
import { ScoreBadge } from '../../components/ScoreBadge'
import { BisikWave } from '../../components/BisikWave'
import { LoginForm } from '../../components/LoginForm'
import { LanguageSwitch } from '../../components/LanguageSwitch'
import { AccountBar } from '../../components/AccountBar'
import { useI18n } from '../../i18n/useI18n'
import { localizeError } from '../../i18n/localize'

export function SupervisorPage() {
  const dispatch = useAppDispatch()
  const { t } = useI18n()
  const [watching, setWatching] = useState<string | null>(null)
  const [sessions, setSessions] = useState<Session[]>([])
  const [listError, setListError] = useState<string | null>(null)
  // Dimulai dari true: daftar memang langsung dimuat begitu supervisor masuk.
  const [loadingList, setLoadingList] = useState(true)
  const [opening, setOpening] = useState<string | null>(null)
  const [openError, setOpenError] = useState<string | null>(null)

  const authUser = useAppSelector(selectAuthUser)
  const authRestoring = useAppSelector(selectAuthRestoring)

  const connected = useAppSelector(selectConnected)
  const obligations = useAppSelector(selectObligations)
  const utterances = useAppSelector(selectUtterances)
  const partial = useAppSelector(selectPartial)
  const violations = useAppSelector(selectViolations)
  const score = useAppSelector(selectLiveScore)

  useSessionStream(watching, 'supervisor')

  // Sengaja tidak menyetel status memuat di awal: fungsi ini juga dipanggil
  // dari effect, dan menyetel state secara sinkron di sana memicu render
  // beruntun. Tombol segarkan yang menyalakan statusnya sendiri.
  const refresh = useCallback(async () => {
    try {
      const list = await container.repositories.session.sessions()
      setSessions(list)
      setListError(null)
    } catch (e) {
      setListError((e as Error).message)
    } finally {
      setLoadingList(false)
    }
  }, [])

  useEffect(() => {
    // Aturan set-state-in-effect keliru di sini: seluruh setState di dalam
    // refresh() berada SETELAH await, jadi tidak ada render beruntun.
    // Memuat daftar saat supervisor masuk justru persis kegunaan effect —
    // menyinkronkan dengan sistem luar.
    // oxlint-disable-next-line react/set-state-in-effect
    if (authUser) void refresh()
  }, [authUser, refresh])

  /**
   * Memuat snapshot sesi sebelum menyambungkan WebSocket.
   *
   * Supervisor dulu memulai dari layar kosong: store hanya terisi oleh event
   * yang datang SETELAH dia menyambung, sehingga kewajiban yang sudah
   * terpenuhi dan pelanggaran yang sudah terjadi tidak terlihat sama sekali.
   *
   * Urutannya penting. Snapshot diambil DULU, baru soket dibuka. Kalau
   * dibalik, event yang tiba selama pengambilan snapshot akan tertimpa data
   * lama yang datang belakangan.
   */
  async function open(sessionId: string) {
    setOpening(sessionId)
    setOpenError(null)
    dispatch(complianceReset())
    dispatch(transcriptCleared())
    try {
      const snapshot = await container.usecases.getReport.execute(sessionId)
      dispatch(obligationsLoaded(snapshot.obligations))
      dispatch(violationsLoaded(snapshot.violations))
      dispatch(transcriptLoaded(snapshot.transcript))
      setWatching(sessionId)
    } catch (e) {
      setOpenError((e as Error).message)
    } finally {
      setOpening(null)
    }
  }

  /**
   * Keluar sekaligus melupakan sesi yang sedang dipantau. Tanpa ini akun
   * berikutnya yang masuk di peramban yang sama langsung mendarat di sesi
   * pantauan orang sebelumnya.
   */
  function handleLogout() {
    setWatching(null)
    setSessions([])
    setLoadingList(true)
    dispatch(complianceReset())
    dispatch(transcriptCleared())
    container.repositories.auth.logout()
    dispatch(loggedOut())
  }

  return (
    <main className="page page--supervisor">
      <header className="page__head">
        <div>
          <h1>{t.brand.supervisorTitle}</h1>
          <p className="brandline">{t.brand.supervisorTagline}</p>
        </div>
        <div className="page__tools">
          <LanguageSwitch />
          <ScoreBadge score={score} />
        </div>
      </header>

      {authRestoring ? (
        <p className="loading">
          <BisikWave />
          {t.auth.checkingSession}
        </p>
      ) : !authUser ? (
        <LoginForm
          expects="supervisor"
          title={t.auth.supervisorTitle}
          hint={t.auth.supervisorHint}
        />
      ) : !watching ? (
        <section className="start">
          <AccountBar name={authUser.name} onLogout={handleLogout} />

          <h2>{t.supervisor.recent}</h2>
          {loadingList ? (
            <p className="loading">
              <BisikWave />
              {t.supervisor.loadingList}
            </p>
          ) : listError ? (
            <p className="error-box">{localizeError(t, listError)}</p>
          ) : sessions.length === 0 ? (
            <p className="muted">{t.supervisor.empty}</p>
          ) : (
            <ul className="session-list">
              {sessions.map((s) => (
                <li key={s.id}>
                  <button onClick={() => void open(s.id)} disabled={opening !== null}>
                    {opening === s.id ? (
                      <span className="btn__busy">
                        <BisikWave />
                        {t.supervisor.loadingSession}
                      </span>
                    ) : (
                      <>
                        {s.productId} · {s.status === 'active' ? t.supervisor.active : t.supervisor.ended}
                        <small>
                          {new Date(s.startedAt).toLocaleString(t.locale)} · {s.id}
                        </small>
                      </>
                    )}
                  </button>
                </li>
              ))}
            </ul>
          )}

          {openError && <p className="error-box">{localizeError(t, openError)}</p>}

          <button
            className="btn btn--secondary"
            onClick={() => {
              setLoadingList(true)
              void refresh()
            }}
          >
            {t.supervisor.refresh}
          </button>
        </section>
      ) : (
        <div className="grid">
          <div className="grid__wide">
            {/* Tanpa ini supervisor terjebak di satu sesi sampai halaman
                dimuat ulang — menyulitkan saat memantau beberapa petugas. */}
            <button
              className="btn--link"
              onClick={() => {
                setWatching(null)
                dispatch(complianceReset())
                dispatch(transcriptCleared())
              }}
            >
              {t.supervisor.back}
            </button>
          </div>

          <section>
            <h2>{t.supervisor.obligations}</h2>
            <ObligationList items={obligations} />
            <p className="status">
              <span className={`dot ${connected ? 'dot--on' : 'dot--off'}`} />
              {connected ? t.supervisor.live : t.supervisor.disconnected}
            </p>
          </section>

          <section>
            <h2>{t.supervisor.violations}</h2>
            {violations.length === 0 ? (
              <p className="muted">{t.supervisor.none}</p>
            ) : (
              <ul className="violations">
                {violations.map((v, i) => (
                  <li key={`${v.phrase}-${i}`} className="violation">
                    <strong>{v.phrase}</strong> <span>{t.severity[v.severity] ?? v.severity}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="grid__wide">
            <h2>{t.supervisor.transcript}</h2>
            <TranscriptView utterances={utterances} partial={partial} />
          </section>
        </div>
      )}
    </main>
  )
}
