package postgres

import (
	"errors"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct{ pool *pgxpool.Pool }

func NewSessionRepo(p *pgxpool.Pool) *SessionRepo { return &SessionRepo{pool: p} }

func (r *SessionRepo) Create(ctx domain.Context, s *domain.Session) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions (id, officer_id, product_id, status, started_at)
		VALUES ($1,$2,$3,$4,$5)`,
		s.ID, s.OfficerID, s.ProductID, s.Status, s.StartedAt)
	return err
}

func (r *SessionRepo) FindByID(ctx domain.Context, id string) (*domain.Session, error) {
	var s domain.Session
	err := r.pool.QueryRow(ctx, `
		SELECT id, officer_id, product_id, status, started_at, ended_at, score
		FROM sessions WHERE id=$1`, id).
		Scan(&s.ID, &s.OfficerID, &s.ProductID, &s.Status, &s.StartedAt, &s.EndedAt, &s.Score)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) Update(ctx domain.Context, s *domain.Session) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions SET status=$2, ended_at=$3, score=$4 WHERE id=$1`,
		s.ID, s.Status, s.EndedAt, s.Score)
	return err
}

func (r *SessionRepo) ListByOfficer(ctx domain.Context, officerID string, limit int) ([]*domain.Session, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, officer_id, product_id, status, started_at, ended_at, score
		FROM sessions WHERE officer_id=$1 ORDER BY started_at DESC LIMIT $2`, officerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Session
	for rows.Next() {
		var s domain.Session
		if err := rows.Scan(&s.ID, &s.OfficerID, &s.ProductID, &s.Status, &s.StartedAt, &s.EndedAt, &s.Score); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *SessionRepo) ListRecent(ctx domain.Context, limit int) ([]*domain.Session, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, officer_id, product_id, status, started_at, ended_at, score
		FROM sessions ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Session
	for rows.Next() {
		var s domain.Session
		if err := rows.Scan(&s.ID, &s.OfficerID, &s.ProductID, &s.Status,
			&s.StartedAt, &s.EndedAt, &s.Score); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}
