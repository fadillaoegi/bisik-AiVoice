import { useI18n } from '../i18n/useI18n'

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
  const { t } = useI18n()
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
        {t.calibration.eyebrow}
        {step < 3 && t.calibration.step(step)}
      </p>
      <h2 id="calibration-title">{t.calibration.title}</h2>

      {/* LANGKAH 1 — petugas */}
      {step === 1 ? (
        <>
          <p className="calibration__prompt">
            <strong>{t.speakers.officer}</strong>, {t.calibration.officerPrompt}
          </p>
          <p className="calibration__script">{t.calibration.officerScript}</p>
          <p className="calibration__waiting">{t.calibration.listeningOfficer}</p>
        </>
      ) : (
        <div className="calibration__done">
          <span className="calibration__check">✓</span>
          <span>
            <strong>{t.calibration.officerRegistered}</strong> · {t.calibration.voice(officer!)}
            <small>{utterances.get(officer!)}</small>
          </span>
        </div>
      )}

      {/* LANGKAH 2 — nasabah */}
      {step === 2 && (
        <>
          <p className="calibration__prompt">
            {t.calibration.customerPromptLead} <strong>{t.calibration.customerWord}</strong>
            {t.calibration.customerPromptTail}
          </p>
          <p className="calibration__script">{t.calibration.customerScript}</p>
          {duplicateVoice ? (
            <p className="warn-box">
              {t.calibration.duplicateLead} <strong>{t.calibration.duplicateStrong}</strong>{' '}
              {t.calibration.duplicateTail}
            </p>
          ) : (
            <p className="calibration__waiting">{t.calibration.listeningCustomer}</p>
          )}
        </>
      )}

      {step === 3 && (
        <div className="calibration__done">
          <span className="calibration__check">✓</span>
          <span>
            <strong>{t.calibration.customerRegistered}</strong> · {t.calibration.voice(customer!)}
            <small>{utterances.get(customer!)}</small>
          </span>
        </div>
      )}

      {error && <p className="error-box">{t.calibration.failed(error)}</p>}

      <button
        className="btn btn--primary btn--wide"
        disabled={step !== 3}
        onClick={() => officer && customer && onConfirm(officer, customer)}
      >
        {step === 3 ? t.calibration.confirm : t.calibration.waiting}
      </button>

      {/* Jalan keluar kalau orang yang salah bicara duluan, atau kalau suara
          kedua tidak pernah terpisah. */}
      <button className="btn--link" onClick={onRestart}>
        {t.calibration.restart}
      </button>

      <small className="calibration__note">{t.calibration.note}</small>
    </section>
  )
}
