package usecase

import (
	"context"
	"testing"

	"github.com/bisik/bisik_backend/internal/domain"
)

type fakeCompliance struct {
	states     []*domain.ObligationState
	violations []*domain.Violation
}

type fakeSessionRepo struct {
	session *domain.Session
}

func (f *fakeSessionRepo) Create(domain.Context, *domain.Session) error { return nil }
func (f *fakeSessionRepo) FindByID(domain.Context, string) (*domain.Session, error) {
	return f.session, nil
}
func (f *fakeSessionRepo) Update(_ domain.Context, session *domain.Session) error {
	f.session = session
	return nil
}
func (f *fakeSessionRepo) ListByOfficer(domain.Context, string, int) ([]*domain.Session, error) {
	return nil, nil
}

func (f *fakeSessionRepo) ListRecent(domain.Context, int) ([]*domain.Session, error) {
	return nil, nil
}

type fakeSTT struct {
	stop func()
}

func (f *fakeSTT) Start(context.Context, string) (<-chan TranscriptEvent, error) {
	return nil, nil
}
func (f *fakeSTT) PushAudio(string, []byte) error { return nil }
func (f *fakeSTT) Stop(string) error {
	if f.stop != nil {
		f.stop()
	}
	return nil
}

func (f *fakeCompliance) InitStates(domain.Context, string, []domain.Obligation) error { return nil }
func (f *fakeCompliance) ListStates(domain.Context, string) ([]*domain.ObligationState, error) {
	return f.states, nil
}
func (f *fakeCompliance) MarkSatisfied(domain.Context, string, string, string, float64) error {
	return nil
}
func (f *fakeCompliance) RecordViolation(domain.Context, *domain.Violation) error { return nil }
func (f *fakeCompliance) ListViolations(domain.Context, string) ([]*domain.Violation, error) {
	return f.violations, nil
}

func states(satisfied, pending int) []*domain.ObligationState {
	out := make([]*domain.ObligationState, 0, satisfied+pending)
	for range satisfied {
		out = append(out, &domain.ObligationState{Status: domain.ObligationSatisfied})
	}
	for range pending {
		out = append(out, &domain.ObligationState{Status: domain.ObligationPending})
	}
	return out
}

func TestScore(t *testing.T) {
	cases := []struct {
		name       string
		satisfied  int
		pending    int
		violations int
		want       int
	}{
		{"semua terpenuhi tanpa pelanggaran", 5, 0, 0, 100},
		{"tiga dari lima", 3, 2, 0, 60},
		{"dipotong satu pelanggaran", 5, 0, 1, 90},
		{"tidak pernah negatif", 0, 5, 3, 0},
		{"tanpa butir sama sekali", 0, 0, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeCompliance{states: states(tc.satisfied, tc.pending)}
			for range tc.violations {
				fake.violations = append(fake.violations, &domain.Violation{})
			}

			uc := NewSessionUsecase(nil, fake, nil)
			got, err := uc.Score(context.Background(), "sesi-1")
			if err != nil {
				t.Fatalf("tidak mengharapkan error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Score() = %d, mau %d", got, tc.want)
			}
		})
	}
}

func TestEndMenungguStopSebelumMenghitungScore(t *testing.T) {
	compliance := &fakeCompliance{states: states(0, 1)}
	repository := &fakeSessionRepo{session: &domain.Session{ID: "sesi-1", Status: domain.StatusActive}}
	stt := &fakeSTT{stop: func() {
		compliance.states[0].Status = domain.ObligationSatisfied
	}}
	uc := NewSessionUsecase(repository, compliance, stt)

	ended, err := uc.End(context.Background(), "sesi-1")
	if err != nil {
		t.Fatalf("End(): %v", err)
	}
	if ended.Score != 100 {
		t.Fatalf("score = %d, mau 100 setelah finalisasi STT", ended.Score)
	}
	if ended.Status != domain.StatusEnded {
		t.Fatalf("status = %q, mau ended", ended.Status)
	}
}
