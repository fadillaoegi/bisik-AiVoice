package usecase

import (
	"context"

	"github.com/bisik/bisik_backend/internal/domain"
)

// TranscriptEvent dikirim oleh penyedia STT setiap kali ada hasil.
type TranscriptEvent struct {
	UtteranceID string
	Speaker     domain.Speaker
	// SourceSpeaker adalah label mentah diarization (mis. "A"/"B").
	// Nilai ini hanya ditampilkan saat kalibrasi dan tidak menjadi identitas.
	SourceSpeaker string
	Text          string
	StartMS       int
	EndMS         int
	IsFinal       bool
	IsRevision    bool // diarization merevisi label sebelumnya
	IsCalibration bool // ucapan kalibrasi tidak disimpan atau dinilai
	// Acknowledge dipanggil adapter setelah event selesai diproses. Upstream
	// memakai ack ini saat finalisasi agar report tidak mendahului revision.
	Acknowledge func()
	// Err diisi kalau stream upstream berhenti sebelum sesi diakhiri klien.
	// Adapter wajib memberi tahu petugas: tanpa ini UI tetap tampak merekam
	// padahal tidak ada lagi transkrip yang masuk.
	Err error
}

// SpeakerRoleCalibrator mengaktifkan pemetaan role eksplisit untuk PWA dan
// mobile. Klien lama yang belum mengirim perintah kalibrasi tetap dapat
// memakai fallback pemetaan otomatis.
type SpeakerRoleCalibrator interface {
	BeginSpeakerCalibration(sessionID string) error
	ConfirmSpeakerRoles(sessionID, officerLabel, customerLabel string) error
	SpeakerCalibrationActive(sessionID string) bool
}

// SpeechToText adalah port ke AssemblyAI Streaming STT + diarization.
// Implementasinya ada di infrastructure/assemblyai.
type SpeechToText interface {
	// Start membuka koneksi upstream dan mengembalikan channel event.
	Start(ctx context.Context, sessionID string) (<-chan TranscriptEvent, error)
	// PushAudio mengirim frame PCM16 ke upstream.
	PushAudio(sessionID string, pcm []byte) error
	Stop(sessionID string) error
}

// SemanticMatcher menilai apakah sebuah ucapan memenuhi butir kewajiban.
// Implementasi: AssemblyAI LLM Gateway.
type SemanticMatcher interface {
	Match(ctx context.Context, utterance string, obligation domain.Obligation) (matched bool, confidence float64, err error)
}

// Guardrail mendeteksi janji terlarang dalam ucapan petugas.
type Guardrail interface {
	Inspect(ctx context.Context, utterance string) (phrase string, severity string, found bool, err error)
}

// Nudger membisikkan pengingat HANYA ke earpiece petugas.
type Nudger interface {
	Whisper(ctx context.Context, sessionID string, text string) error
}

// Broadcaster mendorong update realtime ke klien (officer + supervisor).
type Broadcaster interface {
	Publish(sessionID string, event any)
}
