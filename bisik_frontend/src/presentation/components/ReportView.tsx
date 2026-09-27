import type { ComplianceReport } from '../../domain/entities/compliance'
import type { Speaker } from '../../domain/entities/session'
import { ScoreBadge } from './ScoreBadge'

const speakerLabel: Record<Speaker, string> = {
  officer: 'Petugas',
  customer: 'Nasabah',
  unknown: 'Tidak diketahui',
}

function timestamp(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

interface Props {
  report: ComplianceReport
  simulated?: boolean
}

export function ReportView({ report, simulated = false }: Props) {
  const evidence = new Map(report.transcript.map((utterance) => [utterance.id, utterance]))
  const satisfied = report.obligations.filter((item) => item.status === 'satisfied').length

  return (
    <article className="report">
      {simulated && (
        <p className="demo-banner">
          Simulasi lokal · bukan transkripsi AssemblyAI live
        </p>
      )}
      <header className="report__hero">
        <div>
          <p className="report__eyebrow">Laporan kepatuhan</p>
          <h2>Sesi {report.session.id.slice(0, 8)}</h2>
          <p className="muted">
            Petugas {report.session.officerId} · Produk {report.session.productId}
          </p>
        </div>
        <ScoreBadge score={report.session.score} />
      </header>

      <div className="report__stats" aria-label="Ringkasan laporan">
        <div><strong>{satisfied}/{report.obligations.length}</strong><span>Kewajiban</span></div>
        <div><strong>{report.violations.length}</strong><span>Pelanggaran</span></div>
        <div><strong>{report.transcript.length}</strong><span>Ucapan berbukti</span></div>
      </div>

      <section className="report__section">
        <h2>Bukti per kewajiban</h2>
        <ol className="evidence-list">
          {report.obligations.map((item) => {
            const utterance = item.evidenceId ? evidence.get(item.evidenceId) : undefined
            return (
              <li key={item.code} className={`evidence evidence--${item.status}`}>
                <div className="evidence__head">
                  <span>{item.status === 'satisfied' ? '✓' : '!'}</span>
                  <strong>{item.label}</strong>
                  <small>
                    {item.status === 'satisfied'
                      ? `${Math.round(item.confidence * 100)}% confidence`
                      : 'Belum terpenuhi'}
                  </small>
                </div>
                {utterance ? (
                  <blockquote>
                    “{utterance.text}”
                    <footer>{speakerLabel[utterance.speaker]} · {timestamp(utterance.startMs)}</footer>
                  </blockquote>
                ) : (
                  <p className="muted">Tidak ada kutipan yang memenuhi evidence gate.</p>
                )}
              </li>
            )
          })}
        </ol>
      </section>

      <section className="report__section">
        <h2>Pelanggaran</h2>
        {report.violations.length === 0 ? (
          <p className="report__clean">✓ Tidak ada janji terlarang yang terdeteksi.</p>
        ) : (
          <ul className="violations">
            {report.violations.map((violation, index) => {
              const utterance = evidence.get(violation.evidenceId)
              return (
                <li key={`${violation.evidenceId}-${index}`} className="violation violation--report">
                  <div>
                    <strong>{violation.phrase}</strong>
                    {utterance && <p>“{utterance.text}”</p>}
                  </div>
                  <span>{violation.severity}</span>
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <section className="report__section">
        <h2>Transkrip berbukti</h2>
        <div className="transcript transcript--report">
          {report.transcript.map((utterance) => (
            <p key={utterance.id} className={`line line--${utterance.speaker}`}>
              <time>{timestamp(utterance.startMs)}</time>
              <strong>{speakerLabel[utterance.speaker]}</strong>
              <span>{utterance.text}</span>
              {utterance.revised && <em className="line__revised">label direvisi</em>}
            </p>
          ))}
        </div>
      </section>
    </article>
  )
}
