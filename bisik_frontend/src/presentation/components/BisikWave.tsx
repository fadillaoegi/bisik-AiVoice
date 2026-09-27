/**
 * Indikator memuat berbentuk gelombang suara lima batang.
 *
 * Bentuknya diambil dari ikon Bisik. Kebetulan yang berguna: lima batang juga
 * jumlah butir kewajiban, jadi indikatornya terbaca sebagai "sedang menyimak",
 * bukan spinner generik. Warnanya mengikuti `currentColor` supaya sama-sama
 * benar di atas tombol mint maupun di atas latar gelap.
 */
export function BisikWave() {
  return (
    <span className="wave" aria-hidden="true">
      <span />
      <span />
      <span />
      <span />
      <span />
    </span>
  )
}
