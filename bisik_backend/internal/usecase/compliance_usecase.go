package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/google/uuid"
)

const minimumMatchConfidence = 0.80

var digitPattern = regexp.MustCompile(`\d`)

var numberWords = []string{
	"nol", "satu", "dua", "tiga", "empat", "lima", "enam", "tujuh", "delapan", "sembilan",
	"sepuluh", "sebelas", "belas", "puluh", "ratus", "ribu", "juta", "miliar",
}

// ComplianceUsecase adalah inti Bisik: menilai tiap ucapan petugas
// terhadap 5 butir kewajiban, dan membisikkan yang belum terpenuhi.
type ComplianceUsecase struct {
	compliance  domain.ComplianceRepository
	transcripts domain.TranscriptRepository
	matcher     SemanticMatcher
	guard       Guardrail
	nudger      Nudger
	broadcaster Broadcaster
	obligations []domain.Obligation

	// degraded menyimpan alasan audio sedang tidak layak per sesi.
	// String kosong berarti audio sehat.
	mu       sync.RWMutex
	degraded map[string]string
	// nudges menyimpan bisikan yang baru diucapkan per sesi, untuk mengenali
	// gemanya sendiri. Lihat isNudgeEcho.
	nudges map[string][]spokenNudge
	// scoringDown menandai sesi yang sudah diberi tahu bahwa penilai
	// semantik gagal, supaya layar petugas tidak dibanjiri pesan yang sama.
	scoringDown map[string]bool
	// scorerProblem diisi saat uji akses penilai gagal ketika startup.
	scorerProblem string

	// fragments menyimpan potongan kalimat petugas yang baru saja diucapkan.
	// Lihat stitchOfficerSentence.
	fragments map[string][]officerFragment
	// reminders mencatat kapan dan berapa kali tiap butir dibisikkan.
	reminders map[string]*reminderLog

	log *slog.Logger
	now func() time.Time
}

type officerFragment struct {
	text           string
	startMS, endMS int
}

type reminderLog struct {
	last  time.Time
	count map[string]int
}

const (
	// Jeda terpanjang di TENGAH kalimat. whisper-rt memotong turn pada jeda
	// sependek ~1 detik dan mengabaikan parameter end-of-turn (diuji
	// 28 Sep 2026: min_end_of_turn_silence_when_confident, max_turn_silence,
	// end_of_turn_confidence_threshold tidak mengubah potongan).
	fragmentJoinGapMS = 2000
	// Batas panjang satu kalimat gabungan, supaya dua kalimat berbeda yang
	// kebetulan berdekatan tidak dijahit menjadi satu bukti.
	fragmentMaxSpanMS = 15000
	fragmentMaxCount  = 4

	// Bisikan pengingat: jarak minimal antar-bisikan dalam satu sesi, dan
	// batas pengulangan per butir. Dulu butir pertama yang pending dibisikkan
	// setiap 45 detik tanpa batas — di uji lapangan itu terdengar sebagai spam.
	reminderSpacing           = 90 * time.Second
	maxRemindersPerObligation = 2
)

// SetScorerProblem mencatat bahwa penilai semantik sudah diketahui tidak bisa
// dipakai sejak startup. Pengingat berhenti — percuma menyuruh petugas
// mengucapkan butir yang mustahil dinilai — dan layar diberi tahu.
func (uc *ComplianceUsecase) SetScorerProblem(problem string) { uc.scorerProblem = problem }

// SetLogger memasang logger. Opsional: tanpa logger, kegagalan penilai
// tetap disiarkan ke layar, hanya tidak tercatat di log server.
func (uc *ComplianceUsecase) SetLogger(l *slog.Logger) { uc.log = l }

type spokenNudge struct {
	words map[string]struct{}
	at    time.Time
}

// Bisikan yang lebih tua dari ini tidak mungkin lagi menjadi gema; TTS
// diputar hampir seketika setelah dikirim.
const nudgeEchoWindow = 20 * time.Second

func NewComplianceUsecase(
	c domain.ComplianceRepository,
	t domain.TranscriptRepository,
	m SemanticMatcher,
	g Guardrail,
	n Nudger,
	b Broadcaster,
) *ComplianceUsecase {
	return &ComplianceUsecase{
		compliance:  c,
		transcripts: t,
		matcher:     m,
		guard:       g,
		nudger:      n,
		broadcaster: b,
		obligations: domain.DefaultObligations(),
		degraded:    make(map[string]string),
		nudges:      make(map[string][]spokenNudge),
		scoringDown: make(map[string]bool),
		fragments:   make(map[string][]officerFragment),
		reminders:   make(map[string]*reminderLog),
		now:         time.Now,
	}
}

// HandleTranscript dipanggil untuk setiap event dari STT.
func (uc *ComplianceUsecase) HandleTranscript(ctx context.Context, sessionID string, ev TranscriptEvent) error {
	// Kalibrasi hanya membantu manusia mengenali label A/B. Ucapannya sengaja
	// tidak masuk database, guardrail, checklist, atau laporan akhir.
	if ev.IsCalibration {
		eventType := "speaker_calibration_partial"
		if ev.IsFinal {
			eventType = "speaker_calibration_utterance"
		}
		// utterance_id ikut dikirim supaya klien bisa MEMPERBARUI contoh yang
		// sama ketika diarization mengoreksi labelnya, bukan menambah kartu
		// baru. Tanpa ini, koreksi label saat kalibrasi tidak bisa dipetakan
		// ke contoh mana pun.
		uc.broadcaster.Publish(sessionID, map[string]any{
			"type": eventType, "utterance_id": ev.UtteranceID,
			"source_speaker": ev.SourceSpeaker, "text": ev.Text,
		})
		return nil
	}

	// Revisi label diarization: perbarui label lama, jangan buat entri baru.
	if ev.IsRevision {
		return uc.handleRevision(ctx, sessionID, ev)
	}
	if !ev.IsFinal {
		uc.broadcaster.Publish(sessionID, map[string]any{
			"type": "partial", "speaker": ev.Speaker, "text": ev.Text,
		})
		return nil
	}

	u := &domain.Utterance{
		ID:            ev.UtteranceID,
		SessionID:     sessionID,
		Speaker:       ev.Speaker,
		SourceSpeaker: ev.SourceSpeaker,
		Text:          ev.Text,
		StartMS:       ev.StartMS,
		EndMS:         ev.EndMS,
		CreatedAt:     time.Now().UTC(),
	}
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	if err := uc.transcripts.Append(ctx, u); err != nil {
		return err
	}
	// source_speaker ikut dikirim supaya label mentah diarization terlihat di
	// UI. Tanpa itu, "semua Petugas" tidak bisa dibedakan antara satu suara
	// yang memang bicara sendiri dan diarization yang gagal memisahkan.
	uc.broadcaster.Publish(sessionID, map[string]any{
		"type": "utterance", "id": u.ID, "speaker": u.Speaker,
		"source_speaker": u.SourceSpeaker, "text": u.Text,
	})

	// Pembicara tidak dikenal: label di luar dua hasil kalibrasi, atau ucapan
	// terlalu pendek sehingga diarization menyerah. Ucapannya tetap masuk
	// transkrip sebagai jejak, tapi petugas harus tahu bagian ini tidak
	// dihitung — supaya tidak mengira kewajibannya sudah tersampaikan.
	if u.Speaker == domain.SpeakerUnknown {
		uc.broadcaster.Publish(sessionID, map[string]any{
			"type": "speaker_unknown", "utterance_id": u.ID, "text": u.Text,
		})
		return nil
	}

	// Butir kewajiban hanya bisa dipenuhi oleh PETUGAS, bukan nasabah.
	if u.Speaker != domain.SpeakerOfficer {
		// Nasabah menyela: kalimat petugas sebelumnya sudah selesai, jangan
		// dijahit dengan kalimat petugas berikutnya.
		uc.resetFragments(sessionID)
		return nil
	}

	// Bisikan aplikasi bisa terekam kembali kalau petugas memakai speaker
	// alih-alih earphone. Kalau dibiarkan, sistem menilai suaranya sendiri:
	// bisikan "Belum disampaikan: Denda keterlambatan" memuat kata "denda"
	// dan "keterlambatan", sehingga evidence gate PENALTY akan cocok dan
	// butir itu berubah hijau tanpa petugas pernah mengucapkannya. Untuk
	// pelanggaran, gemanya bahkan akan tercatat sebagai pelanggaran kedua.
	if uc.isNudgeEcho(sessionID, u.Text) {
		uc.broadcaster.Publish(sessionID, map[string]any{
			"type": "evidence_skipped", "utterance_id": u.ID,
			"reason": "terdengar seperti gema bisikan aplikasi — pakai earphone",
		})
		return nil
	}

	// Guardrail tetap jalan walau audio buruk: frasa terlarang yang sempat
	// tertranskrip lebih baik ditandai daripada dilewatkan diam-diam.
	uc.inspectGuardrail(ctx, sessionID, u)

	// Checklist TIDAK boleh terpenuhi dari audio yang tidak layak. Ini arah
	// gagal yang aman: menunda centang hijau hanya merepotkan, sedangkan
	// centang hijau palsu membuat laporan kepatuhan berbohong.
	if reason := uc.degradedReason(sessionID); reason != "" {
		uc.broadcaster.Publish(sessionID, map[string]any{
			"type": "evidence_skipped", "utterance_id": u.ID, "reason": reason,
		})
		return nil
	}
	return uc.evaluateObligations(ctx, sessionID, u.ID, uc.stitchOfficerSentence(sessionID, u))
}

// stitchOfficerSentence menjahit potongan kalimat petugas yang terpotong di
// jeda singkat, lalu mengembalikan teks gabungannya untuk dinilai.
//
// Ditemukan dari uji lapangan 28 Sep 2026: "perkenalkan nama saya handoko"
// dan "dari bank nusantara" datang sebagai dua turn berjeda 1,5 detik.
// Evidence gate IDENTITY butuh "nama saya" DAN "bank" dalam satu teks, jadi
// kewajiban yang sudah diucapkan lengkap tidak pernah terpenuhi — dan
// pengingatnya terus berbunyi.
//
// Hanya potongan berlabel PETUGAS yang dijahit, dan rantainya putus begitu
// nasabah bicara. Ucapan nasabah tidak pernah bisa menjadi bagian bukti.
// Bukti tetap menunjuk potongan terakhir — yang melengkapi kalimatnya.
func (uc *ComplianceUsecase) stitchOfficerSentence(sessionID string, u *domain.Utterance) string {
	current := officerFragment{text: strings.TrimSpace(u.Text), startMS: u.StartMS, endMS: u.EndMS}
	uc.mu.Lock()
	defer uc.mu.Unlock()

	chain := uc.fragments[sessionID]
	if n := len(chain); n > 0 {
		prev := chain[n-1]
		joinable := current.startMS > 0 &&
			current.startMS >= prev.endMS &&
			current.startMS-prev.endMS <= fragmentJoinGapMS &&
			current.endMS-chain[0].startMS <= fragmentMaxSpanMS &&
			n < fragmentMaxCount
		if !joinable {
			chain = nil
		}
	}
	chain = append(chain, current)
	uc.fragments[sessionID] = chain

	parts := make([]string, len(chain))
	for i, f := range chain {
		parts[i] = f.text
	}
	return strings.Join(parts, " ")
}

func (uc *ComplianceUsecase) resetFragments(sessionID string) {
	uc.mu.Lock()
	delete(uc.fragments, sessionID)
	uc.mu.Unlock()
}

// SetAudioQuality dipanggil adapter saat klien melaporkan kualitas audio.
// Alasan kosong berarti audio kembali sehat.
func (uc *ComplianceUsecase) SetAudioQuality(sessionID, reason string) {
	uc.mu.Lock()
	previous := uc.degraded[sessionID]
	if reason == "" {
		delete(uc.degraded, sessionID)
	} else {
		uc.degraded[sessionID] = reason
	}
	uc.mu.Unlock()

	if previous == reason {
		return // jangan membanjiri UI dengan status yang sama
	}
	uc.broadcaster.Publish(sessionID, map[string]any{
		"type": "audio_quality", "degraded": reason != "", "reason": reason,
	})
}

// rememberNudge mencatat bisikan yang baru diucapkan.
func (uc *ComplianceUsecase) rememberNudge(sessionID, text string) {
	words := map[string]struct{}{}
	for _, w := range strings.Fields(normalizeForEcho(text)) {
		words[w] = struct{}{}
	}
	if len(words) == 0 {
		return
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()
	kept := uc.nudges[sessionID][:0]
	for _, n := range uc.nudges[sessionID] {
		if time.Since(n.at) < nudgeEchoWindow {
			kept = append(kept, n)
		}
	}
	uc.nudges[sessionID] = append(kept, spokenNudge{words: words, at: time.Now()})
}

// isNudgeEcho menebak apakah sebuah ucapan sebenarnya bisikan aplikasi yang
// terekam balik.
//
// Perbandingannya per kata, bukan persis, karena STT jarang mentranskrip
// suara TTS dengan sempurna. Ambang 0,6 dipilih longgar ke arah aman:
// melewatkan satu ucapan sah hanya menunda satu butir, sedangkan meloloskan
// gema membuat laporan mengklaim sesuatu yang tidak pernah diucapkan.
func (uc *ComplianceUsecase) isNudgeEcho(sessionID, text string) bool {
	fields := strings.Fields(normalizeForEcho(text))
	if len(fields) < 2 {
		return false
	}

	uc.mu.RLock()
	defer uc.mu.RUnlock()
	for _, n := range uc.nudges[sessionID] {
		if time.Since(n.at) >= nudgeEchoWindow {
			continue
		}
		matched := 0
		for _, w := range fields {
			if _, ok := n.words[w]; ok {
				matched++
			}
		}
		if float64(matched)/float64(len(fields)) >= 0.6 {
			return true
		}
	}
	return false
}

func normalizeForEcho(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		if r == ' ' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

func (uc *ComplianceUsecase) degradedReason(sessionID string) string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	return uc.degraded[sessionID]
}

// handleRevision menangani koreksi label pembicara dari AssemblyAI.
//
// Ini bukan sekadar mengganti label di UI. Kalau satu ucapan ternyata
// milik PETUGAS padahal tadi dikira nasabah, ucapan itu belum pernah
// dinilai terhadap checklist — jadi harus dinilai sekarang. Tanpa ini,
// koreksi diarization justru membuat laporan kepatuhan salah.
func (uc *ComplianceUsecase) handleRevision(ctx context.Context, sessionID string, ev TranscriptEvent) error {
	if err := uc.transcripts.ReviseSpeaker(ctx, ev.UtteranceID, ev.Speaker); err != nil {
		return err
	}
	uc.broadcaster.Publish(sessionID, map[string]any{
		"type": "speaker_revised", "utterance_id": ev.UtteranceID, "speaker": ev.Speaker,
	})

	if ev.Speaker != domain.SpeakerOfficer {
		return nil
	}
	if reason := uc.degradedReason(sessionID); reason != "" {
		return nil
	}
	u, err := uc.transcripts.FindByID(ctx, ev.UtteranceID)
	if err != nil {
		// Revisi bisa menunjuk turn yang belum sempat tersimpan; abaikan.
		return nil
	}
	uc.inspectGuardrail(ctx, sessionID, u)
	return uc.evaluateObligations(ctx, sessionID, u.ID, u.Text)
}

func (uc *ComplianceUsecase) inspectGuardrail(ctx context.Context, sessionID string, u *domain.Utterance) {
	phrase, severity, found, err := uc.guard.Inspect(ctx, u.Text)
	if err != nil || !found {
		return
	}
	v := &domain.Violation{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		Phrase:     phrase,
		Severity:   severity,
		EvidenceID: u.ID,
		DetectedAt: time.Now().UTC(),
	}
	_ = uc.compliance.RecordViolation(ctx, v)
	uc.broadcaster.Publish(sessionID, map[string]any{
		"type": "violation", "phrase": phrase, "severity": severity, "evidence_id": u.ID,
	})
	warning := fmt.Sprintf("Hati-hati, hindari frasa %q.", phrase)
	uc.rememberNudge(sessionID, warning)
	_ = uc.nudger.Whisper(ctx, sessionID, Nudge{Text: warning, Kind: NudgeAvoidPhrase, Phrase: phrase})
}

// evaluateObligations menilai text terhadap butir yang masih pending.
// evidenceID adalah ucapan yang dicatat sebagai buktinya.
func (uc *ComplianceUsecase) evaluateObligations(ctx context.Context, sessionID, evidenceID, text string) error {
	states, err := uc.compliance.ListStates(ctx, sessionID)
	if err != nil {
		return err
	}
	pending := map[string]bool{}
	for _, st := range states {
		if st.Status == domain.ObligationPending {
			pending[st.Code] = true
		}
	}
	for _, ob := range uc.obligations {
		if !pending[ob.Code] {
			continue
		}
		// Evidence gate menahan false-green dan menghemat panggilan LLM.
		// LLM hanya menilai ucapan yang memiliki bukti minimum sesuai butir.
		if !isEvidenceCandidate(ob.Code, text) {
			continue
		}
		matched, conf, err := uc.matcher.Match(ctx, text, ob)
		if err != nil {
			// Tetap arah gagal yang aman — tidak ada centang hijau tanpa
			// penilai — tetapi TIDAK boleh diam. Dulu error ini ditelan, dan
			// akun yang kehilangan akses LLM Gateway hanya tampak sebagai
			// checklist yang tak pernah hijau plus pengingat tanpa henti.
			uc.reportScoringFailure(sessionID, ob.Code, evidenceID, err)
			continue
		}
		if !matched || conf < minimumMatchConfidence {
			continue
		}
		if err := uc.compliance.MarkSatisfied(ctx, sessionID, ob.Code, evidenceID, conf); err != nil {
			return err
		}
		uc.broadcaster.Publish(sessionID, map[string]any{
			"type": "obligation_satisfied", "code": ob.Code, "confidence": conf, "evidence_id": evidenceID,
		})
	}
	return nil
}

// reportScoringFailure mencatat setiap kegagalan penilai ke log, dan
// memberi tahu layar SEKALI per sesi.
func (uc *ComplianceUsecase) reportScoringFailure(sessionID, code, utteranceID string, err error) {
	if uc.log != nil {
		uc.log.Error("penilai semantik gagal; kewajiban tidak dinilai",
			"sesi", sessionID, "kewajiban", code, "ucapan", utteranceID, "err", err)
	}
	uc.mu.Lock()
	first := !uc.scoringDown[sessionID]
	uc.scoringDown[sessionID] = true
	uc.mu.Unlock()
	if first {
		uc.broadcaster.Publish(sessionID, map[string]any{
			"type": "scoring_unavailable", "message": err.Error(),
		})
	}
}

// isEvidenceCandidate adalah pagar deterministik sebelum semantic match.
// Ia sengaja konservatif: lebih aman butir tetap pending dan dibisikkan
// daripada laporan memberi tanda hijau pada disclosure yang tidak lengkap.
func isEvidenceCandidate(code, utterance string) bool {
	text := strings.ToLower(strings.TrimSpace(utterance))
	if text == "" {
		return false
	}

	switch code {
	case "IDENTITY":
		return containsAny(text, "nama saya", "perkenalkan", "saya bertugas") &&
			containsAny(text, "bank", "bpr", "koperasi", "finance", "multifinance", "asuransi", "perusahaan", "pt ")
	case "RATE":
		return hasNumber(text) && containsAny(text, "bunga", "suku", "biaya", "persen", "%", "apr")
	case "TENOR":
		return hasNumber(text) &&
			containsAny(text, "tenor", "jangka waktu", "bulan", "tahun") &&
			containsAny(text, "cicilan", "angsuran", "per bulan", "per minggu")
	case "PENALTY":
		return containsAny(text, "denda", "terlambat", "keterlambatan", "penalti", "tunggakan")
	case "RIGHT":
		return containsAny(text, "hak", "boleh", "dapat", "bisa") &&
			containsAny(text, "menolak", "membatalkan", "batalkan", "tidak melanjutkan", "tidak setuju")
	default:
		return false
	}
}

func hasNumber(text string) bool {
	return digitPattern.MatchString(text) || containsAny(text, numberWords...)
}

func containsAny(text string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(text, candidate) {
			return true
		}
	}
	return false
}

// RemindPending membisikkan butir yang masih pending ke earpiece petugas.
// Dipanggil oleh timer, bukan setiap ucapan, supaya tidak berisik.
//
// Dibatasi dua cara: paling sering sekali per reminderSpacing, dan tiap butir
// paling banyak maxRemindersPerObligation kali. Butir yang sudah dua kali
// diingatkan dilewati, jadi butir berikutnya yang dibisikkan — alih-alih
// mengulang butir yang sama sampai sesi selesai.
func (uc *ComplianceUsecase) RemindPending(ctx context.Context, sessionID string) error {
	if uc.scorerProblem != "" {
		uc.reportScoringFailure(sessionID, "", "", errors.New(uc.scorerProblem))
		return nil
	}
	states, err := uc.compliance.ListStates(ctx, sessionID)
	if err != nil {
		return err
	}
	labels := map[string]string{}
	for _, ob := range uc.obligations {
		labels[ob.Code] = ob.Label
	}

	now := uc.now()
	uc.mu.Lock()
	log := uc.reminders[sessionID]
	if log == nil {
		log = &reminderLog{count: map[string]int{}}
		uc.reminders[sessionID] = log
	}
	if !log.last.IsZero() && now.Sub(log.last) < reminderSpacing {
		uc.mu.Unlock()
		return nil
	}
	code := ""
	for _, st := range states {
		if st.Status == domain.ObligationPending && log.count[st.Code] < maxRemindersPerObligation {
			code = st.Code
			break
		}
	}
	if code != "" {
		log.last = now
		log.count[code]++
	}
	uc.mu.Unlock()
	if code == "" {
		return nil
	}

	reminder := "Belum disampaikan: " + labels[code]
	uc.rememberNudge(sessionID, reminder)
	return uc.nudger.Whisper(ctx, sessionID, Nudge{Text: reminder, Kind: NudgePendingObligation, Code: code})
}
