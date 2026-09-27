import type { AuthSession, AuthUser } from '../entities/auth'

/** Kontrak autentikasi. Implementasinya ada di infrastructure. */
export interface AuthRepository {
  login(username: string, password: string): Promise<AuthSession>
  /** Memulihkan sesi login dari token tersimpan; null kalau sudah tidak sah. */
  restore(): Promise<AuthUser | null>
  logout(): void
  token(): string | null
}
