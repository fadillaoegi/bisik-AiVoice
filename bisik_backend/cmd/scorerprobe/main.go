// Command scorerprobe menguji kualitas setiap penyedia penilai terhadap
// kalimat yang sudah diketahui jawabannya — sebelum penyedia itu dipercaya
// menilai sesi sungguhan.
//
// Dibuat 29 Sep 2026 saat Gemini dan Groq ditambahkan sebagai penilai
// cadangan: keduanya belum pernah diuji menilai kalimat Bahasa Indonesia
// Bisik. Setiap penyedia di LLM_PROVIDERS diuji SENDIRI-SENDIRI (tanpa
// cadangan), supaya kelemahan satu penyedia tidak tertutup penyedia lain.
//
//	go run ./cmd/scorerprobe
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/infrastructure/assemblyai"
	"github.com/bisik/bisik_backend/internal/infrastructure/config"
)

// Ambang yang sama dengan minimumMatchConfidence di usecase.
const threshold = 0.80

type probeCase struct {
	code string
	text string
	want bool
	note string
}

// Kalimat positif = skrip uji Handoko (submission/skrip-uji-dialog-dwibahasa.txt).
// Kalimat negatif lolos evidence gate kata kunci — hanya penilai yang bisa
// menolaknya — jadi inilah ujian yang sebenarnya.
var cases = []probeCase{
	{"IDENTITY", "Selamat pagi, Pak. Perkenalkan, nama saya Handoko dari Bank Nusantara.", true, "skrip"},
	{"RATE", "Suku bunganya 1,2 persen setiap bulan, dengan biaya administrasi seratus ribu rupiah.", true, "skrip"},
	{"TENOR", "Tenornya 12 bulan, dengan cicilan sekitar 953 ribu rupiah per bulan.", true, "skrip"},
	{"PENALTY", "Kalau terlambat membayar, ada denda lima ribu rupiah per hari.", true, "skrip"},
	{"RIGHT", "Bapak berhak menolak atau membatalkan penawaran ini.", true, "skrip"},
	{"IDENTITY", "perkenalkan nama saya handoko dari bank nusantara", true, "tanpa tanda baca, seperti transkrip"},
	{"RATE", "Nanti bunga dan biaya lainnya saya jelaskan belakangan ya, Pak.", false, "menyinggung tanpa angka nyata"},
	{"TENOR", "Suku bunganya 1,2 persen per bulan.", false, "kalimat bunga, bukan tenor"},
	{"PENALTY", "Tenang saja, tidak ada denda kalau telat sehari dua hari.", false, "menyesatkan soal denda"},
	{"RIGHT", "Bapak tidak bisa menolak kalau sudah tanda tangan.", false, "kebalikan dari hak membatalkan"},
	{"IDENTITY", "Nama saya tidak penting, yang penting banknya terpercaya.", false, "menolak menyebut identitas"},
}

func main() {
	cfg := config.Load()
	providers, warnings := assemblyai.BuildProviders(assemblyai.ProviderSpec{
		Order: cfg.LLMProviders, Keys: cfg.LLMKeys(), Models: cfg.LLMModels(),
	})
	for _, w := range warnings {
		fmt.Println("⚠", w)
	}
	if len(providers) == 0 {
		fmt.Println("tidak ada penyedia yang bisa diuji — isi LLM_PROVIDERS dan kuncinya di .env")
		os.Exit(1)
	}

	obligations := map[string]domain.Obligation{}
	for _, ob := range domain.DefaultObligations() {
		obligations[ob.Code] = ob
	}

	exit := 0
	for _, p := range providers {
		fmt.Printf("\n══ %s · %s ══\n", p.Name(), p.Model())
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := p.Ping(ctx)
		cancel()
		if err != nil {
			fmt.Printf("  ✗ tidak bisa dipakai: %v\n", err)
			exit = 1
			continue
		}

		correct := 0
		var total time.Duration
		for _, c := range cases {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			start := time.Now()
			matched, conf, err := p.Match(ctx, c.text, obligations[c.code])
			took := time.Since(start)
			cancel()
			total += took

			got := err == nil && matched && conf >= threshold
			mark := "✓"
			if err != nil {
				mark = "!"
			} else if got != c.want {
				mark = "✗"
			} else {
				correct++
			}
			detail := fmt.Sprintf("matched=%v conf=%.2f", matched, conf)
			if err != nil {
				detail = "error: " + err.Error()
			}
			fmt.Printf("  %s %-8s mau=%-5v %-24s %5dms\n     %q (%s)\n",
				mark, c.code, c.want, detail, took.Milliseconds(), strings.TrimSpace(c.text), c.note)
			// Beri jeda kecil agar kuota per menit free tier tidak habis
			// oleh alat uji ini sendiri.
			time.Sleep(700 * time.Millisecond)
		}
		fmt.Printf("  → benar %d/%d · rata-rata %dms\n", correct, len(cases),
			total.Milliseconds()/int64(len(cases)))
		if correct != len(cases) {
			exit = 1
		}
	}
	os.Exit(exit)
}
