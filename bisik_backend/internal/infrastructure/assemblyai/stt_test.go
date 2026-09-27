package assemblyai

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/usecase"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestConnectionParamsUntukBahasaIndonesia(t *testing.T) {
	stt := NewStreamingSTT("key", "ws://example.test", "", 5000, testLogger())
	q := stt.connectionParams()

	if got := q.Get("speech_model"); got != "whisper-rt" {
		t.Fatalf("speech_model = %q, mau whisper-rt", got)
	}
	// Nilai dikirim apa adanya, tidak dipaksa naik. Clamp diam-diam membuat
	// interval pendek mustahil diuji, padahal itu tersangka utama kenapa
	// suara kedua tidak muncul saat kalibrasi.
	if got := q.Get("speaker_labels_revision_interval_ms"); got != "5000" {
		t.Fatalf("revision interval = %q, mau 5000 (apa adanya)", got)
	}
	if got := q.Get("format_turns"); got != "" {
		t.Fatalf("format_turns tidak boleh dikirim untuk whisper-rt, dapat %q", got)
	}
}

func TestIntervalRevisiTidakMasukAkalDikembalikanKeMinimum(t *testing.T) {
	// Nol atau negatif bukan eksperimen, itu salah konfigurasi.
	stt := NewStreamingSTT("key", "ws://example.test", "", 0, testLogger())
	if got := stt.connectionParams().Get("speaker_labels_revision_interval_ms"); got != "120000" {
		t.Fatalf("interval = %q, mau kembali ke 120000", got)
	}
}

func TestConnectionParamsUniversalStreamingMemintaFormat(t *testing.T) {
	stt := NewStreamingSTT("key", "ws://example.test", "universal-streaming-multilingual", 300000, testLogger())
	if got := stt.connectionParams().Get("format_turns"); got != "true" {
		t.Fatalf("format_turns = %q, mau true", got)
	}
}

func TestResolveRoleStabilPerLabel(t *testing.T) {
	stt := NewStreamingSTT("key", "ws://example.test", "whisper-rt", 120000, testLogger())
	stt.speakers["sesi-1"] = &speakerSession{
		roles: map[string]domain.Speaker{}, calibrationTurns: map[int]struct{}{},
	}

	if got, _ := stt.resolveSpeaker("sesi-1", "A", 1); got != domain.SpeakerOfficer {
		t.Fatalf("speaker pertama = %q, mau officer", got)
	}
	if got, _ := stt.resolveSpeaker("sesi-1", "B", 2); got != domain.SpeakerCustomer {
		t.Fatalf("speaker kedua = %q, mau customer", got)
	}
	if got, _ := stt.resolveSpeaker("sesi-1", "A", 3); got != domain.SpeakerOfficer {
		t.Fatalf("label A berubah menjadi %q", got)
	}
	if got, _ := stt.resolveSpeaker("sesi-1", "UNKNOWN", 4); got != domain.SpeakerUnknown {
		t.Fatalf("UNKNOWN = %q, mau unknown", got)
	}
}

func TestKalibrasiMenahanRoleSampaiDikonfirmasi(t *testing.T) {
	stt := NewStreamingSTT("key", "ws://example.test", "whisper-rt", 120000, testLogger())
	stt.speakers["sesi-1"] = &speakerSession{
		roles: map[string]domain.Speaker{}, calibrationTurns: map[int]struct{}{},
	}

	if err := stt.BeginSpeakerCalibration("sesi-1"); err != nil {
		t.Fatalf("BeginSpeakerCalibration(): %v", err)
	}
	if got, calibrating := stt.resolveSpeaker("sesi-1", "B", 7); got != domain.SpeakerUnknown || !calibrating {
		t.Fatalf("saat kalibrasi got=(%q,%v), mau (unknown,true)", got, calibrating)
	}
	if !stt.isCalibrationTurn("sesi-1", 7) {
		t.Fatal("turn kalibrasi tidak ditandai")
	}

	if err := stt.ConfirmSpeakerRoles("sesi-1", "B", "A"); err != nil {
		t.Fatalf("ConfirmSpeakerRoles(): %v", err)
	}
	if got, calibrating := stt.resolveSpeaker("sesi-1", "B", 8); got != domain.SpeakerOfficer || calibrating {
		t.Fatalf("label B got=(%q,%v), mau (officer,false)", got, calibrating)
	}
	if got, _ := stt.resolveSpeaker("sesi-1", "A", 9); got != domain.SpeakerCustomer {
		t.Fatalf("label A = %q, mau customer", got)
	}
}

func TestConfirmSpeakerRolesMenolakLabelSama(t *testing.T) {
	stt := NewStreamingSTT("key", "ws://example.test", "whisper-rt", 120000, testLogger())
	stt.speakers["sesi-1"] = &speakerSession{
		roles: map[string]domain.Speaker{}, calibrationTurns: map[int]struct{}{},
	}
	if err := stt.ConfirmSpeakerRoles("sesi-1", "A", "A"); err == nil {
		t.Fatal("label petugas dan nasabah yang sama harus ditolak")
	}
}

func TestEmitTranscriptEventMenungguAcknowledgement(t *testing.T) {
	out := make(chan usecase.TranscriptEvent)
	finished := make(chan bool, 1)

	go func() {
		finished <- emitTranscriptEvent(context.Background(), out, usecase.TranscriptEvent{Text: "uji"})
	}()

	event := <-out
	select {
	case <-finished:
		t.Fatal("emit selesai sebelum event di-acknowledge")
	case <-time.After(20 * time.Millisecond):
	}

	event.Acknowledge()
	select {
	case ok := <-finished:
		if !ok {
			t.Fatal("emit mengembalikan false setelah acknowledgement")
		}
	case <-time.After(time.Second):
		t.Fatal("emit tidak selesai setelah acknowledgement")
	}
}

// Regresi live 25 Sep: AudioWorklet mengirim 128 sample (256 byte) per frame
// sehingga AssemblyAI menutup koneksi dan sesi berikutnya gagal dengan
// "resource tidak ditemukan". Frame keluar wajib berada di 50–1000 ms.
func TestAppendAudioMenahanFrameDiBawahMinimum(t *testing.T) {
	stream := &streamConnection{}

	// 6 x 256 byte = 1536 byte, masih di bawah 4096: belum boleh dikirim.
	for i := 0; i < 6; i++ {
		if frames := stream.appendAudio(make([]byte, 256)); frames != nil {
			t.Fatalf("frame ke-%d: potongan 8 ms tidak boleh diteruskan, dapat %d frame", i, len(frames))
		}
	}

	// Potongan berikutnya melewati 4096 byte dan memicu satu frame penuh.
	frames := stream.appendAudio(make([]byte, 2560))
	if len(frames) != 1 {
		t.Fatalf("mau 1 frame setelah melewati batas, dapat %d", len(frames))
	}
	if len(frames[0]) != RecommendedChunkBytes {
		t.Fatalf("ukuran frame = %d byte, mau %d", len(frames[0]), RecommendedChunkBytes)
	}
}

func TestAppendAudioMemotongFrameDiAtasMaksimum(t *testing.T) {
	stream := &streamConnection{}

	// Satu potongan 2 detik (64000 byte) melewati batas atas 1000 ms.
	frames := stream.appendAudio(make([]byte, 64000))
	if len(frames) == 0 {
		t.Fatal("potongan besar harus menghasilkan frame")
	}

	total := 0
	for i, frame := range frames {
		if len(frame) > MaximumFrameBytes {
			t.Fatalf("frame %d = %d byte, melewati batas %d", i, len(frame), MaximumFrameBytes)
		}
		if len(frame) < MinimumFrameBytes {
			t.Fatalf("frame %d = %d byte, di bawah batas %d", i, len(frame), MinimumFrameBytes)
		}
		total += len(frame)
	}
	if rest := stream.drainAudio(); rest != nil {
		total += len(rest)
	}
	if total != 64000 {
		t.Fatalf("total byte terkirim = %d, mau 64000 (tidak boleh ada audio hilang)", total)
	}
}

func TestDrainAudioMembuangSisaTerlaluPendek(t *testing.T) {
	stream := &streamConnection{}

	// Sisa 800 byte = 25 ms: ditolak AssemblyAI, jadi sengaja dibuang.
	stream.appendAudio(make([]byte, 800))
	if rest := stream.drainAudio(); rest != nil {
		t.Fatalf("sisa 25 ms harus dibuang, dapat %d byte", len(rest))
	}

	// Sisa 1600 byte = 50 ms: tepat di batas dan masih boleh dikirim.
	stream.appendAudio(make([]byte, MinimumFrameBytes))
	rest := stream.drainAudio()
	if len(rest) != MinimumFrameBytes {
		t.Fatalf("sisa 50 ms harus ikut terkirim, dapat %d byte", len(rest))
	}
	if again := stream.drainAudio(); again != nil {
		t.Fatal("drain kedua harus kosong")
	}
}

// Setelah manusia mengunci mapping A/B, label ketiga berarti ada orang lain
// ikut bicara. Dia tidak boleh mewarisi role siapa pun — termasuk nasabah —
// karena transkrip akan berbohong soal siapa yang mengucapkan apa.
func TestLabelKetigaSetelahKonfirmasiJadiUnknown(t *testing.T) {
	stt := NewStreamingSTT("key", "ws://example.test", "whisper-rt", 120000, testLogger())
	stt.speakers["sesi-1"] = &speakerSession{
		roles: map[string]domain.Speaker{}, calibrationTurns: map[int]struct{}{},
	}

	if err := stt.ConfirmSpeakerRoles("sesi-1", "A", "B"); err != nil {
		t.Fatalf("ConfirmSpeakerRoles(): %v", err)
	}

	if role, _ := stt.resolveSpeaker("sesi-1", "A", 1); role != domain.SpeakerOfficer {
		t.Fatalf("label A = %q, mau officer", role)
	}
	if role, _ := stt.resolveSpeaker("sesi-1", "B", 2); role != domain.SpeakerCustomer {
		t.Fatalf("label B = %q, mau customer", role)
	}
	if role, _ := stt.resolveSpeaker("sesi-1", "C", 3); role != domain.SpeakerUnknown {
		t.Fatalf("label C = %q, mau unknown — pembicara ketiga bukan role terkalibrasi", role)
	}
}

// Klien lama (Flutter) belum punya UI kalibrasi, jadi fallback urutan bicara
// harus tetap hidup selama mapping belum pernah dikonfirmasi manusia.
func TestFallbackUrutanBicaraTetapBerlakuTanpaKonfirmasi(t *testing.T) {
	stt := NewStreamingSTT("key", "ws://example.test", "whisper-rt", 120000, testLogger())
	stt.speakers["sesi-1"] = &speakerSession{
		roles: map[string]domain.Speaker{}, calibrationTurns: map[int]struct{}{},
	}

	if role, _ := stt.resolveSpeaker("sesi-1", "A", 1); role != domain.SpeakerOfficer {
		t.Fatalf("pembicara pertama = %q, mau officer", role)
	}
	if role, _ := stt.resolveSpeaker("sesi-1", "B", 2); role != domain.SpeakerCustomer {
		t.Fatalf("pembicara kedua = %q, mau customer", role)
	}
}

func TestMaxSpeakersMenyediakanSlotOrangKetiga(t *testing.T) {
	stt := NewStreamingSTT("key", "ws://example.test", "whisper-rt", 120000, testLogger())
	if got := stt.connectionParams().Get("max_speakers"); got != "3" {
		t.Fatalf("max_speakers = %q, mau 3 — batas 2 memaksa orang ketiga jadi role terkalibrasi", got)
	}
}

func TestLabelFromWordsMemilihPembicaraTerbanyak(t *testing.T) {
	label := labelFromWords([]wsWord{
		{Text: "saya", Speaker: "B"},
		{Text: "nasabah", Speaker: "B"},
		{Text: "eh", Speaker: "A"},
	})
	if label != "B" {
		t.Fatalf("label = %q, mau B", label)
	}

	// Label kosong dan UNKNOWN tidak boleh ikut dihitung.
	if got := labelFromWords([]wsWord{{Speaker: ""}, {Speaker: "UNKNOWN"}}); got != "" {
		t.Fatalf("label = %q, mau kosong", got)
	}
}

func TestTextFromWordsMenyusunUlangKalimatRevisi(t *testing.T) {
	// Pesan SpeakerRevision tidak membawa `transcript`, hanya words[].
	got := textFromWords([]wsWord{{Text: "saya"}, {Text: "nasabah"}, {Text: ""}})
	if got != "saya nasabah" {
		t.Fatalf("teks = %q, mau \"saya nasabah\"", got)
	}
}

// Regresi uji lapangan 25 Sep: hanya suara pertama yang terdeteksi saat
// kalibrasi. Penyebabnya revisi untuk turn kalibrasi dibuang, padahal justru
// revisi itulah cara diarization memberi tahu ada suara kedua.
func TestTurnKalibrasiDitandaiDanTetapBisaDirevisi(t *testing.T) {
	stt := NewStreamingSTT("key", "ws://example.test", "whisper-rt", 120000, testLogger())
	stt.speakers["sesi-1"] = &speakerSession{
		roles: map[string]domain.Speaker{}, calibrationTurns: map[int]struct{}{},
	}
	if err := stt.BeginSpeakerCalibration("sesi-1"); err != nil {
		t.Fatalf("BeginSpeakerCalibration(): %v", err)
	}

	// Dua orang bicara, tetapi diarization masih melabeli keduanya "A".
	for _, turn := range []int{1, 2} {
		role, calibrating := stt.resolveSpeaker("sesi-1", "A", turn)
		if role != domain.SpeakerUnknown || !calibrating {
			t.Fatalf("turn %d = %q calibrating=%v, mau unknown/true", turn, role, calibrating)
		}
	}

	// Keduanya harus tercatat sebagai turn kalibrasi supaya tidak dinilai...
	for _, turn := range []int{1, 2} {
		if !stt.isCalibrationTurn("sesi-1", turn) {
			t.Fatalf("turn %d tidak ditandai sebagai turn kalibrasi", turn)
		}
	}

	// ...tetapi ditandai bukan berarti revisinya boleh dibuang. Saat
	// diarization mengoreksi turn 2 menjadi "B", itu satu-satunya sinyal
	// bahwa ada dua suara.
	role, _ := stt.resolveSpeaker("sesi-1", "B", 2)
	if role != domain.SpeakerUnknown {
		t.Fatalf("selama kalibrasi semua role harus unknown, dapat %q", role)
	}
}
