const KEY = 'bisik.token'

/**
 * Penyimpanan token di perangkat.
 *
 * localStorage dipilih dengan sadar: token ini berumur pendek dan seluruh
 * aplikasi berjalan di satu origin. Untuk produksi, cookie `HttpOnly` lebih
 * aman karena tidak bisa dibaca skrip — tetapi itu menuntut backend dan
 * frontend berbagi domain beserta perlindungan CSRF.
 *
 * Setiap akses dibungkus try/catch: di mode penyamaran atau saat penyimpanan
 * situs diblokir, mengaksesnya bisa melempar, bukan sekadar mengembalikan null.
 */
export const tokenStore = {
  read(): string | null {
    try {
      return window.localStorage.getItem(KEY)
    } catch {
      return null
    }
  },
  write(token: string): void {
    try {
      window.localStorage.setItem(KEY, token)
    } catch {
      // Token tetap hidup di memori selama tab terbuka.
    }
  },
  clear(): void {
    try {
      window.localStorage.removeItem(KEY)
    } catch {
      // Tidak ada yang bisa dilakukan; anggap sudah bersih.
    }
  },
}
