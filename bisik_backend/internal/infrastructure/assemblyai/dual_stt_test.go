package assemblyai

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/usecase"
)

func TestDominantLabelMemilihTumpangTindihTerbanyak(t *testing.T) {
	segments := []speakerSegment{
		{startMS: 0, endMS: 4000, label: "A"},
		{startMS: 4000, endMS: 9000, label: "B"},
	}

	// Ucapan 4500–8500 hampir seluruhnya milik B.
	if got := dominantLabel(segments, 4500, 8500); got != "B" {
		t.Fatalf("label = %q, mau B", got)
	}

	// Ucapan 0–5000 sebagian besar A (4000 ms) dengan sedikit B (1000 ms).
	// Yang menang harus durasi terbanyak, bukan yang terakhir menyentuh.
	if got := dominantLabel(segments, 0, 5000); got != "A" {
		t.Fatalf("label = %q, mau A", got)
	}
}

func TestDominantLabelKosongSaatTidakAdaSegmen(t *testing.T) {
	// Belum ada label yang menutupi rentang ini. Harus kosong, bukan menebak —
	// ucapan tanpa identitas tidak boleh memenuhi kewajiban.
	if got := dominantLabel(nil, 1000, 2000); got != "" {
		t.Fatalf("label = %q, mau kosong", got)
	}
	segments := []speakerSegment{{startMS: 0, endMS: 500, label: "A"}}
	if got := dominantLabel(segments, 1000, 2000); got != "" {
		t.Fatalf("label = %q, mau kosong — segmen tidak bersentuhan", got)
	}
}

func TestDominantLabelMenanganiRentangNol(t *testing.T) {
	// Ucapan sangat pendek bisa datang dengan start == end.
	segments := []speakerSegment{{startMS: 0, endMS: 3000, label: "A"}}
	if got := dominantLabel(segments, 1500, 1500); got != "A" {
		t.Fatalf("label = %q, mau A", got)
	}
}

// PENDING adalah "belum diputuskan", bukan identitas. Membiarkannya lolos
// akan memunculkan kartu suara hantu saat kalibrasi dan memberinya role.
func TestUsableLabelMenolakLabelYangBelumPasti(t *testing.T) {
	for _, label := range []string{"", "UNKNOWN", "PENDING"} {
		if got := usableLabel(label); got != "" {
			t.Fatalf("usableLabel(%q) = %q, mau kosong", label, got)
		}
	}
	for _, label := range []string{"A", "B", "C"} {
		if got := usableLabel(label); got != label {
			t.Fatalf("usableLabel(%q) = %q, mau tetap", label, got)
		}
	}
}

// --- Uji loop penggabung tanpa menyentuh jaringan ---

func newMergeHarness(t *testing.T) (*DualStreamSTT, *dualSession, chan usecase.TranscriptEvent, chan usecase.TranscriptEvent) {
	t.Helper()
	diarizer := NewStreamingSTT("key", "ws://example.test", "multilingual", 120000, testLogger())
	diarizer.speakers["sesi-1"] = &speakerSession{
		roles: map[string]domain.Speaker{}, calibrationTurns: map[int]struct{}{},
	}
	dual := NewDualStreamSTT(
		NewStreamingSTT("key", "ws://example.test", "whisper-rt", 120000, testLogger()),
		diarizer, testLogger(),
	)
	session := &dualSession{
		emitted: make(map[string]emittedTurn),
		roles:   make(map[string]domain.Speaker),
		out:     make(chan usecase.TranscriptEvent, 16),
		done:    make(chan struct{}),
	}
	return dual, session, make(chan usecase.TranscriptEvent), make(chan usecase.TranscriptEvent)
}

// withAck meniru kontrak StreamingSTT: pengirim memblokir sampai diakui.
func withAck(ev usecase.TranscriptEvent) (usecase.TranscriptEvent, chan struct{}) {
	acked := make(chan struct{})
	var once sync.Once
	ev.Acknowledge = func() { once.Do(func() { close(acked) }) }
	return ev, acked
}

// Regresi: penggabung pernah lupa mengakui event upstream, sehingga readLoop
// membeku setelah SATU ucapan dan seluruh sesi berhenti. Tidak terlihat oleh
// test mana pun sampai diuji terhadap API sungguhan.
func TestPenggabungMengakuiEventUpstream(t *testing.T) {
	dual, session, textCh, labelCh := newMergeHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dual.merge(ctx, "sesi-1", session, textCh, labelCh)

	// Hilir dikuras di goroutine terpisah, seperti adapter WebSocket.
	// Pengakuan upstream memang baru terjadi SETELAH hilir mengakui — itulah
	// yang menjaga `Terminate` tetap menjadi flush barrier.
	merged := make(chan usecase.TranscriptEvent, 4)
	go func() {
		for ev := range session.out {
			if ev.Acknowledge != nil {
				ev.Acknowledge()
			}
			merged <- ev
		}
	}()

	label, labelAcked := withAck(usecase.TranscriptEvent{
		SourceSpeaker: "A", StartMS: 0, EndMS: 5000, IsFinal: true,
	})
	labelCh <- label
	select {
	case <-labelAcked:
	case <-time.After(2 * time.Second):
		t.Fatal("event diarizer tidak pernah diakui — readLoop akan membeku")
	}

	text, textAcked := withAck(usecase.TranscriptEvent{
		UtteranceID: "u-1", Text: "halo", StartMS: 1000, EndMS: 4000, IsFinal: true,
	})
	textCh <- text
	select {
	case <-textAcked:
	case <-time.After(2 * time.Second):
		t.Fatal("event transcriber tidak pernah diakui — readLoop akan membeku")
	}

	select {
	case got := <-merged:
		if got.SourceSpeaker != "A" {
			t.Fatalf("label = %q, mau A", got.SourceSpeaker)
		}
		if got.Text != "halo" {
			t.Fatalf("teks = %q, mau halo", got.Text)
		}
		if got.Speaker != domain.SpeakerOfficer {
			t.Fatalf("role = %q, mau officer", got.Speaker)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tidak ada event gabungan yang keluar")
	}
}

// Transkrip yang labelnya tidak kunjung datang harus tetap dilepas sebagai
// unknown, bukan tertahan selamanya. Ucapan tanpa identitas memang tidak
// boleh memenuhi kewajiban, tetapi ia tetap harus masuk transkrip.
func TestTranskripTanpaLabelTetapDilepas(t *testing.T) {
	dual, session, textCh, labelCh := newMergeHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dual.merge(ctx, "sesi-1", session, textCh, labelCh)

	text, _ := withAck(usecase.TranscriptEvent{
		UtteranceID: "u-1", Text: "tanpa label", StartMS: 0, EndMS: 3000, IsFinal: true,
	})
	go func() { textCh <- text }()

	select {
	case got := <-session.out:
		if got.SourceSpeaker != "" || got.Speaker != domain.SpeakerUnknown {
			t.Fatalf("label=%q role=%q, mau kosong/unknown", got.SourceSpeaker, got.Speaker)
		}
		got.Acknowledge()
	case <-time.After(3 * time.Second):
		t.Fatal("ucapan tanpa label tertahan selamanya")
	}
}

// Regresi finalisasi: diarizer selesai lebih dulu dan membersihkan state
// speaker internalnya. Turn terakhir dari transcriber tetap harus memakai
// mapping yang sudah dikonfirmasi, bukan jatuh menjadi unknown.
func TestMappingRoleTetapAdaSetelahDiarizerCleanup(t *testing.T) {
	dual, session, _, _ := newMergeHarness(t)
	dual.sessions["sesi-1"] = session

	if err := dual.ConfirmSpeakerRoles("sesi-1", "A", "B"); err != nil {
		t.Fatalf("konfirmasi role: %v", err)
	}

	// Meniru cleanup StreamingSTT ketika koneksi diarizer menerima
	// Termination, sebelum transcriber mengeluarkan turn terakhir.
	dual.diarizer.mu.Lock()
	delete(dual.diarizer.speakers, "sesi-1")
	dual.diarizer.mu.Unlock()

	session.mu.Lock()
	session.segments = []speakerSegment{{startMS: 0, endMS: 4000, label: "A"}}
	session.pending = []pendingTurn{{
		event: usecase.TranscriptEvent{
			UtteranceID: "u-terakhir", Text: "hak membatalkan",
			StartMS: 1000, EndMS: 3000, IsFinal: true,
		},
		deadline: time.Now(),
	}}
	session.mu.Unlock()

	done := make(chan struct{})
	go func() {
		dual.resolvePending(context.Background(), "sesi-1", session, true)
		close(done)
	}()

	select {
	case got := <-session.out:
		if got.Speaker != domain.SpeakerOfficer {
			t.Fatalf("role turn terakhir = %q, mau officer", got.Speaker)
		}
		got.Acknowledge()
	case <-time.After(2 * time.Second):
		t.Fatal("turn terakhir tidak diteruskan")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("finalisasi tertahan setelah event diakui")
	}
}

// Di dual-stream, kehilangan diarizer sama fatalnya dengan kehilangan teks:
// laporan tidak boleh diam-diam berjalan tanpa tahu siapa yang berbicara.
func TestErrorDiarizerDiteruskanKeHilir(t *testing.T) {
	dual, session, textCh, labelCh := newMergeHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dual.merge(ctx, "sesi-1", session, textCh, labelCh)

	want := errors.New("diarizer terputus")
	event, upstreamAcked := withAck(usecase.TranscriptEvent{Err: want})
	go func() { labelCh <- event }()

	select {
	case got := <-session.out:
		if !errors.Is(got.Err, want) {
			t.Fatalf("error hilir = %v, mau %v", got.Err, want)
		}
		got.Acknowledge()
	case <-time.After(2 * time.Second):
		t.Fatal("error diarizer tidak diteruskan ke hilir")
	}

	select {
	case <-upstreamAcked:
	case <-time.After(2 * time.Second):
		t.Fatal("event error diarizer tidak diakui")
	}
	select {
	case <-session.done:
	case <-time.After(2 * time.Second):
		t.Fatal("penggabung tidak berhenti setelah diarizer gagal")
	}
}
