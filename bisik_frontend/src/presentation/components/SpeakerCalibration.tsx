interface CalibrationSample {
  id: string
  sourceSpeaker: string
  text: string
}

interface Props {
  samples: CalibrationSample[]
  /** Suara yang sudah terdaftar sebagai petugas; null = langkah 1 belum selesai. */
  officerVoice: string | null
  /** Suara yang sama terdengar lagi saat menunggu orang kedua. */
  duplicateVoice: boolean
  error: string | null
  onConfirm: (officerLabel: string, customerLabel: string) => void
  onRestart: () => void
}

/** Ucapan terakhir per suara — satu orang bisa bicara berkali-kali. */
function lastUtteranceByVoice(samples: CalibrationSample[]) {
  const byLabel = new Map<string, string>()
  for (const sample of samples) {
    if (!sample.sourceSpeaker) continue
    if (sample.text) byLabel.set(sample.sourceSpeaker, sample.text)
  }
  return byLabel
}

export function SpeakerCalibration({
  samples,
  officerVoice,
  duplicateVoice,
  error,
  onConfirm,
  onRestart,
}: Props) {
  const utterances = lastUtteranceByVoice(samples)
  const voices = [...utterances.keys()].sort()

  // Kalau label petugas hilang karena diarization merevisinya, jangan
  // berpegang pada label yang sudah tidak ada — mundur ke langkah satu.
  const officer = officerVoice && voices.includes(officerVoice) ? officerVoice : null
  const customer = officer ? (voices.find((v) => v !== officer) ?? null) : null

  const step = officer === null ? 1 : customer === null ? 2 : 3

  return (
    <section className="calibration" aria-labelledby="calibration-title">
      <p className="calibration__eyebrow">
        Langkah keamanan · belum dinilai{step < 3 && ` · langkah ${step} dari 2`}
      </p>
      <h2 id="calibration-title">Kenali dua suara</h2>

      {/* LANGKAH 1 — petugas */}
      {step === 1 ? (
        <>
          <p className="calibration__prompt">
            <strong>Petugas</strong>, ucapkan kalimat ini dengan jelas:
          </p>
          <p className="calibration__script">“Saya petugas yang menjalankan sesi ini.”</p>
          <p className="calibration__waiting">Mendengarkan suara petugas…</p>
        </>
      ) : (
        <div className="calibration__done">
          <span className="calibration__check">✓</span>
          <span>
            <strong>Petugas terdaftar</strong> · Suara {officer}
            <small>{utterances.get(officer!)}</small>
          </span>
        </div>
      )}

      {/* LANGKAH 2 — nasabah */}
      {step === 2 && (
        <>
          <p className="calibration__prompt">
            Sekarang giliran <strong>nasabah</strong>. Ucapkan:
          </p>
          <p className="calibration__script">“Saya nasabah dan siap memulai.”</p>
          {duplicateVoice ? (
            <p className="warn-box">
              Suara itu sudah terdaftar sebagai petugas. Minta <strong>orang
              kedua</strong> yang berbicara — sistem perlu mendengar suara yang
              berbeda untuk bisa membedakan keduanya.
            </p>
          ) : (
            <p className="calibration__waiting">Mendengarkan suara nasabah…</p>
          )}
        </>
      )}

      {step === 3 && (
        <div className="calibration__done">
          <span className="calibration__check">✓</span>
          <span>
            <strong>Nasabah terdaftar</strong> · Suara {customer}
            <small>{utterances.get(customer!)}</small>
          </span>
        </div>
      )}

      {error && <p className="error-box">Kalibrasi gagal: {error}</p>}

      <button
        className="btn btn--primary btn--wide"
        disabled={step !== 3}
        onClick={() => officer && customer && onConfirm(officer, customer)}
      >
        {step === 3 ? 'Konfirmasi dan mulai penilaian' : 'Menunggu dua suara'}
      </button>

      {/* Jalan keluar kalau orang yang salah bicara duluan, atau kalau suara
          kedua tidak pernah terpisah. */}
      <button className="btn--link" onClick={onRestart}>
        Ulangi kalibrasi dari awal
      </button>

      <small className="calibration__note">
        Ucapan kalibrasi tidak masuk laporan dan tidak dinilai.
      </small>
    </section>
  )
}
