package assemblyai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
)

// FailoverMatcher mencoba beberapa penyedia LLM berurutan.
//
// Urutan di LLM_PROVIDERS adalah prioritas: penyedia pertama selalu dicoba
// lebih dulu. Kalau ia kena batas kuota, ditolak, atau gangguan, ucapan yang
// SAMA langsung dicoba ke penyedia berikutnya — petugas tidak merasakan apa
// pun — dan penyedia yang gagal diistirahatkan sementara supaya ucapan
// berikutnya tidak membuang waktu menunggu penolakan yang sama. Begitu masa
// istirahatnya lewat, ia kembali dicoba lebih dulu.
type FailoverMatcher struct {
	providers []*LLMMatcher
	log       *slog.Logger
	now       func() time.Time

	mu    sync.Mutex
	until []time.Time // penyedia ke-i diistirahatkan sampai waktu ini
}

func NewFailoverMatcher(providers []*LLMMatcher, log *slog.Logger) *FailoverMatcher {
	return &FailoverMatcher{
		providers: providers,
		log:       log,
		now:       time.Now,
		until:     make([]time.Time, len(providers)),
	}
}

// ProviderStatus adalah hasil uji akses satu penyedia saat startup.
type ProviderStatus struct {
	Name    string
	Model   string
	Problem string // kosong = siap
}

// PingAll menguji setiap penyedia. Yang gagal langsung diistirahatkan,
// supaya ucapan pertama tidak menunggu penolakan yang sudah diketahui.
func (f *FailoverMatcher) PingAll(ctx context.Context) []ProviderStatus {
	out := make([]ProviderStatus, len(f.providers))
	for i, p := range f.providers {
		out[i] = ProviderStatus{Name: p.Name(), Model: p.Model()}
		if err := p.Ping(ctx); err != nil {
			out[i].Problem = err.Error()
			f.rest(i, err)
		}
	}
	return out
}

func (f *FailoverMatcher) Match(ctx context.Context, utterance string, ob domain.Obligation) (bool, float64, error) {
	if len(f.providers) == 0 {
		return false, 0, errors.New("tidak ada penyedia LLM yang dikonfigurasi (LLM_PROVIDERS)")
	}
	var failures []string
	for i, p := range f.providers {
		if f.resting(i) {
			continue
		}
		matched, conf, err := p.Match(ctx, utterance, ob)
		if err == nil {
			return matched, conf, nil
		}
		f.rest(i, err)
		failures = append(failures, err.Error())
		if f.log != nil {
			f.log.Warn("penilai gagal, mencoba penyedia berikutnya",
				"penyedia", p.Name(), "model", p.Model(), "err", err)
		}
		if ctx.Err() != nil {
			break
		}
	}
	if len(failures) == 0 {
		return false, 0, fmt.Errorf("semua penyedia LLM sedang diistirahatkan setelah gagal: %s", f.restingSummary())
	}
	return false, 0, fmt.Errorf("semua penyedia LLM gagal: %s", strings.Join(failures, " | "))
}

func (f *FailoverMatcher) resting(i int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now().Before(f.until[i])
}

func (f *FailoverMatcher) rest(i int, err error) {
	d := restFor(err)
	if d <= 0 {
		return
	}
	f.mu.Lock()
	f.until[i] = f.now().Add(d)
	f.mu.Unlock()
}

func (f *FailoverMatcher) restingSummary() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	parts := make([]string, 0, len(f.providers))
	for i, p := range f.providers {
		if left := f.until[i].Sub(f.now()); left > 0 {
			parts = append(parts, fmt.Sprintf("%s %ds lagi", p.Name(), int(left.Seconds())))
		}
	}
	return strings.Join(parts, ", ")
}

// restFor menentukan berapa lama penyedia diistirahatkan setelah gagal.
func restFor(err error) time.Duration {
	if errors.Is(err, ErrBadVerdict) {
		// Penyedianya sehat; hanya satu jawaban yang meleset. Coba penyedia
		// lain untuk ucapan ini, tapi jangan hukum penyedianya.
		return 0
	}
	var ge *GatewayError
	if !errors.As(err, &ge) {
		return 30 * time.Second // jaringan atau timeout
	}
	reason := strings.ToLower(ge.Reason)
	switch {
	case ge.Status == 429:
		if ge.RetryAfter > 0 {
			return ge.RetryAfter
		}
		return time.Minute
	case ge.Status == 401 || ge.Status == 403,
		strings.Contains(reason, "access"),
		strings.Contains(reason, "api key"),
		strings.Contains(reason, "quota"):
		// Kunci salah, akun tanpa akses, atau kuota harian habis: tidak akan
		// pulih dalam hitungan detik.
		return 30 * time.Minute
	case ge.Status >= 500:
		return 30 * time.Second
	default:
		// Permintaan ditolak karena formatnya (mis. model tidak dikenal).
		return 10 * time.Minute
	}
}

// ProviderSpec adalah isi LLM_PROVIDERS beserta kunci dan model tiap penyedia.
type ProviderSpec struct {
	Order  string            // mis. "gemini,groq,assemblyai"
	Keys   map[string]string // nama penyedia → kunci, boleh dipisah koma
	Models map[string]string // nama penyedia → model; kosong = bawaan
}

// BuildProviders menyusun penilai berurutan sesuai prioritas. Penyedia tanpa
// kunci atau tak dikenal dilewati dan dilaporkan sebagai peringatan, bukan
// menggagalkan startup — satu penyedia yang siap sudah cukup.
func BuildProviders(spec ProviderSpec) (providers []*LLMMatcher, warnings []string) {
	for _, name := range splitList(spec.Order) {
		if !KnownProvider(name) {
			warnings = append(warnings, fmt.Sprintf("penyedia LLM %q tidak dikenal, dilewati", name))
			continue
		}
		keys := splitList(spec.Keys[name])
		if len(keys) == 0 {
			warnings = append(warnings, fmt.Sprintf("penyedia LLM %q tanpa kunci API, dilewati", name))
			continue
		}
		for i, key := range keys {
			label := name
			if len(keys) > 1 {
				label = fmt.Sprintf("%s#%d", name, i+1)
			}
			p, _ := NewProvider(name, label, key, spec.Models[name])
			providers = append(providers, p)
		}
	}
	return providers, warnings
}

func splitList(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
