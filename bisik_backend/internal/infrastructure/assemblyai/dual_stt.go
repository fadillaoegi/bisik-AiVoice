package assemblyai

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/usecase"
)

// DualStreamSTT menjalankan DUA koneksi AssemblyAI atas audio yang sama dan
// menggabungkan hasilnya.
//
// Alasannya dibuktikan lewat `cmd/diarprobe` pada 25 Sep: tidak ada satu
// model pun yang memberi keduanya sekaligus.
//
//	whisper-rt                        transkrip Bahasa Indonesia bagus,
//	                                  diarization TIDAK berfungsi —
//	                                  semua pembicara dilabeli "A"
//	universal-streaming-multilingual  diarization benar,
//	                                  transkrip Indonesia hancur
//
// Jadi teks diambil dari yang pertama dan label pembicara dari yang kedua,
// lalu dijodohkan berdasarkan rentang waktu. Keduanya menerima byte audio
// yang identik dari titik awal yang sama, sehingga timestamp-nya sebanding.
//
// Biayanya dua kali menit streaming. Itu harga yang disadari untuk laporan
// kepatuhan yang bisa menyebut siapa yang mengatakan apa.
type DualStreamSTT struct {
	transcriber *StreamingSTT // sumber teks
	diarizer    *StreamingSTT // sumber label pembicara
	log         *slog.Logger

	mu       sync.Mutex
	sessions map[string]*dualSession
}

// Transkrip menunggu label selama ini sebelum menyerah. Lebih lama berarti
// bisikan terlambat; lebih pendek berarti lebih sering jatuh ke "unknown"
// dan kewajiban tidak terhitung padahal petugas sudah menyampaikannya.
const labelWaitWindow = 1200 * time.Millisecond

type speakerSegment struct {
	startMS, endMS int
	label          string
}

type pendingTurn struct {
	event    usecase.TranscriptEvent
	deadline time.Time
}

type emittedTurn struct {
	startMS, endMS int
	label          string
}

type dualSession struct {
	mu       sync.Mutex
	segments []speakerSegment
	pending  []pendingTurn
	// roles disalin ke penggabung saat kalibrasi dikonfirmasi. StreamingSTT
	// membersihkan state speaker-nya ketika koneksi diarizer selesai, padahal
	// transcriber masih dapat mengirim turn terakhir sesudahnya. Salinan ini
	// menjaga atribusi tetap tersedia sampai KEDUA stream selesai.
	roles       map[string]domain.Speaker
	calibrating bool
	confirmed   bool
	// emitted mencatat ucapan yang SUDAH terkirim beserta rentang waktu dan
	// label yang dipakai, supaya koreksi label belakangan bisa dikirim
	// sebagai revisi — tanpa ini, kewajiban yang ternyata diucapkan petugas
	// tidak akan pernah dinilai ulang.
	emitted map[string]emittedTurn
	out     chan usecase.TranscriptEvent
	done    chan struct{}
}

func NewDualStreamSTT(transcriber, diarizer *StreamingSTT, log *slog.Logger) *DualStreamSTT {
	return &DualStreamSTT{
		transcriber: transcriber,
		diarizer:    diarizer,
		log:         log,
		sessions:    make(map[string]*dualSession),
	}
}

func (d *DualStreamSTT) Start(ctx context.Context, sessionID string) (<-chan usecase.TranscriptEvent, error) {
	textEvents, err := d.transcriber.Start(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	labelEvents, err := d.diarizer.Start(ctx, sessionID)
	if err != nil {
		// Transcriber sudah terbuka; tutup supaya tidak ada koneksi menggantung.
		_ = d.transcriber.Stop(sessionID)
		return nil, err
	}

	session := &dualSession{
		emitted: make(map[string]emittedTurn),
		roles:   make(map[string]domain.Speaker),
		out:     make(chan usecase.TranscriptEvent, 64),
		done:    make(chan struct{}),
	}
	d.mu.Lock()
	d.sessions[sessionID] = session
	d.mu.Unlock()

	go d.merge(ctx, sessionID, session, textEvents, labelEvents)
	return session.out, nil
}

// merge adalah satu-satunya goroutine yang menulis ke channel keluaran,
// sehingga urutan event terjaga tanpa penguncian tambahan di sisi pembaca.
func (d *DualStreamSTT) merge(
	ctx context.Context, sessionID string, session *dualSession,
	textEvents, labelEvents <-chan usecase.TranscriptEvent,
) {
	defer close(session.out)
	defer close(session.done)
	defer func() {
		d.mu.Lock()
		delete(d.sessions, sessionID)
		d.mu.Unlock()
	}()

	// Ticker membebaskan transkrip yang labelnya tidak kunjung datang.
	// Tanpa ini sebuah ucapan bisa tertahan selamanya kalau diarizer
	// kehilangan rentang waktunya.
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for textEvents != nil || labelEvents != nil {
		select {
		case <-ctx.Done():
			return

		case ev, ok := <-labelEvents:
			if !ok {
				labelEvents = nil
				continue
			}
			if ev.Err != nil {
				// Kehilangan diarizer berarti teks masih mungkin datang tetapi
				// identitas pembicaranya tidak lagi dapat dipercaya. Itu fatal
				// untuk laporan kepatuhan, jadi beri tahu hilir dan hentikan
				// seluruh pipeline alih-alih diam-diam menghasilkan unknown.
				d.emit(ctx, session, ev)
				acknowledge(ev)
				return
			}
			d.absorbLabel(ctx, sessionID, session, ev)
			// WAJIB. Upstream memblokir sampai event diakui; tanpa ini
			// readLoop membeku setelah event pertama dan seluruh sesi
			// berhenti setelah satu ucapan.
			acknowledge(ev)

		case ev, ok := <-textEvents:
			if !ok {
				textEvents = nil
				continue
			}
			if ev.Err != nil {
				d.emit(ctx, session, ev)
				acknowledge(ev)
				return
			}
			d.absorbText(ctx, sessionID, session, ev)
			acknowledge(ev)

		case <-ticker.C:
			d.resolvePending(ctx, sessionID, session, false)
		}
	}

	// Stream teks habis: keluarkan sisa apa adanya daripada menghilangkannya.
	d.resolvePending(ctx, sessionID, session, true)
}

// absorbLabel menyerap hasil diarizer: perbarui garis waktu, lalu coba
// selesaikan transkrip yang sedang menunggu dan koreksi yang sudah terkirim.
func acknowledge(ev usecase.TranscriptEvent) {
	if ev.Acknowledge != nil {
		ev.Acknowledge()
	}
}

func (d *DualStreamSTT) absorbLabel(ctx context.Context, sessionID string, session *dualSession, ev usecase.TranscriptEvent) {
	label := usableLabel(ev.SourceSpeaker)
	if label == "" || ev.EndMS <= ev.StartMS {
		return
	}

	session.mu.Lock()
	session.segments = append(session.segments, speakerSegment{
		startMS: ev.StartMS, endMS: ev.EndMS, label: label,
	})
	sort.Slice(session.segments, func(i, j int) bool {
		return session.segments[i].startMS < session.segments[j].startMS
	})
	session.mu.Unlock()

	d.resolvePending(ctx, sessionID, session, false)
	d.reviseEmitted(ctx, sessionID, session)
}

func (d *DualStreamSTT) absorbText(ctx context.Context, sessionID string, session *dualSession, ev usecase.TranscriptEvent) {
	// Partial hanya untuk ditampilkan sekilas; jangan ditahan menunggu label.
	if !ev.IsFinal {
		ev.Speaker = domain.SpeakerUnknown
		ev.SourceSpeaker = ""
		d.emit(ctx, session, ev)
		return
	}

	session.mu.Lock()
	session.pending = append(session.pending, pendingTurn{
		event: ev, deadline: time.Now().Add(labelWaitWindow),
	})
	session.mu.Unlock()

	d.resolvePending(ctx, sessionID, session, false)
}

// resolvePending mengirim transkrip yang labelnya sudah tersedia.
// `force` mengirim sisanya sebagai unknown — jalur gagal yang aman, karena
// ucapan tanpa identitas memang tidak boleh memenuhi kewajiban.
func (d *DualStreamSTT) resolvePending(ctx context.Context, sessionID string, session *dualSession, force bool) {
	now := time.Now()

	session.mu.Lock()
	keep := session.pending[:0]
	var ready []usecase.TranscriptEvent
	for _, p := range session.pending {
		label := dominantLabel(session.segments, p.event.StartMS, p.event.EndMS)
		expired := force || now.After(p.deadline)
		if label == "" && !expired {
			keep = append(keep, p)
			continue
		}
		p.event.SourceSpeaker = label
		session.emitted[p.event.UtteranceID] = emittedTurn{
			startMS: p.event.StartMS, endMS: p.event.EndMS, label: label,
		}
		ready = append(ready, p.event)
	}
	session.pending = keep
	session.mu.Unlock()

	for _, ev := range ready {
		role, calibrating := d.resolveLabelRole(sessionID, session, ev.SourceSpeaker)
		ev.Speaker = role
		ev.IsCalibration = calibrating
		if ev.SourceSpeaker == "" {
			d.log.Warn("ucapan tanpa label pembicara setelah menunggu",
				"session", sessionID, "utterance", ev.UtteranceID)
		}
		d.emit(ctx, session, ev)
	}
}

// reviseEmitted mengirim revisi ketika label sebuah ucapan yang SUDAH
// terkirim berubah setelah ada segmen baru dari diarizer.
//
// Inilah yang membuat koreksi diarization MEMPERBAIKI laporan alih-alih
// merusaknya: ucapan yang ternyata milik petugas belum pernah dinilai, jadi
// harus dinilai sekarang. Use case sudah menangani sisi penilaian ulangnya.
func (d *DualStreamSTT) reviseEmitted(ctx context.Context, sessionID string, session *dualSession) {
	session.mu.Lock()
	var revisions []usecase.TranscriptEvent
	for id, previous := range session.emitted {
		current := dominantLabel(session.segments, previous.startMS, previous.endMS)
		if current == "" || current == previous.label {
			continue
		}
		session.emitted[id] = emittedTurn{
			startMS: previous.startMS, endMS: previous.endMS, label: current,
		}
		revisions = append(revisions, usecase.TranscriptEvent{
			UtteranceID:   id,
			SourceSpeaker: current,
			StartMS:       previous.startMS,
			EndMS:         previous.endMS,
			IsFinal:       true,
			IsRevision:    true,
		})
	}
	session.mu.Unlock()

	for _, ev := range revisions {
		role, calibrating := d.resolveLabelRole(sessionID, session, ev.SourceSpeaker)
		ev.Speaker = role
		ev.IsCalibration = calibrating
		d.log.Info("label pembicara dikoreksi",
			"session", sessionID, "utterance", ev.UtteranceID, "label", ev.SourceSpeaker)
		d.emit(ctx, session, ev)
	}
}

// emit meneruskan event ke hilir dan MENUNGGU pengakuan, meniru kontrak
// StreamingSTT. Ini yang menjaga `Terminate` tetap menjadi flush barrier:
// saat goroutine penggabung selesai, seluruh event sudah diproses hilir.
func (d *DualStreamSTT) emit(ctx context.Context, session *dualSession, ev usecase.TranscriptEvent) {
	ack := make(chan struct{})
	var once sync.Once
	ev.Acknowledge = func() { once.Do(func() { close(ack) }) }

	select {
	case session.out <- ev:
	case <-ctx.Done():
		return
	}
	select {
	case <-ack:
	case <-ctx.Done():
	}
}

// dominantLabel memilih label yang paling banyak menutupi rentang ucapan.
// Dipilih berdasarkan durasi tumpang tindih, bukan sekadar yang pertama
// menyentuh, supaya sisipan pendek dari orang lain tidak merebut ucapan.
func dominantLabel(segments []speakerSegment, startMS, endMS int) string {
	if endMS <= startMS {
		endMS = startMS + 1
	}
	overlaps := map[string]int{}
	for _, seg := range segments {
		lo := max(seg.startMS, startMS)
		hi := min(seg.endMS, endMS)
		if hi > lo {
			overlaps[seg.label] += hi - lo
		}
	}
	best, bestOverlap := "", 0
	for label, n := range overlaps {
		if n > bestOverlap {
			best, bestOverlap = label, n
		}
	}
	return best
}

func (d *DualStreamSTT) PushAudio(sessionID string, pcm []byte) error {
	// Kedua koneksi menerima byte yang sama persis. Kalau salah satu gagal,
	// error dikembalikan tetapi yang lain tetap diberi makan — sesi yang
	// setengah hidup lebih berguna daripada mati dua-duanya.
	errText := d.transcriber.PushAudio(sessionID, pcm)
	errLabel := d.diarizer.PushAudio(sessionID, pcm)
	if errText != nil {
		return errText
	}
	return errLabel
}

func (d *DualStreamSTT) Stop(sessionID string) error {
	// Diarizer dihentikan lebih dulu agar revisi label final sudah masuk ke
	// garis waktu sebelum transcriber mengirim turn terakhir. Mapping role
	// tidak ikut hilang karena DualStreamSTT menyimpannya di dualSession.
	errLabel := d.diarizer.Stop(sessionID)
	errText := d.transcriber.Stop(sessionID)
	if errText != nil {
		return errText
	}
	return errLabel
}

// --- Kalibrasi didelegasikan ke diarizer: dialah yang melihat label. ---

func (d *DualStreamSTT) BeginSpeakerCalibration(sessionID string) error {
	if err := d.diarizer.BeginSpeakerCalibration(sessionID); err != nil {
		return err
	}
	if session := d.session(sessionID); session != nil {
		session.mu.Lock()
		session.roles = make(map[string]domain.Speaker)
		session.calibrating = true
		session.confirmed = false
		session.mu.Unlock()
	}
	return nil
}

func (d *DualStreamSTT) ConfirmSpeakerRoles(sessionID, officerLabel, customerLabel string) error {
	if err := d.diarizer.ConfirmSpeakerRoles(sessionID, officerLabel, customerLabel); err != nil {
		return err
	}
	if session := d.session(sessionID); session != nil {
		session.mu.Lock()
		session.roles = map[string]domain.Speaker{
			officerLabel:  domain.SpeakerOfficer,
			customerLabel: domain.SpeakerCustomer,
		}
		session.calibrating = false
		session.confirmed = true
		session.mu.Unlock()
	}
	return nil
}

func (d *DualStreamSTT) SpeakerCalibrationActive(sessionID string) bool {
	if session := d.session(sessionID); session != nil {
		session.mu.Lock()
		defer session.mu.Unlock()
		return session.calibrating
	}
	return d.diarizer.SpeakerCalibrationActive(sessionID)
}

func (d *DualStreamSTT) session(sessionID string) *dualSession {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sessions[sessionID]
}

// resolveLabelRole memakai mapping milik penggabung lebih dulu. Selain
// mempertahankan role saat diarizer sudah cleanup, fungsi ini juga menyimpan
// hasil fallback klien lama supaya finalisasi tetap konsisten.
func (d *DualStreamSTT) resolveLabelRole(sessionID string, session *dualSession, label string) (domain.Speaker, bool) {
	if usableLabel(label) == "" {
		session.mu.Lock()
		calibrating := session.calibrating
		session.mu.Unlock()
		return domain.SpeakerUnknown, calibrating
	}

	session.mu.Lock()
	if session.calibrating {
		session.mu.Unlock()
		return domain.SpeakerUnknown, true
	}
	if role, ok := session.roles[label]; ok {
		session.mu.Unlock()
		return role, false
	}
	confirmed := session.confirmed
	session.mu.Unlock()
	if confirmed {
		return domain.SpeakerUnknown, false
	}

	role, calibrating := d.diarizer.ResolveLabelRole(sessionID, label)
	if role != domain.SpeakerUnknown && !calibrating {
		session.mu.Lock()
		if !session.confirmed {
			if session.roles == nil {
				session.roles = make(map[string]domain.Speaker)
			}
			session.roles[label] = role
		}
		session.mu.Unlock()
	}
	return role, calibrating
}
