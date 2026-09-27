import { useState } from 'react'
import { container } from '../../infrastructure/di/container'
import { useAppDispatch, useAppSelector } from '../hooks/redux'
import { loggedIn, loginFailed } from '../../application/store/slices/authSlice'
import { selectAuthError } from '../../application/store/selectors'
import { BisikWave } from './BisikWave'
import type { Role } from '../../domain/entities/auth'

interface Props {
  /** Peran yang dibutuhkan halaman ini; login dengan peran lain ditolak. */
  expects: Role
  title: string
  hint: string
}

const roleLabel: Record<Role, string> = {
  officer: 'petugas',
  supervisor: 'supervisor',
}

export function LoginForm({ expects, title, hint }: Props) {
  const dispatch = useAppDispatch()
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
        dispatch(
          loginFailed(
            `Akun ini adalah ${roleLabel[session.user.role]}. Halaman ini untuk ${roleLabel[expects]}.`,
          ),
        )
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
        <label htmlFor="username">Nama pengguna</label>
        <input
          id="username"
          autoComplete="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
      </div>

      <div className="field">
        <label htmlFor="password">Kata sandi</label>
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
            Memeriksa…
          </span>
        ) : (
          'Masuk'
        )}
      </button>

      {error && <p className="error-box">{error}</p>}
    </form>
  )
}
