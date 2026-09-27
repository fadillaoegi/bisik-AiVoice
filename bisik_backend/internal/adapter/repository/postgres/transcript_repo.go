package postgres

import (
	"errors"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TranscriptRepo struct{ pool *pgxpool.Pool }

func NewTranscriptRepo(p *pgxpool.Pool) *TranscriptRepo { return &TranscriptRepo{pool: p} }

func (r *TranscriptRepo) Append(ctx domain.Context, u *domain.Utterance) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO utterances (id, session_id, speaker, source_speaker, text, start_ms, end_ms, revised, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET
			text=EXCLUDED.text, end_ms=EXCLUDED.end_ms, source_speaker=EXCLUDED.source_speaker`,
		u.ID, u.SessionID, u.Speaker, u.SourceSpeaker, u.Text, u.StartMS, u.EndMS, u.Revised, u.CreatedAt)
	return err
}

func (r *TranscriptRepo) FindByID(ctx domain.Context, id string) (*domain.Utterance, error) {
	var u domain.Utterance
	err := r.pool.QueryRow(ctx, `
		SELECT id, session_id, speaker, source_speaker, text, start_ms, end_ms, revised, created_at
		FROM utterances WHERE id=$1`, id).
		Scan(&u.ID, &u.SessionID, &u.Speaker, &u.SourceSpeaker, &u.Text, &u.StartMS, &u.EndMS, &u.Revised, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *TranscriptRepo) ListBySession(ctx domain.Context, sessionID string) ([]*domain.Utterance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, session_id, speaker, source_speaker, text, start_ms, end_ms, revised, created_at
		FROM utterances WHERE session_id=$1 ORDER BY start_ms ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Utterance
	for rows.Next() {
		var u domain.Utterance
		if err := rows.Scan(&u.ID, &u.SessionID, &u.Speaker, &u.SourceSpeaker, &u.Text, &u.StartMS, &u.EndMS, &u.Revised, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

func (r *TranscriptRepo) ReviseSpeaker(ctx domain.Context, utteranceID string, speaker domain.Speaker) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE utterances SET speaker=$2, revised=TRUE WHERE id=$1`, utteranceID, speaker)
	return err
}
