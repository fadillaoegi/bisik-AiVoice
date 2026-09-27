package domain

import "time"

// Speaker adalah label hasil diarization AssemblyAI.
// Label bisa direvisi saat konteks bertambah, jadi selalu perlakukan
// sebagai nilai terbaru, bukan nilai final.
type Speaker string

const (
	SpeakerUnknown  Speaker = "unknown"
	SpeakerOfficer  Speaker = "officer"  // petugas
	SpeakerCustomer Speaker = "customer" // nasabah
)

type SessionStatus string

const (
	StatusActive  SessionStatus = "active"
	StatusEnded   SessionStatus = "ended"
	StatusAborted SessionStatus = "aborted"
)

// Session adalah satu percakapan tatap muka petugas <-> nasabah.
type Session struct {
	ID        string
	OfficerID string
	ProductID string
	Status    SessionStatus
	StartedAt time.Time
	EndedAt   *time.Time
	Score     int // 0-100, dihitung saat sesi berakhir
}

func (s *Session) IsRunning() bool { return s.Status == StatusActive }

type SessionRepository interface {
	Create(ctx Context, s *Session) error
	FindByID(ctx Context, id string) (*Session, error)
	Update(ctx Context, s *Session) error
	ListByOfficer(ctx Context, officerID string, limit int) ([]*Session, error)
	// ListRecent dipakai supervisor untuk memilih sesi yang akan dipantau.
	ListRecent(ctx Context, limit int) ([]*Session, error)
}
