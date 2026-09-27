export function ScoreBadge({ score }: { score: number }) {
  const tone = score >= 80 ? 'good' : score >= 50 ? 'warn' : 'bad'
  return (
    <div className={`score score--${tone}`}>
      <span className="score__value">{score}</span>
      <span className="score__unit">/100</span>
    </div>
  )
}
