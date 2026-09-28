import type { Role } from './auth'

/**
 * Kesalahan yang dibuat klien sendiri, dalam bentuk kode.
 *
 * Disimpan di store sebagai string `err:kode[:arg...]` dan baru diterjemahkan
 * saat ditampilkan — jadi mengganti bahasa ikut mengganti pesan yang sedang
 * tampil. Pesan dari server tidak berkode dan tampil apa adanya.
 */
export const AppError = {
  invalidCredentials: 'err:invalid-credentials',
  loginFailed: (status: number) => `err:login-failed:${status}`,
  roleMismatch: (actual: Role, expected: Role) => `err:role-mismatch:${actual}:${expected}`,
  gatewayLost: 'err:gateway-lost',
} as const
