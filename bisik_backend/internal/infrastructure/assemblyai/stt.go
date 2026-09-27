package assemblyai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/usecase"
	"github.com/gorilla/websocket"
)

// StreamingSTT adalah klien WebSocket ke AssemblyAI Universal Streaming v3
// dengan streaming speaker diarization aktif.
//
// Protokol: JSON biasa di atas WebSocket, tanpa SDK proprietary — Go bicara
// langsung ke endpoint. Referensi: docs/streaming/api-spec/streaming-websocket.
//
// Pesan server yang kita tangani:
//
//	Begin           {type, id, expires_at, configuration}
//	Turn            {type, turn_order, end_of_turn, transcript, speaker_label, words[]}
//	SpeakerRevision {type, revisions:[{turn_order, speaker_label, words[]}]}
//	Termination     {type, audio_duration_seconds, session_duration_seconds}
type StreamingSTT struct {
	apiKey             string
	wsURL              string
	model              string
	revisionIntervalMS int
	log                *slog.Logger

	mu    sync.RWMutex
	conns map[string]*streamConnection
	// speakers menyimpan pemetaan dan turn kalibrasi per sesi.
	speakers map[string]*speakerSession
}

type speakerSession struct {
	roles            map[string]domain.Speaker
	calibrating      bool
	calibrationTurns map[int]struct{}
	// confirmed menandai manusia sudah mengunci mapping role. Setelah ini,
	// label baru berarti pembicara ketiga — bukan role yang belum terisi.
	confirmed bool
}

type streamConnection struct {
	conn    *websocket.Conn
	done    chan struct{}
	writeMu sync.Mutex
	once    sync.Once

	// audioMu melindungi buffer akumulasi frame audio. Dipisahkan dari
	// writeMu supaya klien yang mendorong audio tidak menghalangi Terminate.
	audioMu  sync.Mutex
	audioBuf []byte

	// terminating ditandai saat Stop dipanggil, supaya penutupan yang
	// disengaja tidak dilaporkan sebagai kegagalan ke petugas.
	terminating atomic.Bool
}

func (c *streamConnection) write(messageType int, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(messageType, payload)
}

func NewStreamingSTT(apiKey, wsURL, model string, revisionIntervalMS int, log *slog.Logger) *StreamingSTT {
	if model == "" {
		model = DefaultSpeechModel
	}
	// Nilai TIDAK lagi dipaksa naik ke minimum dokumentasi.
	//
	// Clamp diam-diam sebelumnya membuat nilai lebih kecil mustahil diuji,
	// padahal interval revisi 120 detik adalah tersangka utama kenapa suara
	// kedua tidak pernah muncul saat kalibrasi: sesi kalibrasi hanya belasan
	// detik, jauh sebelum revisi pertama dijadwalkan. Biarkan server yang
	// memutuskan dan menaikkan kalau memang perlu — konfigurasi efektifnya
	// tercetak dari pesan `Begin`.
	if revisionIntervalMS < 1000 {
		revisionIntervalMS = MinimumRevisionIntervalMS
	}
	if revisionIntervalMS < MinimumRevisionIntervalMS {
		log.Warn("interval revisi di bawah minimum dokumentasi; server boleh menaikkannya",
			"diminta_ms", revisionIntervalMS, "minimum_dokumentasi_ms", MinimumRevisionIntervalMS)
	}
	return &StreamingSTT{
		apiKey:             apiKey,
		wsURL:              wsURL,
		model:              model,
		revisionIntervalMS: revisionIntervalMS,
		log:                log,
		conns:              make(map[string]*streamConnection),
		speakers:           make(map[string]*speakerSession),
	}
}

// --- Bentuk pesan upstream ---

type wsWord struct {
	Text        string `json:"text"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Speaker     string `json:"speaker"`
	WordIsFinal bool   `json:"word_is_final"`
}

type wsRevision struct {
	TurnOrder    int      `json:"turn_order"`
	SpeakerLabel string   `json:"speaker_label"`
	Words        []wsWord `json:"words"`
}

type wsMessage struct {
	Type string `json:"type"`

	// Begin
	ID string `json:"id"`

	// Turn
	TurnOrder    int      `json:"turn_order"`
	EndOfTurn    bool     `json:"end_of_turn"`
	Transcript   string   `json:"transcript"`
	SpeakerLabel string   `json:"speaker_label"`
	Words        []wsWord `json:"words"`

	// SpeakerRevision
	Revisions []wsRevision `json:"revisions"`
}

func (s *StreamingSTT) Start(ctx context.Context, sessionID string) (<-chan usecase.TranscriptEvent, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("%w: ASSEMBLYAI_API_KEY kosong", domain.ErrUpstreamAudio)
	}

	q := s.connectionParams()

	endpoint := s.wsURL + "?" + q.Encode()
	header := http.Header{"Authorization": []string{s.apiKey}} // tanpa prefix "Bearer"

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, endpoint, header)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrUpstreamAudio, err)
	}

	stream := &streamConnection{conn: conn, done: make(chan struct{})}
	s.mu.Lock()
	s.conns[sessionID] = stream
	s.speakers[sessionID] = &speakerSession{
		roles:            make(map[string]domain.Speaker),
		calibrationTurns: make(map[int]struct{}),
	}
	s.mu.Unlock()

	out := make(chan usecase.TranscriptEvent, 64)
	go s.readLoop(ctx, sessionID, stream, out)
	return out, nil
}

// Konstanta protokol.
const (
	SampleRate         = 16000
	DefaultSpeechModel = "whisper-rt"
	// Frame 4096 byte adalah ukuran yang direkomendasikan AssemblyAI
	// (128 ms pada PCM16 16 kHz mono).
	RecommendedChunkBytes = 4096
	// Streaming API hanya menerima frame 50–1000 ms. Pada PCM16 16 kHz mono
	// satu milidetik = 32 byte, jadi batasnya 1600–32000 byte.
	MinimumFrameBytes         = 1600
	MaximumFrameBytes         = 32000
	MinimumRevisionIntervalMS = 120000
	// MaxSpeakers 3 = dua role terkalibrasi + satu slot penampung orang ketiga.
	MaxSpeakers = 3
)

// connectionParams dipisahkan agar kontrak koneksi dapat diuji tanpa
// membuka WebSocket sungguhan. Universal-3.5 Pro selalu memformat hasil,
// sedangkan keluarga Universal Streaming memerlukan format_turns=true.
func (s *StreamingSTT) connectionParams() url.Values {
	q := url.Values{}
	q.Set("speech_model", s.model)
	q.Set("encoding", "pcm_s16le")
	q.Set("sample_rate", strconv.Itoa(SampleRate))
	if s.model == "universal-streaming-english" || s.model == "universal-streaming-multilingual" {
		q.Set("format_turns", "true")
	}
	// Diarization. Batas sengaja 3, bukan 2: percakapannya memang dua orang,
	// tetapi kalau ada orang ketiga yang menyela, batas 2 memaksa suaranya
	// dijejalkan ke salah satu role terkalibrasi — dan ucapan orang asing
	// bisa terhitung sebagai kepatuhan petugas. Dengan batas 3 dia mendapat
	// label sendiri, lalu kita tolak sebagai pembicara tidak dikenal.
	q.Set("speaker_labels", "true")
	q.Set("max_speakers", strconv.Itoa(MaxSpeakers))
	// Dokumentasi menetapkan minimum 120 detik. Nilainya tetap dikirim apa
	// adanya supaya bisa diuji; kalau server menaikkannya, itu akan terlihat
	// pada konfigurasi efektif di pesan `Begin`. Revisi final tetap datang
	// saat Terminate.
	q.Set("speaker_labels_revision_interval_ms", strconv.Itoa(s.revisionIntervalMS))
	return q
}

func (s *StreamingSTT) readLoop(ctx context.Context, sessionID string, stream *streamConnection, out chan<- usecase.TranscriptEvent) {
	defer close(out)
	defer s.cleanup(sessionID, stream)

	turnsSeen, turnsWithoutLabel := 0, 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, raw, err := stream.conn.ReadMessage()
		if err != nil {
			s.log.Info("stt stream selesai", "session", sessionID, "err", err)
			if !stream.terminating.Load() {
				// Upstream memutus duluan — mis. frame audio ditolak. Sesi
				// tidak bisa dilanjutkan, jadi petugas harus diberi tahu.
				emitTranscriptEvent(ctx, out, usecase.TranscriptEvent{
					Err: fmt.Errorf("%w: %v", domain.ErrUpstreamAudio, err),
				})
			}
			return
		}

		var m wsMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}

		switch m.Type {
		case "Begin":
			// Konfigurasi efektif dicetak apa adanya. Server boleh MENERIMA
			// parameter `speaker_labels` lalu diam-diam tidak mengaktifkannya
			// untuk model tertentu — dan tanpa cetakan ini, satu-satunya
			// gejala adalah label pembicara yang kosong terus.
			s.log.Info("stt terhubung", "session", sessionID, "upstream_id", m.ID,
				"konfigurasi_efektif", string(raw))

		case "Turn":
			if m.Transcript == "" {
				continue
			}
			// Tiga turn pertama dicetak mentah. Ini cara tercepat memastikan
			// nama field dan apakah `speaker_label`/`words[].speaker` benar
			// benar dikirim server, alih-alih menebak dari gejala.
			if turnsSeen < 3 {
				turnsSeen++
				s.log.Info("turn mentah", "session", sessionID, "ke", turnsSeen,
					"payload", string(raw))
			}
			if usableLabel(m.SpeakerLabel) == "" {
				turnsWithoutLabel++
				if labelFromWords(m.Words) == "" {
					s.log.Warn("turn tanpa label pembicara",
						"session", sessionID,
						"turn", m.TurnOrder,
						"tanpa_label", turnsWithoutLabel,
						"speaker_label", m.SpeakerLabel,
						"jumlah_kata", len(m.Words),
						"kesimpulan", "diarization tidak menghasilkan label untuk model ini")
				}
			}
			start, end := spanOf(m.Words)
			label := usableLabel(m.SpeakerLabel)
			if label == "" {
				label = labelFromWords(m.Words)
			}
			role, calibrating := s.resolveSpeaker(sessionID, label, m.TurnOrder)
			if !emitTranscriptEvent(ctx, out, usecase.TranscriptEvent{
				UtteranceID:   utteranceID(sessionID, m.TurnOrder),
				Speaker:       role,
				SourceSpeaker: label,
				Text:          m.Transcript,
				StartMS:       start,
				EndMS:         end,
				IsFinal:       m.EndOfTurn,
				IsCalibration: calibrating,
			}) {
				return
			}

		// Model menyusun ulang profil pembicara dan mengoreksi label turn
		// sebelumnya. Hanya turn yang berubah yang dikirim.
		case "SpeakerRevision":
			for _, rev := range m.Revisions {
				label := usableLabel(rev.SpeakerLabel)
				if label == "" {
					label = labelFromWords(rev.Words)
				}

				// Revisi untuk turn kalibrasi dulu DIBUANG di sini, dan itu
				// membuat suara kedua tidak pernah muncul.
				//
				// Streaming diarization membangun profil suara seiring
				// bertambahnya audio. Di detik-detik awal sesi ia sering
				// melabeli dua orang dengan label yang sama, lalu
				// mengoreksinya lewat SpeakerRevision. Kalibrasi terjadi
				// persis di detik-detik awal itu — jadi revisi inilah
				// satu-satunya cara sistem tahu ada dua suara.
				// Rentang waktu WAJIB ikut. Penggabung dua stream menjodohkan
				// teks dengan label berdasarkan waktu; revisi tanpa rentang
				// akan dibuang diam-diam dan labelnya tidak pernah sampai.
				revStart, revEnd := spanOf(rev.Words)

				if s.isCalibrationTurn(sessionID, rev.TurnOrder) {
					if !emitTranscriptEvent(ctx, out, usecase.TranscriptEvent{
						UtteranceID:   utteranceID(sessionID, rev.TurnOrder),
						Speaker:       domain.SpeakerUnknown,
						SourceSpeaker: label,
						Text:          textFromWords(rev.Words),
						StartMS:       revStart,
						EndMS:         revEnd,
						IsFinal:       true,
						IsCalibration: true,
					}) {
						return
					}
					continue
				}

				role, _ := s.resolveSpeaker(sessionID, label, rev.TurnOrder)
				if !emitTranscriptEvent(ctx, out, usecase.TranscriptEvent{
					UtteranceID:   utteranceID(sessionID, rev.TurnOrder),
					Speaker:       role,
					SourceSpeaker: label,
					StartMS:       revStart,
					EndMS:         revEnd,
					IsFinal:       true,
					IsRevision:    true,
				}) {
					return
				}
			}

		case "Termination":
			s.log.Info("stt diakhiri upstream", "session", sessionID)
			return
		}
	}
}

// emitTranscriptEvent menunggu adapter mengonfirmasi bahwa persistence dan
// compliance evaluation selesai. Ini membuat Stop() menjadi flush barrier.
func emitTranscriptEvent(ctx context.Context, out chan<- usecase.TranscriptEvent, event usecase.TranscriptEvent) bool {
	ack := make(chan struct{})
	var once sync.Once
	event.Acknowledge = func() { once.Do(func() { close(ack) }) }

	select {
	case out <- event:
	case <-ctx.Done():
		return false
	}

	select {
	case <-ack:
		return true
	case <-ctx.Done():
		return false
	}
}

// utteranceID stabil per (sesi, turn) supaya revisi bisa menunjuk balik
// ke baris transkrip yang sama.
func utteranceID(sessionID string, turnOrder int) string {
	return fmt.Sprintf("%s-t%d", sessionID, turnOrder)
}

// labelFromWords menurunkan label pembicara dari words[] ketika label
// tingkat turn kosong.
//
// AssemblyAI tidak selalu mengisi `speaker_label` di level turn, tetapi
// hampir selalu mengisi `speaker` per kata. Tanpa cadangan ini, turn yang
// labelnya kosong akan hilang begitu saja dari kalibrasi — dan petugas
// melihat layar yang diam tanpa penjelasan.
func labelFromWords(words []wsWord) string {
	counts := map[string]int{}
	for _, w := range words {
		if usableLabel(w.Speaker) != "" {
			counts[w.Speaker]++
		}
	}
	best, bestCount := "", 0
	for label, n := range counts {
		if n > bestCount {
			best, bestCount = label, n
		}
	}
	return best
}

// textFromWords menyusun ulang kalimat dari words[]. Pesan SpeakerRevision
// tidak membawa `transcript`, hanya daftar kata.
func textFromWords(words []wsWord) string {
	parts := make([]string, 0, len(words))
	for _, w := range words {
		if w.Text != "" {
			parts = append(parts, w.Text)
		}
	}
	return strings.Join(parts, " ")
}

func spanOf(words []wsWord) (int, int) {
	if len(words) == 0 {
		return 0, 0
	}
	return words[0].Start, words[len(words)-1].End
}

// resolveSpeaker memetakan label diarization ke role. Dalam mode kalibrasi,
// label belum dipercaya: semua ucapan menjadi unknown dan tidak dinilai.
// usableLabel menyaring label yang BELUM merupakan identitas pembicara.
//
// `UNKNOWN` dipakai model untuk ucapan terlalu pendek. `PENDING` dipakai
// Universal Streaming saat ia belum memutuskan, lalu dikoreksi lewat
// SpeakerRevision. Keduanya harus diperlakukan sebagai "belum tahu" —
// membiarkan `PENDING` lolos akan memunculkan kartu suara hantu di kalibrasi
// dan, lebih buruk, memberinya role.
func usableLabel(label string) string {
	switch label {
	case "", "UNKNOWN", "PENDING":
		return ""
	}
	return label
}

// ResolveLabelRole memetakan label mentah ke role TANPA menandai turn
// kalibrasi. Dipakai penggabung dua stream, yang nomor turn-nya berasal dari
// koneksi lain sehingga tidak boleh ikut menandai turn di sini.
func (s *StreamingSTT) ResolveLabelRole(sessionID, label string) (domain.Speaker, bool) {
	if usableLabel(label) == "" {
		return domain.SpeakerUnknown, s.isCalibrating(sessionID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.speakers[sessionID]
	if !ok {
		return domain.SpeakerUnknown, false
	}
	if state.calibrating {
		return domain.SpeakerUnknown, true
	}
	if role, ok := state.roles[label]; ok {
		return role, false
	}
	if state.confirmed {
		return domain.SpeakerUnknown, false
	}
	if len(state.roles) == 0 {
		state.roles[label] = domain.SpeakerOfficer
	} else {
		state.roles[label] = domain.SpeakerCustomer
	}
	return state.roles[label], false
}

func (s *StreamingSTT) resolveSpeaker(sessionID, label string, turnOrder int) (domain.Speaker, bool) {
	if usableLabel(label) == "" {
		return domain.SpeakerUnknown, s.isCalibrating(sessionID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.speakers[sessionID]
	if !ok {
		return domain.SpeakerUnknown, false
	}
	if state.calibrating {
		state.calibrationTurns[turnOrder] = struct{}{}
		return domain.SpeakerUnknown, true
	}
	if role, ok := state.roles[label]; ok {
		return role, false
	}
	// Mapping sudah dikunci manusia, tetapi muncul label di luar keduanya.
	// Itu pembicara ketiga/pengamat. Dia tidak boleh mewarisi role siapa pun:
	// dianggap nasabah pun salah, karena transkrip jadi berbohong soal siapa
	// yang bicara. Kembalikan unknown supaya scoring melewatinya.
	if state.confirmed {
		return domain.SpeakerUnknown, false
	}
	// Fallback untuk klien lama yang belum menjalankan kalibrasi.
	if len(state.roles) == 0 {
		state.roles[label] = domain.SpeakerOfficer
	} else {
		state.roles[label] = domain.SpeakerCustomer
	}
	return state.roles[label], false
}

func (s *StreamingSTT) isCalibrating(sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := s.speakers[sessionID]
	return state != nil && state.calibrating
}

// SpeakerCalibrationActive memungkinkan adapter menahan pengingat berkala
// agar petugas tidak dibisiki sebelum role dua pembicara terkonfirmasi.
func (s *StreamingSTT) SpeakerCalibrationActive(sessionID string) bool {
	return s.isCalibrating(sessionID)
}

func (s *StreamingSTT) isCalibrationTurn(sessionID string, turnOrder int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := s.speakers[sessionID]
	if state == nil {
		return false
	}
	_, ok := state.calibrationTurns[turnOrder]
	return ok
}

// BeginSpeakerCalibration menahan scoring sampai dua label dikonfirmasi.
func (s *StreamingSTT) BeginSpeakerCalibration(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.speakers[sessionID]
	if !ok {
		return domain.ErrNotFound
	}
	state.roles = make(map[string]domain.Speaker)
	state.calibrationTurns = make(map[int]struct{})
	state.calibrating = true
	state.confirmed = false
	return nil
}

// ConfirmSpeakerRoles mengikat label mentah A/B ke role domain secara eksplisit.
func (s *StreamingSTT) ConfirmSpeakerRoles(sessionID, officerLabel, customerLabel string) error {
	if officerLabel == "" || customerLabel == "" || officerLabel == customerLabel ||
		officerLabel == "UNKNOWN" || customerLabel == "UNKNOWN" {
		return domain.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.speakers[sessionID]
	if !ok {
		return domain.ErrNotFound
	}
	state.roles = map[string]domain.Speaker{
		officerLabel:  domain.SpeakerOfficer,
		customerLabel: domain.SpeakerCustomer,
	}
	state.calibrating = false
	state.confirmed = true
	return nil
}

// PushAudio mengakumulasi potongan PCM16 dari klien sampai memenuhi durasi
// minimum AssemblyAI sebelum dikirim ke upstream.
//
// Ini jaring pengaman untuk SEMUA klien, bukan hanya PWA. AudioWorklet web
// mengeluarkan 128 sample (8 ms) per render quantum dan `record` di Flutter
// memberi potongan dengan ukuran yang tidak dijamin. Mengirim frame di bawah
// 50 ms membuat AssemblyAI menutup koneksi, sehingga sesi berikutnya gagal
// dengan "resource tidak ditemukan".
func (s *StreamingSTT) PushAudio(sessionID string, pcm []byte) error {
	s.mu.RLock()
	stream, ok := s.conns[sessionID]
	s.mu.RUnlock()
	if !ok {
		return domain.ErrNotFound
	}

	frames := stream.appendAudio(pcm)
	for _, frame := range frames {
		if err := stream.write(websocket.BinaryMessage, frame); err != nil {
			return err
		}
	}
	return nil
}

// appendAudio menumpuk potongan baru dan mengembalikan frame yang sudah
// cukup besar untuk dikirim. Sisa di bawah batas disimpan untuk panggilan
// berikutnya, dan frame besar dipotong agar tidak melewati batas atas.
func (c *streamConnection) appendAudio(pcm []byte) [][]byte {
	c.audioMu.Lock()
	defer c.audioMu.Unlock()

	c.audioBuf = append(c.audioBuf, pcm...)

	var frames [][]byte
	for len(c.audioBuf) >= MaximumFrameBytes {
		frames = append(frames, append([]byte(nil), c.audioBuf[:MaximumFrameBytes]...))
		c.audioBuf = c.audioBuf[MaximumFrameBytes:]
	}
	if len(c.audioBuf) >= RecommendedChunkBytes {
		frames = append(frames, append([]byte(nil), c.audioBuf...))
		c.audioBuf = c.audioBuf[:0]
	}
	return frames
}

// drainAudio mengembalikan sisa buffer kalau masih memenuhi durasi minimum.
// Sisa yang lebih pendek dari 50 ms sengaja dibuang: AssemblyAI menolaknya,
// dan potongan sependek itu tidak mengubah hasil transkrip.
func (c *streamConnection) drainAudio() []byte {
	c.audioMu.Lock()
	defer c.audioMu.Unlock()

	rest := c.audioBuf
	c.audioBuf = nil
	if len(rest) < MinimumFrameBytes {
		return nil
	}
	return rest
}

func (s *StreamingSTT) Stop(sessionID string) error {
	s.mu.RLock()
	stream, ok := s.conns[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	stream.terminating.Store(true)

	// Kirim sisa audio yang masih tertahan di buffer sebelum menutup sesi,
	// supaya kalimat terakhir tetap ikut dinilai.
	if rest := stream.drainAudio(); rest != nil {
		if err := stream.write(websocket.BinaryMessage, rest); err != nil {
			s.log.Warn("flush audio terakhir gagal", "session", sessionID, "err", err)
		}
	}

	if err := stream.write(websocket.TextMessage, []byte(`{"type":"Terminate"}`)); err != nil {
		_ = stream.conn.Close()
		return err
	}

	select {
	case <-stream.done:
		return nil
	case <-time.After(5 * time.Second):
		_ = stream.conn.Close()
		return fmt.Errorf("%w: timeout menunggu Termination", domain.ErrUpstreamAudio)
	}
}

func (s *StreamingSTT) cleanup(sessionID string, stream *streamConnection) {
	stream.once.Do(func() {
		s.mu.Lock()
		if current, ok := s.conns[sessionID]; ok && current == stream {
			delete(s.conns, sessionID)
			delete(s.speakers, sessionID)
		}
		s.mu.Unlock()
		_ = stream.conn.Close()
		close(stream.done)
	})
}
