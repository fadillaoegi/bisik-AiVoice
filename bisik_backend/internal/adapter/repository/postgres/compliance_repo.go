package postgres

import (
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ComplianceRepo struct{ pool *pgxpool.Pool }

func NewComplianceRepo(p *pgxpool.Pool) *ComplianceRepo { return &ComplianceRepo{pool: p} }

func (r *ComplianceRepo) InitStates(ctx domain.Context, sessionID string, obligations []domain.Obligation) error {
	batch := make([][]any, 0, len(obligations))
	for _, ob := range obligations {
		batch = append(batch, []any{sessionID, ob.Code, domain.ObligationPending})
	}
	for _, row := range batch {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO obligation_states (session_id, code, status)
			VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, row...); err != nil {
			return err
		}
	}
	return nil
}

func (r *ComplianceRepo) ListStates(ctx domain.Context, sessionID string) ([]*domain.ObligationState, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT session_id, code, status, confidence, COALESCE(evidence_id,''), satisfied_at
		FROM obligation_states WHERE session_id=$1 ORDER BY code`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.ObligationState
	for rows.Next() {
		var s domain.ObligationState
		if err := rows.Scan(&s.SessionID, &s.Code, &s.Status, &s.Confidence, &s.EvidenceID, &s.SatisfiedAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *ComplianceRepo) MarkSatisfied(ctx domain.Context, sessionID, code, evidenceID string, confidence float64) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE obligation_states
		SET status=$3, evidence_id=$4, confidence=$5, satisfied_at=$6
		WHERE session_id=$1 AND code=$2`,
		sessionID, code, domain.ObligationSatisfied, evidenceID, confidence, now)
	return err
}

func (r *ComplianceRepo) RecordViolation(ctx domain.Context, v *domain.Violation) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO violations (id, session_id, phrase, severity, evidence_id, detected_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		v.ID, v.SessionID, v.Phrase, v.Severity, v.EvidenceID, v.DetectedAt)
	return err
}

func (r *ComplianceRepo) ListViolations(ctx domain.Context, sessionID string) ([]*domain.Violation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, session_id, phrase, severity, COALESCE(evidence_id,''), detected_at
		FROM violations WHERE session_id=$1 ORDER BY detected_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Violation
	for rows.Next() {
		var v domain.Violation
		if err := rows.Scan(&v.ID, &v.SessionID, &v.Phrase, &v.Severity, &v.EvidenceID, &v.DetectedAt); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}
