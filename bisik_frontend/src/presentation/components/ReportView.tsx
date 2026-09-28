import type { ComplianceReport } from '../../domain/entities/compliance'
import type { Speaker } from '../../domain/entities/session'
import { ScoreBadge } from './ScoreBadge'
import { useI18n } from '../i18n/useI18n'
import { obligationLabel } from '../i18n/localize'

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
  const { t } = useI18n()
  const speakerLabel: Record<Speaker, string> = {
    officer: t.speakers.officer,
    customer: t.speakers.customer,
    unknown: t.speakers.unknownLong,
  }
  const evidence = new Map(report.transcript.map((utterance) => [utterance.id, utterance]))
  const satisfied = report.obligations.filter((item) => item.status === 'satisfied').length

  return (
    <article className="report">
      {simulated && (
        <p className="demo-banner">{t.report.simulated}</p>
      )}
      <header className="report__hero">
        <div>
          <p className="report__eyebrow">{t.report.eyebrow}</p>
          <h2>{t.report.session(report.session.id.slice(0, 8))}</h2>
          <p className="muted">
            {t.report.meta(report.session.officerId, report.session.productId)}
          </p>
        </div>
        <ScoreBadge score={report.session.score} />
      </header>

      <div className="report__stats" aria-label={t.report.summary}>
        <div><strong>{satisfied}/{report.obligations.length}</strong><span>{t.report.obligations}</span></div>
        <div><strong>{report.violations.length}</strong><span>{t.report.violations}</span></div>
        <div><strong>{report.transcript.length}</strong><span>{t.report.evidenceUtterances}</span></div>
      </div>

      <section className="report__section">
        <h2>{t.report.evidenceTitle}</h2>
        <ol className="evidence-list">
          {report.obligations.map((item) => {
            const utterance = item.evidenceId ? evidence.get(item.evidenceId) : undefined
            return (
              <li key={item.code} className={`evidence evidence--${item.status}`}>
                <div className="evidence__head">
                  <span>{item.status === 'satisfied' ? '✓' : '!'}</span>
                  <strong>{obligationLabel(t, item)}</strong>
                  <small>
                    {item.status === 'satisfied'
                      ? t.report.confidence(Math.round(item.confidence * 100))
                      : t.report.notSatisfied}
                  </small>
                </div>
                {utterance ? (
                  <blockquote>
                    “{utterance.text}”
                    <footer>{speakerLabel[utterance.speaker]} · {timestamp(utterance.startMs)}</footer>
                  </blockquote>
                ) : (
                  <p className="muted">{t.report.noQuote}</p>
                )}
              </li>
            )
          })}
        </ol>
      </section>

      <section className="report__section">
        <h2>{t.report.violations}</h2>
        {report.violations.length === 0 ? (
          <p className="report__clean">{t.report.clean}</p>
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
                  <span>{t.severity[violation.severity] ?? violation.severity}</span>
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <section className="report__section">
        <h2>{t.report.transcriptTitle}</h2>
        <div className="transcript transcript--report">
          {report.transcript.map((utterance) => (
            <p key={utterance.id} className={`line line--${utterance.speaker}`}>
              <time>{timestamp(utterance.startMs)}</time>
              <strong>{speakerLabel[utterance.speaker]}</strong>
              <span>{utterance.text}</span>
              {utterance.revised && <em className="line__revised">{t.transcript.revised}</em>}
            </p>
          ))}
        </div>
      </section>
    </article>
  )
}
