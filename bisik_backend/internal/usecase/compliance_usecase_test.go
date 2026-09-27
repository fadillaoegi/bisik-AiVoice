package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/bisik/bisik_backend/internal/domain"
)

type complianceRepoFake struct {
	states     []*domain.ObligationState
	marked     []string
	violations []*domain.Violation
}

func (f *complianceRepoFake) InitStates(domain.Context, string, []domain.Obligation) error {
	return nil
}
func (f *complianceRepoFake) ListStates(domain.Context, string) ([]*domain.ObligationState, error) {
	return f.states, nil
}
func (f *complianceRepoFake) MarkSatisfied(_ domain.Context, _ string, code, evidenceID string, confidence float64) error {
	f.marked = append(f.marked, code)
	for _, state := range f.states {
		if state.Code == code {
			state.Status = domain.ObligationSatisfied
			state.EvidenceID = evidenceID
			state.Confidence = confidence
		}
	}
	return nil
}
func (f *complianceRepoFake) RecordViolation(_ domain.Context, violation *domain.Violation) error {
	f.violations = append(f.violations, violation)
	return nil
}
func (f *complianceRepoFake) ListViolations(domain.Context, string) ([]*domain.Violation, error) {
	return f.violations, nil
}

type transcriptRepoFake struct {
	items   map[string]*domain.Utterance
	revised map[string]domain.Speaker
}

func newTranscriptRepoFake() *transcriptRepoFake {
	return &transcriptRepoFake{items: map[string]*domain.Utterance{}, revised: map[string]domain.Speaker{}}
}
func (f *transcriptRepoFake) Append(_ domain.Context, utterance *domain.Utterance) error {
	copy := *utterance
	f.items[utterance.ID] = &copy
	return nil
}
func (f *transcriptRepoFake) FindByID(_ domain.Context, id string) (*domain.Utterance, error) {
	utterance, ok := f.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return utterance, nil
}
func (f *transcriptRepoFake) ListBySession(domain.Context, string) ([]*domain.Utterance, error) {
	return nil, nil
}
func (f *transcriptRepoFake) ReviseSpeaker(_ domain.Context, id string, speaker domain.Speaker) error {
	f.revised[id] = speaker
	if utterance, ok := f.items[id]; ok {
		utterance.Speaker = speaker
		utterance.Revised = true
	}
	return nil
}

type matcherFake struct {
	matched    bool
	confidence float64
	err        error
	calls      int
}

func (f *matcherFake) Match(context.Context, string, domain.Obligation) (bool, float64, error) {
	f.calls++
	return f.matched, f.confidence, f.err
}

type guardFake struct {
	phrase   string
	severity string
	found    bool
	err      error
}

func (f *guardFake) Inspect(context.Context, string) (string, string, bool, error) {
	return f.phrase, f.severity, f.found, f.err
}

type nudgerFake struct{ messages []string }

func (f *nudgerFake) Whisper(_ context.Context, _ string, message string) error {
	f.messages = append(f.messages, message)
	return nil
}

type broadcasterFake struct{ events []any }

func (f *broadcasterFake) Publish(_ string, event any) { f.events = append(f.events, event) }

const sessionIDForTest = "sesi-1"

func pendingState(code string) []*domain.ObligationState {
	return []*domain.ObligationState{{SessionID: "sesi-1", Code: code, Status: domain.ObligationPending}}
}

func newComplianceTestUsecase(states []*domain.ObligationState, matcher *matcherFake, guard *guardFake) (*ComplianceUsecase, *complianceRepoFake, *transcriptRepoFake, *nudgerFake) {
	compliance := &complianceRepoFake{states: states}
	transcripts := newTranscriptRepoFake()
	nudger := &nudgerFake{}
	uc := NewComplianceUsecase(compliance, transcripts, matcher, guard, nudger, &broadcasterFake{})
	return uc, compliance, transcripts, nudger
}

func TestHandleTranscriptMenandaiBuktiYangLolosDuaGate(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.93}
	uc, compliance, transcripts, _ := newComplianceTestUsecase(pendingState("RATE"), matcher, &guardFake{})

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer,
		Text: "Suku bunganya dua persen per bulan.", IsFinal: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.marked) != 1 || compliance.marked[0] != "RATE" {
		t.Fatalf("marked = %v, mau [RATE]", compliance.marked)
	}
	if matcher.calls != 1 {
		t.Fatalf("matcher dipanggil %d kali, mau 1", matcher.calls)
	}
	if transcripts.items["u-1"] == nil {
		t.Fatal("utterance final tidak disimpan")
	}
}

func TestHandleTranscriptMenolakFalseGreenTanpaAngka(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.99}
	uc, compliance, _, _ := newComplianceTestUsecase(pendingState("RATE"), matcher, &guardFake{})

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer,
		Text: "Nanti bunga mengikuti ketentuan yang berlaku.", IsFinal: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.marked) != 0 {
		t.Fatalf("ucapan tanpa angka tidak boleh memenuhi RATE: %v", compliance.marked)
	}
	if matcher.calls != 0 {
		t.Fatalf("evidence gate harus menahan panggilan LLM, dipanggil %d kali", matcher.calls)
	}
}

func TestHandleTranscriptMenolakConfidenceRendah(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.79}
	uc, compliance, _, _ := newComplianceTestUsecase(pendingState("PENALTY"), matcher, &guardFake{})

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer,
		Text: "Jika terlambat akan ada denda.", IsFinal: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.marked) != 0 {
		t.Fatalf("confidence rendah tidak boleh ditandai: %v", compliance.marked)
	}
}

func TestHandleTranscriptNasabahTidakMemenuhiKewajiban(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.99}
	uc, compliance, _, _ := newComplianceTestUsecase(pendingState("RATE"), matcher, &guardFake{})

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerCustomer,
		Text: "Berarti bunganya dua persen per bulan?", IsFinal: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.marked) != 0 || matcher.calls != 0 {
		t.Fatalf("ucapan nasabah tidak boleh dinilai: marked=%v calls=%d", compliance.marked, matcher.calls)
	}
}

func TestHandleTranscriptKalibrasiTidakDisimpanAtauDinilai(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.99}
	guard := &guardFake{phrase: "pasti cair", severity: "high", found: true}
	uc, compliance, transcripts, nudger := newComplianceTestUsecase(pendingState("RATE"), matcher, guard)

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "kal-1", Speaker: domain.SpeakerUnknown, SourceSpeaker: "A",
		Text: "Bunga dua persen dan pasti cair.", IsFinal: true, IsCalibration: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(kalibrasi): %v", err)
	}
	if len(transcripts.items) != 0 || matcher.calls != 0 || len(compliance.marked) != 0 ||
		len(compliance.violations) != 0 || len(nudger.messages) != 0 {
		t.Fatalf("kalibrasi bocor ke penilaian: transcript=%d calls=%d marked=%v violations=%d nudge=%d",
			len(transcripts.items), matcher.calls, compliance.marked, len(compliance.violations), len(nudger.messages))
	}
}

func TestHandleRevisionMenilaiUcapanYangBerubahJadiPetugas(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.91}
	uc, compliance, transcripts, _ := newComplianceTestUsecase(pendingState("PENALTY"), matcher, &guardFake{})
	transcripts.items["u-1"] = &domain.Utterance{
		ID: "u-1", SessionID: "sesi-1", Speaker: domain.SpeakerCustomer,
		Text: "Kalau terlambat ada denda lima puluh ribu rupiah.",
	}

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer, IsFinal: true, IsRevision: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(revision): %v", err)
	}
	if transcripts.revised["u-1"] != domain.SpeakerOfficer {
		t.Fatalf("speaker tidak direvisi: %v", transcripts.revised)
	}
	if len(compliance.marked) != 1 || compliance.marked[0] != "PENALTY" {
		t.Fatalf("revision tidak memicu evaluasi: %v", compliance.marked)
	}
}

func TestHandleTranscriptMencatatJanjiTerlarangDanMembisikkan(t *testing.T) {
	matcher := &matcherFake{}
	guard := &guardFake{phrase: "pasti cair", severity: "high", found: true}
	uc, compliance, _, nudger := newComplianceTestUsecase(nil, matcher, guard)

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer,
		Text: "Tenang, pengajuan ini pasti cair.", IsFinal: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.violations) != 1 || compliance.violations[0].EvidenceID != "u-1" {
		t.Fatalf("violation = %+v", compliance.violations)
	}
	if len(nudger.messages) != 1 {
		t.Fatalf("nudge = %v, mau satu pesan", nudger.messages)
	}
}

func TestEvidenceCandidateButirUtama(t *testing.T) {
	cases := []struct {
		code string
		text string
		want bool
	}{
		{"IDENTITY", "Perkenalkan, nama saya Rani dari BPR Nusantara.", true},
		{"RATE", "Suku bunganya 2 persen per bulan.", true},
		{"RATE", "Ada bunga sesuai ketentuan.", false},
		{"TENOR", "Tenornya dua belas bulan dengan cicilan satu juta per bulan.", true},
		{"TENOR", "Tenornya dua belas bulan.", false},
		{"PENALTY", "Keterlambatan dikenai denda.", true},
		{"RIGHT", "Bapak berhak membatalkan penawaran ini.", true},
		{"RIGHT", "Penawaran ini dapat dipertimbangkan.", false},
	}

	for _, tc := range cases {
		t.Run(tc.code+"/"+tc.text, func(t *testing.T) {
			if got := isEvidenceCandidate(tc.code, tc.text); got != tc.want {
				t.Fatalf("isEvidenceCandidate() = %v, mau %v", got, tc.want)
			}
		})
	}
}

func TestHandleTranscriptMengabaikanErrorMatcherTanpaFalseGreen(t *testing.T) {
	matcher := &matcherFake{err: errors.New("gateway timeout")}
	uc, compliance, _, _ := newComplianceTestUsecase(pendingState("PENALTY"), matcher, &guardFake{})

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer,
		Text: "Kalau terlambat ada denda.", IsFinal: true,
	})
	if err != nil {
		t.Fatalf("error matcher tidak boleh memutus stream: %v", err)
	}
	if len(compliance.marked) != 0 {
		t.Fatalf("error matcher tidak boleh menjadi tanda hijau: %v", compliance.marked)
	}
}

// newComplianceTestUsecaseWithEvents sama dengan helper di atas, tetapi juga
// mengembalikan broadcaster supaya event peringatan bisa diperiksa.
func newComplianceTestUsecaseWithEvents(
	states []*domain.ObligationState, matcher *matcherFake, guard *guardFake,
) (*ComplianceUsecase, *complianceRepoFake, *broadcasterFake) {
	compliance := &complianceRepoFake{states: states}
	broadcaster := &broadcasterFake{}
	uc := NewComplianceUsecase(
		compliance, newTranscriptRepoFake(), matcher, guard, &nudgerFake{}, broadcaster,
	)
	return uc, compliance, broadcaster
}

func countEventType(events []any, eventType string) int {
	n := 0
	for _, event := range events {
		if payload, ok := event.(map[string]any); ok && payload["type"] == eventType {
			n++
		}
	}
	return n
}

// Pembicara ketiga atau ucapan yang diarization-nya menyerah tidak boleh
// memenuhi checklist: kalimat yang sama dari petugas memang lolos, jadi
// satu-satunya pembeda adalah identitas pembicaranya.
func TestHandleTranscriptPembicaraTidakDikenalTidakJadiBukti(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.99}
	uc, compliance, broadcaster := newComplianceTestUsecaseWithEvents(
		pendingState("RATE"), matcher, &guardFake{},
	)

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerUnknown,
		Text: "Suku bunganya dua persen per bulan.", IsFinal: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.marked) != 0 {
		t.Fatalf("marked = %v, mau kosong — suara tak dikenal bukan bukti", compliance.marked)
	}
	if matcher.calls != 0 {
		t.Fatalf("matcher dipanggil %d kali, mau 0", matcher.calls)
	}
	if countEventType(broadcaster.events, "speaker_unknown") != 1 {
		t.Fatal("petugas tidak diberi tahu ucapan itu dilewati")
	}
}

// Audio tidak layak menahan checklist, tetapi pelanggaran tetap dicatat:
// menahan centang hijau itu aman, membiarkan janji terlarang lolos tidak.
func TestAudioTidakLayakMenahanChecklistTapiTetapMencatatPelanggaran(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.99}
	guard := &guardFake{phrase: "pasti cair", severity: "high", found: true}
	uc, compliance, broadcaster := newComplianceTestUsecaseWithEvents(
		pendingState("RATE"), matcher, guard,
	)

	uc.SetAudioQuality("sesi-1", "Suara terlalu pelan dari mikrofon")

	err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer,
		Text: "Suku bunganya dua persen, dana pasti cair.", IsFinal: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.marked) != 0 {
		t.Fatalf("marked = %v, mau kosong selama audio buruk", compliance.marked)
	}
	if matcher.calls != 0 {
		t.Fatalf("matcher dipanggil %d kali, mau 0 — hemat panggilan LLM", matcher.calls)
	}
	if len(compliance.violations) != 1 {
		t.Fatalf("violations = %d, mau 1 — guardrail harus tetap jalan", len(compliance.violations))
	}
	if countEventType(broadcaster.events, "evidence_skipped") != 1 {
		t.Fatal("alasan ucapan dilewati tidak disampaikan ke petugas")
	}

	// Audio pulih: kalimat yang sama sekarang boleh dinilai.
	uc.SetAudioQuality("sesi-1", "")
	if err := uc.HandleTranscript(context.Background(), "sesi-1", TranscriptEvent{
		UtteranceID: "u-2", Speaker: domain.SpeakerOfficer,
		Text: "Suku bunganya dua persen per bulan.", IsFinal: true,
	}); err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.marked) != 1 || compliance.marked[0] != "RATE" {
		t.Fatalf("marked = %v, mau [RATE] setelah audio pulih", compliance.marked)
	}
}

func TestSetAudioQualityHanyaMenyiarkanSaatBerubah(t *testing.T) {
	uc, _, broadcaster := newComplianceTestUsecaseWithEvents(
		pendingState("RATE"), &matcherFake{}, &guardFake{},
	)

	uc.SetAudioQuality("sesi-1", "Suara terlalu pelan dari mikrofon")
	uc.SetAudioQuality("sesi-1", "Suara terlalu pelan dari mikrofon")
	if got := countEventType(broadcaster.events, "audio_quality"); got != 1 {
		t.Fatalf("audio_quality disiarkan %d kali, mau 1 — jangan membanjiri UI", got)
	}

	uc.SetAudioQuality("sesi-1", "")
	if got := countEventType(broadcaster.events, "audio_quality"); got != 2 {
		t.Fatalf("audio_quality disiarkan %d kali, mau 2 — pemulihan harus diumumkan", got)
	}
}

// Bisikan aplikasi bisa terekam balik kalau petugas memakai speaker.
// Kalau dibiarkan, sistem menilai suaranya sendiri: bisikan
// "Belum disampaikan: Denda keterlambatan" memuat kata "denda" dan
// "keterlambatan", yang persis dicari evidence gate PENALTY.
func TestGemaBisikanTidakBolehMemenuhiKewajiban(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.99}
	uc, compliance, broadcaster := newComplianceTestUsecaseWithEvents(
		pendingState("PENALTY"), matcher, &guardFake{},
	)
	ctx := context.Background()

	// Pengingat berkala diucapkan…
	if err := uc.RemindPending(ctx, sessionIDForTest); err != nil {
		t.Fatalf("RemindPending(): %v", err)
	}

	// …lalu mikrofon menangkapnya kembali, nyaris kata per kata.
	err := uc.HandleTranscript(ctx, sessionIDForTest, TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer,
		Text: "belum disampaikan denda keterlambatan", IsFinal: true,
	})
	if err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}

	if len(compliance.marked) != 0 {
		t.Fatalf("marked = %v, mau kosong — itu suara aplikasi sendiri", compliance.marked)
	}
	if matcher.calls != 0 {
		t.Fatalf("matcher dipanggil %d kali, mau 0", matcher.calls)
	}
	if countEventType(broadcaster.events, "evidence_skipped") != 1 {
		t.Fatal("petugas tidak diberi tahu ucapan itu dilewati")
	}
}

// Gema juga tidak boleh dicatat sebagai pelanggaran kedua: bisikan koreksi
// mengutip frasa terlarangnya, jadi guardrail akan cocok lagi.
func TestGemaPeringatanTidakMenggandakanPelanggaran(t *testing.T) {
	guard := &guardFake{phrase: "pasti cair", severity: "high", found: true}
	uc, compliance, _ := newComplianceTestUsecaseWithEvents(
		pendingState("RATE"), &matcherFake{}, guard,
	)
	ctx := context.Background()

	// Pelanggaran pertama, sah.
	if err := uc.HandleTranscript(ctx, sessionIDForTest, TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer,
		Text: "dana bapak pasti cair minggu ini", IsFinal: true,
	}); err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.violations) != 1 {
		t.Fatalf("violations = %d, mau 1", len(compliance.violations))
	}

	// Bisikan koreksi tertangkap balik oleh mikrofon.
	if err := uc.HandleTranscript(ctx, sessionIDForTest, TranscriptEvent{
		UtteranceID: "u-2", Speaker: domain.SpeakerOfficer,
		Text: "hati hati hindari frasa pasti cair", IsFinal: true,
	}); err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}
	if len(compliance.violations) != 1 {
		t.Fatalf("violations = %d, mau tetap 1 — yang kedua itu gema",
			len(compliance.violations))
	}
}

// Ucapan sah yang kebetulan memakai satu dua kata yang sama TIDAK boleh
// ikut dibuang; ambangnya harus membedakan gema dari kalimat biasa.
func TestUcapanSahTidakDikiraGema(t *testing.T) {
	matcher := &matcherFake{matched: true, confidence: 0.95}
	uc, compliance, _ := newComplianceTestUsecaseWithEvents(
		pendingState("PENALTY"), matcher, &guardFake{},
	)
	ctx := context.Background()

	if err := uc.RemindPending(ctx, sessionIDForTest); err != nil {
		t.Fatalf("RemindPending(): %v", err)
	}

	if err := uc.HandleTranscript(ctx, sessionIDForTest, TranscriptEvent{
		UtteranceID: "u-1", Speaker: domain.SpeakerOfficer,
		Text: "kalau bapak terlambat membayar cicilan maka ada denda " +
			"nol koma satu persen per hari dari sisa tagihan", IsFinal: true,
	}); err != nil {
		t.Fatalf("HandleTranscript(): %v", err)
	}

	if len(compliance.marked) != 1 || compliance.marked[0] != "PENALTY" {
		t.Fatalf("marked = %v, mau [PENALTY] — kalimat petugas yang sah", compliance.marked)
	}
}
