import type { Obligation } from '../../domain/entities/compliance'

/**
 * Kalimat perintah per butir kewajiban.
 *
 * Sengaja TIDAK memakai `Description` dari backend: yang di sana adalah
 * prompt untuk semantic matcher — kalimat orang ketiga yang deskriptif.
 * Petugas yang sedang bicara butuh instruksi langsung dan pendek.
 */
const hint: Record<string, string> = {
  IDENTITY: 'Sebutkan nama kamu dan nama lembaga tempatmu bekerja.',
  RATE: 'Sebutkan suku bunga atau total biaya yang harus dibayar.',
  TENOR: 'Sebutkan jangka waktu dan besar cicilan per bulan.',
  PENALTY: 'Jelaskan denda kalau nasabah telat membayar.',
  RIGHT: 'Beri tahu nasabah berhak menolak atau membatalkan.',
}

/**
 * Satu butir besar + titik progres.
 *
 * Layar ini dilirik sambil petugas menatap nasabah, bukan dibaca. Karena itu
 * hanya kewajiban berikutnya yang tampil besar; sisanya cukup jadi titik.
 */
export function ObligationFocus({ items }: { items: Obligation[] }) {
  // Nol butir berarti daftarnya belum termuat, BUKAN semuanya terpenuhi.
  // Membedakan keduanya penting: "Lengkap" pada checklist kosong adalah
  // klaim palsu, dan itu justru kebalikan dari gunanya produk ini.
  if (items.length === 0) {
    return (
      <section className="focus">
        <p className="focus__eyebrow">Memuat kewajiban</p>
        <h2 className="focus__title">Menyiapkan…</h2>
      </section>
    )
  }

  const next = items.find((o) => o.status === 'pending')
  const satisfied = items.filter((o) => o.status === 'satisfied').length

  return (
    <>
      <section className={next ? 'focus' : 'focus focus--done'}>
        <p className="focus__eyebrow">
          {next ? 'Belum disampaikan' : 'Semua kewajiban terpenuhi'}
        </p>
        <h2 className="focus__title">{next ? next.label : 'Lengkap'}</h2>
        {next && <p className="focus__hint">{hint[next.code]}</p>}
      </section>

      <div className="dots">
        <span className="dots__row">
          {items.map((o) => (
            <span key={o.code} className={`dots__dot dots__dot--${o.status}`} />
          ))}
        </span>
        <span className="dots__count">
          {satisfied} dari {items.length}
        </span>
      </div>
    </>
  )
}
