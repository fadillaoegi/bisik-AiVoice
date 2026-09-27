package domain

import "time"

// ObligationStatus adalah status satu butir kewajiban dalam satu sesi.
type ObligationStatus string

const (
	ObligationPending   ObligationStatus = "pending"
	ObligationSatisfied ObligationStatus = "satisfied"
	ObligationViolated  ObligationStatus = "violated"
)

// Obligation adalah satu butir yang WAJIB diucapkan petugas.
// Scope hackathon: 5 butir, hardcoded. Jangan bikin editor rule.
type Obligation struct {
	Code        string
	Label       string
	Description string // dipakai sebagai prompt semantic match
	Required    bool
}

// ObligationState adalah progres satu butir di dalam satu sesi berjalan.
type ObligationState struct {
	SessionID   string
	Code        string
	Status      ObligationStatus
	Confidence  float64
	EvidenceID  string // Utterance.ID yang memenuhi butir ini
	SatisfiedAt *time.Time
}

// Violation adalah janji terlarang yang terdeteksi Guardrails.
type Violation struct {
	ID         string
	SessionID  string
	Phrase     string
	Severity   string
	EvidenceID string
	DetectedAt time.Time
}

type ComplianceRepository interface {
	InitStates(ctx Context, sessionID string, obligations []Obligation) error
	ListStates(ctx Context, sessionID string) ([]*ObligationState, error)
	MarkSatisfied(ctx Context, sessionID, code, evidenceID string, confidence float64) error
	RecordViolation(ctx Context, v *Violation) error
	ListViolations(ctx Context, sessionID string) ([]*Violation, error)
}

// DefaultObligations = 5 butir kewajiban penyampaian produk keuangan.
func DefaultObligations() []Obligation {
	return []Obligation{
		{Code: "IDENTITY", Label: "Identitas & lembaga", Description: "Petugas menyebutkan nama dirinya dan nama lembaga/perusahaan tempatnya bekerja.", Required: true},
		{Code: "RATE", Label: "Suku bunga / biaya", Description: "Petugas menyebutkan suku bunga atau total biaya yang harus dibayar nasabah secara eksplisit.", Required: true},
		{Code: "TENOR", Label: "Jangka waktu & cicilan", Description: "Petugas menyebutkan jangka waktu pembayaran dan besaran cicilan per periode.", Required: true},
		{Code: "PENALTY", Label: "Denda keterlambatan", Description: "Petugas menjelaskan konsekuensi atau denda jika nasabah terlambat membayar.", Required: true},
		{Code: "RIGHT", Label: "Hak membatalkan", Description: "Petugas memberi tahu nasabah bahwa nasabah berhak menolak atau membatalkan penawaran ini.", Required: true},
	}
}

// ForbiddenPhrases = janji terlarang yang memicu Violation.
func ForbiddenPhrases() []string {
	return []string{
		"dijamin untung",
		"pasti cair",
		"pasti disetujui",
		"tanpa risiko",
		"bebas denda",
		"dijamin approve",
	}
}
