/** Peran yang memakai aplikasi. Nasabah sengaja tidak ada di sini. */
export type Role = 'officer' | 'supervisor'

export interface AuthUser {
  id: string
  name: string
  role: Role
}

export interface AuthSession {
  token: string
  user: AuthUser
}
