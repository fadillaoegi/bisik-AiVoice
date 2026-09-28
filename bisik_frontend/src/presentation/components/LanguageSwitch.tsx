import { useI18n } from '../i18n/useI18n'
import type { Lang } from '../i18n/dictionary'

const LANGS: Lang[] = ['id', 'en']

/** Pengganti bahasa antarmuka. Dua pilihan cukup sebagai tombol berdampingan. */
export function LanguageSwitch() {
  const { lang, t, setLang } = useI18n()
  return (
    <div className="lang" role="group" aria-label={t.language.label}>
      {LANGS.map((option) => (
        <button
          key={option}
          type="button"
          className={option === lang ? 'lang__opt lang__opt--on' : 'lang__opt'}
          aria-pressed={option === lang}
          title={t.language.name[option]}
          lang={option}
          onClick={() => setLang(option)}
        >
          {t.language.short[option]}
        </button>
      ))}
    </div>
  )
}
