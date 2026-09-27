import type { Obligation } from '../../domain/entities/compliance'

const icon: Record<Obligation['status'], string> = {
  pending: '○',
  satisfied: '●',
  violated: '✕',
}

export function ObligationList({ items }: { items: Obligation[] }) {
  return (
    <ul className="obligations">
      {items.map((o) => (
        <li key={o.code} className={`obligation obligation--${o.status}`}>
          <span className="obligation__icon">{icon[o.status]}</span>
          <span className="obligation__label">{o.label}</span>
          {o.status === 'satisfied' && (
            <span className="obligation__conf">{Math.round(o.confidence * 100)}%</span>
          )}
        </li>
      ))}
    </ul>
  )
}
