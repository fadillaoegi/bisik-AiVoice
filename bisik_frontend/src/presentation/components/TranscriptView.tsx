import type { Speaker, Utterance } from '../../domain/entities/session'

const speakerLabel: Record<Speaker, string> = {
  officer: 'Petugas',
  customer: 'Nasabah',
  unknown: '—',
}

interface Props {
  utterances: Utterance[]
  partial: { speaker: Speaker; text: string } | null
}

export function TranscriptView({ utterances, partial }: Props) {
  return (
    <div className="transcript">
      {utterances.map((u) => (
        <p key={u.id} className={`line line--${u.speaker}`}>
          <strong>{speakerLabel[u.speaker]}</strong>
          {/* Label mentah diarization. Kecil dan redup: alat verifikasi,
              bukan informasi yang dibutuhkan petugas saat bicara. */}
          {u.sourceSpeaker && <span className="line__source">{u.sourceSpeaker}</span>}{' '}
          {u.text}
          {/* Label direvisi diarization — tonjolkan, jangan disembunyikan */}
          {u.revised && <em className="line__revised">label dikoreksi</em>}
        </p>
      ))}
      {partial && (
        <p className="line line--partial">
          <strong>{speakerLabel[partial.speaker]}</strong> {partial.text}
        </p>
      )}
    </div>
  )
}
