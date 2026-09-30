import { useState } from 'react'
import { container } from '../../infrastructure/di/container'
import { useAppDispatch, useAppSelector } from '../hooks/redux'
import { loggedIn, loginFailed } from '../../application/store/slices/authSlice'
import { selectAuthError } from '../../application/store/selectors'
import { BisikWave } from './BisikWave'
import type { Role } from '../../domain/entities/auth'
import { AppError } from '../../domain/entities/app_error'
import { useI18n } from '../i18n/useI18n'
import { localizeError } from '../i18n/localize'

interface Props {
  /** Peran yang dibutuhkan halaman ini; login dengan peran lain ditolak. */
  expects: Role
  title: string
  hint: string
}

export function LoginForm({ expects, title, hint }: Props) {
  const dispatch = useAppDispatch()
  const { t } = useI18n()
  const error = useAppSelector(selectAuthError)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  // Tersamar secara bawaan; petugas bisa mengintip untuk memastikan tidak
  // salah ketik — terutama di layar sentuh saat berdiri di depan nasabah.
  const [showPassword, setShowPassword] = useState(false)

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (busy) return
    setBusy(true)
    try {
      const session = await container.repositories.auth.login(username, password)
      // Akun supervisor tidak boleh menjalankan sesi, dan sebaliknya. Ini
      // hanya penjagaan UI — backend tetap menolak sendiri lewat peran token.
      if (session.user.role !== expects) {
        container.repositories.auth.logout()
        dispatch(loginFailed(AppError.roleMismatch(session.user.role, expects)))
        return
      }
      dispatch(loggedIn(session.user))
    } catch (e) {
      dispatch(loginFailed((e as Error).message))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form className="start start--login" onSubmit={submit}>
      <h2 className="login__title">{title}</h2>
      <p className="muted">{hint}</p>

      <div className="field">
        <label htmlFor="username">{t.auth.username}</label>
        <input
          id="username"
          autoComplete="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
      </div>

      <div className="field">
        <label htmlFor="password">{t.auth.password}</label>
        <div className="password">
          <input
            id="password"
            type={showPassword ? 'text' : 'password'}
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <button
            type="button"
            className="password__toggle"
            onClick={() => setShowPassword((v) => !v)}
            aria-label={showPassword ? t.auth.hidePassword : t.auth.showPassword}
            aria-pressed={showPassword}
            aria-controls="password"
            title={showPassword ? t.auth.hidePassword : t.auth.showPassword}
          >
            {showPassword ? <EyeOffIcon /> : <EyeIcon />}
          </button>
        </div>
      </div>

      <button className="btn btn--primary" type="submit" disabled={busy}>
        {busy ? (
          <span className="btn__busy">
            <BisikWave />
            {t.auth.checking}
          </span>
        ) : (
          t.auth.submit
        )}
      </button>

      {error && <p className="error-box">{localizeError(t, error)}</p>}
    </form>
  )
}

// Ikon garis sederhana, mengikuti warna teks tombol (currentColor).
function EyeIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  )
}

function EyeOffIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M10.6 5.1A9.8 9.8 0 0 1 12 5c6.5 0 10 7 10 7a17.6 17.6 0 0 1-2.6 3.6" />
      <path d="M6.6 6.6A17.4 17.4 0 0 0 2 12s3.5 7 10 7a9.6 9.6 0 0 0 5.4-1.6" />
      <path d="M9.9 9.9a3 3 0 0 0 4.2 4.2" />
      <path d="m2 2 20 20" />
    </svg>
  )
}
