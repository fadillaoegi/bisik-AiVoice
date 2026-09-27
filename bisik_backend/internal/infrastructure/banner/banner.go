// Package banner mencetak ringkasan konfigurasi saat backend mulai.
//
// Bukan hiasan. Selama pengembangan berulang kali muncul kebingungan model
// mana yang sebenarnya aktif — dan gejala "diarization tidak jalan" ternyata
// berasal dari model yang salah. Banner ini menjawabnya sebelum sesi pertama
// dibuka, tanpa perlu membaca `.env` atau menyaring log.
package banner

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Warna merek Bisik, sama dengan token di frontend.
const (
	mint  = "\x1b[38;2;110;231;183m"
	amber = "\x1b[38;2;251;191;36m"
	muted = "\x1b[38;2;139;163;156m"
	bold  = "\x1b[1m"
	reset = "\x1b[0m"
)

// Config adalah apa yang perlu dilihat orang saat backend mulai.
type Config struct {
	Port            string
	TranscriptModel string
	DiarizerModel   string // kosong = mode satu stream
	LLMModel        string
	StaticDir       string
	AuthConfigured  bool
}

// Balon percakapan dengan gelombang suara di dalamnya — bentuk yang sama
// dengan ikon aplikasi, dibuat ulang seadanya dalam karakter kotak.
var mark = []string{
	"╭───────────────╮",
	"│ ▁  ▄  █  ▅  ▂ │",
	"│               │",
	"╰────╮   ╭──────╯",
	"     ╰───╯       ",
}

// providerMark memilih tanda yang sesuai dengan penyedia model.
//
// Sengaja lambang sederhana, bukan tiruan logo: yang dibutuhkan pembaca
// adalah membedakan penyedia dalam sekali lihat, bukan replika merek.
func providerMark(model string) (string, string) {
	switch {
	case strings.Contains(model, "claude"):
		return "✳", "Anthropic"
	case model == "":
		return " ", ""
	default:
		return "◭", "AssemblyAI"
	}
}

// Print menulis banner. Warna hanya dipakai kalau tujuannya terminal —
// di log Render atau berkas, kode ANSI cuma jadi sampah.
func Print(w io.Writer, cfg Config) {
	color := isTerminal(w)
	paint := func(code, text string) string {
		if !color {
			return text
		}
		return code + text + reset
	}

	var b strings.Builder
	b.WriteString("\n")

	rows := []string{
		paint(bold, "BISIK"),
		paint(muted, "Kopilot kepatuhan petugas lapangan"),
		"",
		paint(muted, "MODEL AKTIF"),
		modelLine(paint, cfg.TranscriptModel, "transkrip"),
	}
	if cfg.DiarizerModel != "" {
		rows = append(rows, modelLine(paint, cfg.DiarizerModel, "label pembicara"))
	}
	rows = append(rows, modelLine(paint, cfg.LLMModel, "semantic match"))

	// Lebar dihitung per RUNE, bukan byte: karakter kotak multi-byte, dan
	// %-17s di Go menghitung byte sehingga kolomnya berantakan.
	artWidth := 0
	for _, line := range mark {
		if n := len([]rune(line)); n > artWidth {
			artWidth = n
		}
	}
	pad := func(line string) string {
		return line + strings.Repeat(" ", artWidth-len([]rune(line)))
	}

	lines := max(len(mark), len(rows))
	for i := range lines {
		left, right := strings.Repeat(" ", artWidth), ""
		if i < len(mark) {
			left = paint(mint, pad(mark[i]))
		}
		if i < len(rows) {
			right = rows[i]
		}
		b.WriteString(strings.TrimRight("  "+left+"   "+right, " ") + "\n")
	}

	b.WriteString("\n")
	mode := paint(amber, "satu stream — label pembicara mengikuti model transkrip")
	if cfg.DiarizerModel != "" {
		mode = paint(mint, "dua stream")
	}
	b.WriteString(fmt.Sprintf("  %s  %s · %s\n",
		paint(muted, "gateway "), ":"+cfg.Port, mode))

	auth := paint(amber, "AUTH_SECRET kosong")
	if cfg.AuthConfigured {
		auth = "auth aktif"
	}
	b.WriteString(fmt.Sprintf("  %s  %s\n", paint(muted, "keamanan"), auth))

	static := paint(muted, "tidak menyajikan frontend")
	if cfg.StaticDir != "" {
		static = cfg.StaticDir
	}
	b.WriteString(fmt.Sprintf("  %s  %s\n\n", paint(muted, "statis  "), static))

	_, _ = io.WriteString(w, b.String())
}

func modelLine(paint func(string, string) string, model, role string) string {
	if model == "" {
		return ""
	}
	glyph, provider := providerMark(model)
	return fmt.Sprintf("%s %s %s %s",
		paint(mint, glyph),
		paint(muted, provider+" ·"),
		model,
		paint(muted, "· "+role))
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
