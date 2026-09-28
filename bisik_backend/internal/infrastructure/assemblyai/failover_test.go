package assemblyai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
)

// fakeProvider adalah penyedia LLM tiruan. respond menentukan jawabannya.
type fakeProvider struct {
	srv   *httptest.Server
	calls atomic.Int32
	auth  atomic.Value
}

func newFake(t *testing.T, respond func(w http.ResponseWriter)) *fakeProvider {
	t.Helper()
	f := &fakeProvider{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		f.auth.Store(r.Header.Get("Authorization"))
		respond(w)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeProvider) matcher(name string, bearer bool) *LLMMatcher {
	m := newLLMMatcher("kunci-"+name, "model-"+name, f.srv.URL)
	m.name, m.bearer = name, bearer
	return m
}

func ok(w http.ResponseWriter) {
	_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"matched\":true,\"confidence\":0.9}"}}]}`))
}

func status(code int, body string, header ...string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		for i := 0; i+1 < len(header); i += 2 {
			w.Header().Set(header[i], header[i+1])
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}
}

var identity = domain.DefaultObligations()[0]

func failover(ms ...*LLMMatcher) (*FailoverMatcher, *time.Time) {
	f := NewFailoverMatcher(ms, nil)
	clock := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	f.now = func() time.Time { return clock }
	return f, &clock
}

// Inti fitur: kuota penyedia pertama habis, ucapan yang SAMA langsung dinilai
// penyedia berikutnya — petugas tidak kehilangan penilaian.
func TestKuotaHabisPindahKePenyediaBerikutnya(t *testing.T) {
	gemini := newFake(t, status(429, `{"error":{"message":"Resource has been exhausted"}}`))
	groq := newFake(t, ok)
	f, _ := failover(gemini.matcher("gemini", true), groq.matcher("groq", true))

	matched, conf, err := f.Match(context.Background(), "nama saya handoko dari bank nusantara", identity)
	if err != nil || !matched || conf != 0.9 {
		t.Fatalf("Match() = %v %.2f %v, mau dinilai oleh groq", matched, conf, err)
	}
	if gemini.calls.Load() != 1 || groq.calls.Load() != 1 {
		t.Fatalf("panggilan gemini=%d groq=%d, mau 1 dan 1", gemini.calls.Load(), groq.calls.Load())
	}
}

// Penyedia yang baru kena batas diistirahatkan: ucapan berikutnya tidak
// membuang waktu menunggu penolakan yang sama. Setelah istirahatnya lewat,
// penyedia prioritas utama dipakai lagi.
func TestPenyediaDiistirahatkanLaluDicobaLagi(t *testing.T) {
	gemini := newFake(t, status(429, `{"error":{"message":"rate limit"}}`, "Retry-After", "20"))
	groq := newFake(t, ok)
	f, clock := failover(gemini.matcher("gemini", true), groq.matcher("groq", true))
	ctx := context.Background()

	_, _, _ = f.Match(ctx, "a", identity)
	_, _, _ = f.Match(ctx, "b", identity)
	if gemini.calls.Load() != 1 {
		t.Fatalf("gemini dipanggil %d kali selama istirahat, mau 1", gemini.calls.Load())
	}

	*clock = clock.Add(21 * time.Second) // Retry-After 20 detik sudah lewat
	_, _, _ = f.Match(ctx, "c", identity)
	if gemini.calls.Load() != 2 {
		t.Fatalf("gemini tidak dicoba lagi setelah Retry-After lewat (panggilan=%d)", gemini.calls.Load())
	}
}

func TestLamaIstirahatMenurutJenisKegagalan(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want time.Duration
	}{
		{"429 dengan Retry-After", &GatewayError{Status: 429, RetryAfter: 7 * time.Second}, 7 * time.Second},
		{"429 tanpa Retry-After", &GatewayError{Status: 429}, time.Minute},
		{"akun tanpa akses", &GatewayError{Status: 400, Reason: "Your account does not have access to this LLM Gateway model"}, 30 * time.Minute},
		{"kunci salah", &GatewayError{Status: 401, Reason: "invalid"}, 30 * time.Minute},
		{"kuota harian", &GatewayError{Status: 400, Reason: "Quota exceeded for metric"}, 30 * time.Minute},
		{"server error", &GatewayError{Status: 503}, 30 * time.Second},
		{"model tidak dikenal", &GatewayError{Status: 404, Reason: "model not found"}, 10 * time.Minute},
		{"jawaban bukan JSON", ErrBadVerdict, 0},
	}
	for _, c := range cases {
		if got := restFor(c.err); got != c.want {
			t.Errorf("%s: istirahat %v, mau %v", c.name, got, c.want)
		}
	}
}

// Jawaban yang meleset bukan tanda penyedianya rusak: ucapan ini dicoba ke
// penyedia lain, tetapi ucapan berikutnya tetap mulai dari prioritas utama.
func TestJawabanNgawurTidakMenghukumPenyedia(t *testing.T) {
	gemini := newFake(t, func(w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Tentu! Kalimat itu memenuhi."}}]}`))
	})
	groq := newFake(t, ok)
	f, _ := failover(gemini.matcher("gemini", true), groq.matcher("groq", true))

	if matched, _, err := f.Match(context.Background(), "a", identity); err != nil || !matched {
		t.Fatalf("Match() = %v %v, mau dinilai groq", matched, err)
	}
	_, _, _ = f.Match(context.Background(), "b", identity)
	if gemini.calls.Load() != 2 {
		t.Fatalf("gemini dipanggil %d kali, mau 2 — tidak boleh diistirahatkan", gemini.calls.Load())
	}
}

func TestSemuaGagalMenyebutSetiapPenyebab(t *testing.T) {
	a := newFake(t, status(400, `{"metadata":{"errors":["Your account does not have access to this LLM Gateway model"]}}`))
	b := newFake(t, status(429, `{"error":{"message":"rate limit reached"}}`))
	f, _ := failover(a.matcher("assemblyai", false), b.matcher("groq", true))

	_, _, err := f.Match(context.Background(), "x", identity)
	if err == nil || !strings.Contains(err.Error(), "does not have access") || !strings.Contains(err.Error(), "rate limit reached") {
		t.Fatalf("err = %v, mau menyebut penyebab kedua penyedia", err)
	}
	// Keduanya kini beristirahat: tidak ada panggilan jaringan sama sekali.
	_, _, err = f.Match(context.Background(), "y", identity)
	if a.calls.Load() != 1 || b.calls.Load() != 1 || err == nil || !strings.Contains(err.Error(), "diistirahatkan") {
		t.Fatalf("panggilan a=%d b=%d err=%v, mau tanpa panggilan baru", a.calls.Load(), b.calls.Load(), err)
	}
}

func TestPingAllMengistirahatkanYangGagal(t *testing.T) {
	a := newFake(t, status(400, `{"metadata":{"errors":["Your account does not have access to this LLM Gateway model"]}}`))
	b := newFake(t, ok)
	f, _ := failover(a.matcher("assemblyai", false), b.matcher("gemini", true))

	st := f.PingAll(context.Background())
	if st[0].Problem == "" || st[1].Problem != "" {
		t.Fatalf("status = %+v, mau assemblyai gagal dan gemini siap", st)
	}
	_, _, _ = f.Match(context.Background(), "x", identity)
	if a.calls.Load() != 1 { // hanya dari Ping
		t.Fatalf("assemblyai dicoba lagi padahal sudah gagal saat startup (panggilan=%d)", a.calls.Load())
	}
}

func TestGayaHeaderAuthorizationPerPenyedia(t *testing.T) {
	a := newFake(t, ok)
	g := newFake(t, ok)
	_, _, _ = a.matcher("assemblyai", false).Match(context.Background(), "x", identity)
	_, _, _ = g.matcher("gemini", true).Match(context.Background(), "x", identity)
	if got := a.auth.Load(); got != "kunci-assemblyai" {
		t.Errorf("AssemblyAI Authorization = %q, mau kunci apa adanya", got)
	}
	if got := g.auth.Load(); got != "Bearer kunci-gemini" {
		t.Errorf("Gemini Authorization = %q, mau Bearer", got)
	}
}

func TestPresetPenyedia(t *testing.T) {
	for _, name := range []string{"assemblyai", "gemini", "groq", "openrouter", "mistral", "cerebras"} {
		m, err := NewProvider(name, name, "k", "")
		if err != nil || m.Model() == "" {
			t.Fatalf("NewProvider(%s) = %v, %v", name, m, err)
		}
	}
	// GitHub Models ditutup 30 Juli 2026 — sengaja tidak ada presetnya.
	if _, err := NewProvider("github", "github", "k", ""); err == nil {
		t.Fatal("penyedia tak dikenal harus ditolak")
	}
}

func TestPenyebabDariErrorGayaOpenAI(t *testing.T) {
	got := gatewayReason([]byte(`{"error":{"message":"Invalid API Key","type":"invalid_request_error"}}`))
	if got != "Invalid API Key" {
		t.Fatalf("gatewayReason = %q, mau Invalid API Key", got)
	}
}

// Bentuk error Gemini yang sebenarnya (29 Sep 2026): array berisi satu objek.
func TestPenyebabDariErrorGeminiBerbentukArray(t *testing.T) {
	raw := []byte(`[{"error":{"code":404,"message":"This model models/gemini-2.5-flash-lite is no longer available to new users.","status":"NOT_FOUND"}}]`)
	if got := gatewayReason(raw); got != "This model models/gemini-2.5-flash-lite is no longer available to new users." {
		t.Fatalf("gatewayReason = %q", got)
	}
}
