package banner

import (
	"bytes"
	"strings"
	"testing"
)

// Semua baris gambar harus sama lebar dalam RUNE, kalau tidak kolom kanan
// bergeser. Perataan Go (%-17s) menghitung byte, jadi karakter kotak
// multi-byte akan merusaknya tanpa ada yang menyadari.
func TestGambarSemuaBarisSamaLebar(t *testing.T) {
	width := len([]rune(mark[0]))
	for i, line := range mark {
		if n := len([]rune(line)); n != width {
			t.Fatalf("baris %d selebar %d rune, baris 0 selebar %d", i, n, width)
		}
	}
}

func TestBannerMenyebutSemuaModelDanModenya(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, Config{
		Port:            "8080",
		TranscriptModel: "whisper-rt",
		DiarizerModel:   "universal-streaming-multilingual",
		Scorers:         []Scorer{{Name: "assemblyai", Model: "claude-sonnet-4-6"}},
		AuthConfigured:  true,
	})
	out := buf.String()

	for _, want := range []string{
		"whisper-rt", "universal-streaming-multilingual", "claude-sonnet-4-6",
		"AssemblyAI", "Anthropic", "dua stream", "auth aktif", ":8080",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("banner tidak menyebut %q:\n%s", want, out)
		}
	}
}

// Tanpa diarizer, banner harus MENGATAKAN bahwa label pembicara ikut model
// transkrip — itu perbedaan yang menjelaskan kenapa diarization tidak jalan.
func TestBannerMemperingatkanModeSatuStream(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, Config{Port: "8080", TranscriptModel: "whisper-rt"})
	if !strings.Contains(buf.String(), "satu stream") {
		t.Fatalf("mode satu stream tidak disebut:\n%s", buf.String())
	}
}

func TestBannerMemperingatkanAuthSecretKosong(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, Config{Port: "8080", TranscriptModel: "whisper-rt"})
	if !strings.Contains(buf.String(), "AUTH_SECRET kosong") {
		t.Fatalf("peringatan AUTH_SECRET tidak muncul:\n%s", buf.String())
	}
}

// Ke berkas atau log Render, kode ANSI hanya jadi sampah yang menyulitkan
// pencarian. Warna hanya untuk terminal.
func TestTanpaWarnaSaatBukanTerminal(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, Config{Port: "8080", TranscriptModel: "whisper-rt"})
	if strings.Contains(buf.String(), "\x1b[") {
		t.Fatal("kode ANSI bocor ke tujuan non-terminal")
	}
}

// Akun tanpa akses LLM Gateway tetap bisa mentranskrip, jadi dari luar
// semuanya tampak sehat. Banner harus menyebutnya terang-terangan.
func TestBannerMemperingatkanPenilaiTidakBisaDipakai(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, Config{Port: "8080", TranscriptModel: "whisper-rt", Scorers: []Scorer{{
		Name: "assemblyai", Model: "claude-sonnet-4-6",
		Problem: "assemblyai HTTP 400: Your account does not have access to this LLM Gateway model",
	}}})
	out := buf.String()
	if !strings.Contains(out, "TIDAK BISA DIPAKAI") || !strings.Contains(out, "does not have access") {
		t.Fatalf("masalah penilai tidak disebut:\n%s", out)
	}
}

// Sebagian penyedia gagal: banner harus menyebut berapa yang siap, urutan
// cadangannya, dan penyebab yang gagal — tanpa menyatakan penilai mati.
func TestBannerMenyebutPenyediaCadangan(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, Config{Port: "8080", TranscriptModel: "whisper-rt", Scorers: []Scorer{
		{Name: "gemini", Model: "gemini-2.5-flash-lite"},
		{Name: "groq", Model: "openai/gpt-oss-120b"},
		{Name: "assemblyai", Model: "claude-sonnet-4-6", Problem: "assemblyai HTTP 400: no access"},
	}})
	out := buf.String()
	for _, want := range []string{"2 dari 3 siap", "gemini → groq → assemblyai", "✗ assemblyai: assemblyai HTTP 400", "Google", "OpenAI"} {
		if !strings.Contains(out, want) {
			t.Fatalf("banner tidak menyebut %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "TIDAK BISA DIPAKAI") {
		t.Fatalf("banner menyatakan penilai mati padahal dua penyedia siap:\n%s", out)
	}
}
