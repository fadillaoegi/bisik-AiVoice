import type { AuthRepository } from '../../domain/repositories/auth_repository'
import type { AuthSession, AuthUser, Role } from '../../domain/entities/auth'
import { tokenStore } from './token_store'

const API = import.meta.env.VITE_API_URL || window.location.origin

interface UserDTO {
  id: string
  username?: string
  name: string
  role: Role
}

export class HttpAuthRepository implements AuthRepository {
  async login(username: string, password: string): Promise<AuthSession> {
    const res = await fetch(`${API}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })
    if (!res.ok) {
      // Backend sengaja tidak membedakan akun tidak ada dan sandi salah.
      throw new Error(
        res.status === 401
          ? 'Nama pengguna atau kata sandi salah'
          : `Gagal masuk (HTTP ${res.status})`,
      )
    }
    const raw = (await res.json()) as { token: string; user: UserDTO }
    tokenStore.write(raw.token)
    return {
      token: raw.token,
      user: { id: raw.user.id, name: raw.user.name, role: raw.user.role },
    }
  }

  /**
   * Token disimpan di perangkat dan bisa saja sudah kedaluwarsa atau
   * ditandatangani rahasia lama. Server yang memutuskan, bukan klien.
   */
  async restore(): Promise<AuthUser | null> {
    const token = tokenStore.read()
    if (!token) return null
    try {
      const res = await fetch(`${API}/api/auth/me`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) {
        tokenStore.clear()
        return null
      }
      const raw = (await res.json()) as UserDTO
      return { id: raw.id, name: raw.name, role: raw.role }
    } catch {
      // Gateway tidak bisa dihubungi. Token belum tentu tidak sah, jadi
      // jangan dibuang — cukup anggap belum masuk untuk saat ini.
      return null
    }
  }

  logout(): void {
    tokenStore.clear()
  }

  token(): string | null {
    return tokenStore.read()
  }
}
