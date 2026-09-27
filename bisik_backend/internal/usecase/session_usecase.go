package usecase

import (
	"context"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/google/uuid"
)

type SessionUsecase struct {
	sessions   domain.SessionRepository
	compliance domain.ComplianceRepository
	stt        SpeechToText
}

func NewSessionUsecase(s domain.SessionRepository, c domain.ComplianceRepository, stt SpeechToText) *SessionUsecase {
	return &SessionUsecase{sessions: s, compliance: c, stt: stt}
}

// Start membuat sesi baru dan menyiapkan 5 butir kewajiban dalam status pending.
func (uc *SessionUsecase) Start(ctx context.Context, officerID, productID string) (*domain.Session, error) {
	if officerID == "" {
		return nil, domain.ErrInvalidInput
	}
	s := &domain.Session{
		ID:        uuid.NewString(),
		OfficerID: officerID,
		ProductID: productID,
		Status:    domain.StatusActive,
		StartedAt: time.Now().UTC(),
	}
	if err := uc.sessions.Create(ctx, s); err != nil {
		return nil, err
	}
	if err := uc.compliance.InitStates(ctx, s.ID, domain.DefaultObligations()); err != nil {
		return nil, err
	}
	return s, nil
}

// End menutup sesi dan menghitung skor kepatuhan.
func (uc *SessionUsecase) End(ctx context.Context, sessionID string) (*domain.Session, error) {
	s, err := uc.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !s.IsRunning() {
		return nil, domain.ErrSessionEnded
	}
	// Tunggu Termination + SpeakerRevision final diproses sebelum skor.
	// Kalau upstream gagal ditutup, report tetap dibuat dari state terakhir.
	_ = uc.stt.Stop(sessionID)
	score, err := uc.Score(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	s.Status = domain.StatusEnded
	s.EndedAt = &now
	s.Score = score
	if err := uc.sessions.Update(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// Score: persentase butir terpenuhi, dikurangi 10 poin per pelanggaran.
func (uc *SessionUsecase) Score(ctx context.Context, sessionID string) (int, error) {
	states, err := uc.compliance.ListStates(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	if len(states) == 0 {
		return 0, nil
	}
	satisfied := 0
	for _, st := range states {
		if st.Status == domain.ObligationSatisfied {
			satisfied++
		}
	}
	score := satisfied * 100 / len(states)

	violations, err := uc.compliance.ListViolations(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	score -= len(violations) * 10
	if score < 0 {
		score = 0
	}
	return score, nil
}
