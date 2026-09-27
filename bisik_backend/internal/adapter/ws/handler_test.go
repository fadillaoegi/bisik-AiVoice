package ws

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bisik/bisik_backend/internal/usecase"
)

type errorEventSTT struct {
	events chan usecase.TranscriptEvent
}

func (f *errorEventSTT) Start(context.Context, string) (<-chan usecase.TranscriptEvent, error) {
	return f.events, nil
}

func (f *errorEventSTT) PushAudio(string, []byte) error { return nil }
func (f *errorEventSTT) Stop(string) error              { return nil }

// Event error juga memakai flush barrier. Adapter wajib mengakuinya supaya
// channel upstream dapat ditutup, pipeline membatalkan context, dan nudge
// tidak terus berjalan setelah STT mati.
func TestPipelineMengakuiEventError(t *testing.T) {
	events := make(chan usecase.TranscriptEvent, 1)
	stt := &errorEventSTT{events: events}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := NewHub(log)
	officer := &Client{role: "officer", send: make(chan []byte, 1)}
	hub.Join("sesi-1", officer)

	h := &Handler{hub: hub, stt: stt, log: log}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := h.startPipeline(ctx, cancel, "sesi-1"); err != nil {
		t.Fatalf("start pipeline: %v", err)
	}

	acked := make(chan struct{})
	events <- usecase.TranscriptEvent{
		Err: errors.New("diarizer terputus"),
		Acknowledge: func() {
			close(acked)
		},
	}
	close(events)

	select {
	case <-acked:
	case <-time.After(2 * time.Second):
		t.Fatal("adapter tidak mengakui event error")
	}

	select {
	case payload := <-officer.send:
		if !strings.Contains(string(payload), `"type":"session_error"`) {
			t.Fatalf("event ke petugas = %s, mau session_error", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("petugas tidak menerima session_error")
	}

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context pipeline tidak dibatalkan setelah stream selesai")
	}
}
