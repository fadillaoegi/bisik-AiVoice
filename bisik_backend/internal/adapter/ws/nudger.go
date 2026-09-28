package ws

import (
	"context"

	"github.com/bisik/bisik_backend/internal/usecase"
)

// HubNudger membisikkan pengingat HANYA ke klien berperan officer.
//
// Keputusan: TTS dieksekusi di sisi klien (Web Speech API / flutter_tts),
// bukan streaming audio dari server. Alasannya latensi lebih rendah,
// tidak ada bandwidth audio balik, dan nasabah dijamin tidak mendengar
// karena audio keluar lewat earpiece perangkat petugas.
type HubNudger struct{ hub *Hub }

func NewHubNudger(h *Hub) *HubNudger { return &HubNudger{hub: h} }

func (n *HubNudger) Whisper(_ context.Context, sessionID string, nudge usecase.Nudge) error {
	n.hub.PublishTo(sessionID, "officer", map[string]any{
		"type":   "nudge",
		"text":   nudge.Text,
		"kind":   nudge.Kind,
		"code":   nudge.Code,
		"phrase": nudge.Phrase,
	})
	return nil
}
