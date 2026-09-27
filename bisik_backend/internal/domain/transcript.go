package domain

import "time"

// Utterance adalah satu potongan ucapan yang sudah final dari STT.
type Utterance struct {
	ID        string
	SessionID string
	Speaker   Speaker
	// SourceSpeaker adalah label mentah diarization (mis. A/B) untuk audit.
	SourceSpeaker string
	Text          string
	StartMS       int
	EndMS         int
	Revised       bool // true kalau label speaker pernah direvisi diarization
	CreatedAt     time.Time
}

type TranscriptRepository interface {
	Append(ctx Context, u *Utterance) error
	FindByID(ctx Context, id string) (*Utterance, error)
	ListBySession(ctx Context, sessionID string) ([]*Utterance, error)
	// ReviseSpeaker dipakai saat diarization memperbaiki label sebelumnya.
	ReviseSpeaker(ctx Context, utteranceID string, speaker Speaker) error
}
