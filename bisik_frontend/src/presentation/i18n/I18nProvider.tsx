import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { I18nContext } from './context'
import { DEFAULT_LANG, dictionaries, type Lang } from './dictionary'

const STORAGE_KEY = 'bisik.lang'

/**
 * Pilihan bahasa hanya kenyamanan per perangkat, jadi cukup di localStorage.
 * Penyimpanan bisa dilarang (mode privat, data situs diblokir); dalam kasus
 * itu aplikasinya tetap jalan dengan Bahasa Indonesia.
 */
function readStored(): Lang {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY)
    return stored === 'en' || stored === 'id' ? stored : DEFAULT_LANG
  } catch {
    return DEFAULT_LANG
  }
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLang] = useState<Lang>(readStored)

  useEffect(() => {
    // Pembaca layar dan terjemahan otomatis peramban memakai atribut ini.
    document.documentElement.lang = lang
    try {
      window.localStorage.setItem(STORAGE_KEY, lang)
    } catch {
      // Tidak tersimpan berarti kembali ke bawaan saat dibuka ulang — wajar.
    }
  }, [lang])

  const value = useMemo(() => ({ lang, t: dictionaries[lang], setLang }), [lang])
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}
