import type { Obligation } from '../../domain/entities/compliance'
import { useI18n } from '../i18n/useI18n'
import { obligationLabel } from '../i18n/localize'

// Kalimat perintah per butir ada di kamus (`obligations.hint`). Sengaja TIDAK
// memakai `Description` dari backend: yang di sana adalah prompt untuk
// semantic matcher — kalimat orang ketiga yang deskriptif. Petugas yang
// sedang bicara butuh instruksi langsung dan pendek.

/**
 * Satu butir besar + titik progres.
 *
 * Layar ini dilirik sambil petugas menatap nasabah, bukan dibaca. Karena itu
 * hanya kewajiban berikutnya yang tampil besar; sisanya cukup jadi titik.
 */
export function ObligationFocus({ items }: { items: Obligation[] }) {
  const { t } = useI18n()
  // Nol butir berarti daftarnya belum termuat, BUKAN semuanya terpenuhi.
  // Membedakan keduanya penting: "Lengkap" pada checklist kosong adalah
  // klaim palsu, dan itu justru kebalikan dari gunanya produk ini.
  if (items.length === 0) {
    return (
      <section className="focus">
        <p className="focus__eyebrow">{t.focus.loading}</p>
        <h2 className="focus__title">{t.focus.preparing}</h2>
      </section>
    )
  }

  const next = items.find((o) => o.status === 'pending')
  const satisfied = items.filter((o) => o.status === 'satisfied').length

  return (
    <>
      <section className={next ? 'focus' : 'focus focus--done'}>
        <p className="focus__eyebrow">
          {next ? t.focus.pending : t.focus.allDone}
        </p>
        <h2 className="focus__title">{next ? obligationLabel(t, next) : t.focus.complete}</h2>
        {next && <p className="focus__hint">{t.obligations.hint[next.code]}</p>}
      </section>

      <div className="dots">
        <span className="dots__row">
          {items.map((o) => (
            <span key={o.code} className={`dots__dot dots__dot--${o.status}`} />
          ))}
        </span>
        <span className="dots__count">
          {t.focus.count(satisfied, items.length)}
        </span>
      </div>
    </>
  )
}
