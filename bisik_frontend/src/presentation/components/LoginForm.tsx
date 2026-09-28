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
        <input
          id="password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
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
