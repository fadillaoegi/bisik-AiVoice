package assemblyai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
)

// LLMMatcher menilai apakah satu ucapan sudah memenuhi satu butir kewajiban,
// lewat satu penyedia LLM yang memakai format chat-completions ala OpenAI.
//
// Awalnya hanya AssemblyAI LLM Gateway. Gemini dan Groq ditambahkan
// 29 Sep 2026 karena LLM Gateway tidak tersedia untuk akun free tier,
// sedangkan keduanya punya kuota gratis tanpa kartu kredit. Formatnya sama;
// yang berbeda hanya alamat, gaya header Authorization, dan ruang token.
type LLMMatcher struct {
	name      string
	apiKey    string
	model     string
	url       string
	bearer    bool // AssemblyAI memakai kunci apa adanya; lainnya "Bearer <kunci>"
	maxTokens int
	client    *http.Client
}

// Preset penyedia yang didukung. Kunci peta adalah nama di LLM_PROVIDERS.
var providerPresets = map[string]struct {
	url          string
	bearer       bool
	defaultModel string
	// Model penalar (gpt-oss di Groq, Gemini seri thinking) menghabiskan
	// token untuk berpikir sebelum menjawab; 64 token bisa habis sebelum
	// JSON-nya keluar. Biayanya tetap kecil karena jawabannya pendek.
	maxTokens int
}{
	"assemblyai": {"https://llm-gateway.assemblyai.com/v1/chat/completions", false, "claude-sonnet-4-6", 64},
	// gemini-2.5-flash-lite ditolak untuk kunci baru (404, 29 Sep 2026), dan
	// seri 3.5 menggantung hingga 30 detik saat kelebihan beban. 3.6-flash
	// menjawab ±2 detik dan, saat penuh, MENOLAK cepat (503/429) — cadangan
	// bisa langsung mengambil alih alih-alih menunggu timeout.
	"gemini": {"https://generativelanguage.googleapis.com/v1beta/openai/chat/completions", true, "gemini-3.6-flash", 512},
	"groq":   {"https://api.groq.com/openai/v1/chat/completions", true, "openai/gpt-oss-120b", 1024},
	// OpenRouter: model berakhiran ":free" gratis tanpa kartu (20/menit,
	// 50/hari). Kapasitas tiap model gratis bisa penuh → 429 → cadangan.
	"openrouter": {"https://openrouter.ai/api/v1/chat/completions", true, "google/gemma-4-31b-it:free", 512},
	// Mistral paket Experiment: gratis, verifikasi nomor HP, data boleh
	// dipakai Mistral untuk melatih model.
	"mistral":  {"https://api.mistral.ai/v1/chat/completions", true, "mistral-small-latest", 256},
	"cerebras": {"https://api.cerebras.ai/v1/chat/completions", true, "gpt-oss-120b", 1024},
}

// KnownProvider melaporkan apakah nama penyedia dikenali.
func KnownProvider(name string) bool {
	_, ok := providerPresets[name]
	return ok
}

// NewProvider membuat penilai untuk satu penyedia. label membedakan kunci
// yang berbeda pada penyedia yang sama (mis. "gemini#2"); model kosong
// berarti model bawaan penyedia.
func NewProvider(provider, label, apiKey, model string) (*LLMMatcher, error) {
	preset, ok := providerPresets[provider]
	if !ok {
		return nil, fmt.Errorf("penyedia LLM %q tidak dikenal", provider)
	}
	if model == "" {
		model = preset.defaultModel
	}
	m := newLLMMatcher(apiKey, model, preset.url)
	m.name, m.bearer, m.maxTokens = label, preset.bearer, preset.maxTokens
	return m, nil
}

// NewLLMMatcher mempertahankan perilaku lama: satu penilai AssemblyAI.
func NewLLMMatcher(apiKey, model string) *LLMMatcher {
	m, _ := NewProvider("assemblyai", "assemblyai", apiKey, model)
	return m
}

func newLLMMatcher(apiKey, model, endpoint string) *LLMMatcher {
	return &LLMMatcher{
		name:      "assemblyai",
		apiKey:    apiKey,
		model:     model,
		url:       endpoint,
		maxTokens: 64,
		// Ucapan dinilai di jalur transkrip; kalau satu penyedia menggantung,
		// penyedia berikutnya harus sempat dicoba sebelum petugas menunggu.
		client: &http.Client{Timeout: 6 * time.Second},
	}
}

// Name adalah label penyedia untuk log dan banner.
func (m *LLMMatcher) Name() string { return m.name }

// Model adalah model yang dipakai penyedia ini.
func (m *LLMMatcher) Model() string { return m.model }

// GatewayError adalah respons non-2xx dari penyedia. Jenis statusnya
// menentukan berapa lama penyedia ini diistirahatkan oleh FailoverMatcher.
type GatewayError struct {
	Provider   string
	Status     int
	Reason     string
	RetryAfter time.Duration
}

func (e *GatewayError) Error() string {
	return fmt.Sprintf("%s HTTP %d: %s", e.Provider, e.Status, e.Reason)
}

// ErrBadVerdict: penyedia menjawab, tetapi bukan JSON putusan yang sah.
var ErrBadVerdict = errors.New("putusan penilai bukan JSON valid")

const matchPrompt = `Kamu adalah auditor kepatuhan. Tugasmu menilai SATU kalimat petugas.

Butir kewajiban: %s
Definisi: %s

Kalimat petugas: "%s"

Jawab HANYA dengan JSON: {"matched": true|false, "confidence": 0.0-1.0}
matched=true hanya jika kalimat itu benar-benar memenuhi butir di atas secara substantif, bukan sekadar menyinggung topiknya.
matched=false jika kalimat itu meniadakan, meremehkan, atau membalik kewajibannya — misalnya "tidak ada denda kalau telat", "tidak bisa dibatalkan", atau menjanjikan tanpa risiko. Kalimat seperti itu justru menyesatkan nasabah.`

// Ping memastikan kunci ini benar-benar boleh memakai model ini.
//
// Dipanggil saat backend menyala. Tanpanya, akun yang kehilangan akses
// tampak sehat — transkripsi tetap jalan — padahal tidak satu pun kewajiban
// bisa terpenuhi, dan satu-satunya gejala adalah pengingat yang terus
// berbunyi.
func (m *LLMMatcher) Ping(ctx context.Context) error {
	_, err := m.complete(ctx, "ok", 1)
	return err
}

func (m *LLMMatcher) Match(ctx context.Context, utterance string, ob domain.Obligation) (bool, float64, error) {
	if m.apiKey == "" {
		return false, 0, nil
	}
	content, err := m.complete(ctx, fmt.Sprintf(matchPrompt, ob.Label, ob.Description, utterance), m.maxTokens)
	if err != nil {
		return false, 0, err
	}

	var verdict struct {
		Matched    bool    `json:"matched"`
		Confidence float64 `json:"confidence"`
	}
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(strings.TrimPrefix(content, "```"), "```")
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &verdict); err != nil {
		return false, 0, fmt.Errorf("%s: %w: %v", m.name, ErrBadVerdict, err)
	}
	return verdict.Matched, verdict.Confidence, nil
}

// complete mengirim satu pesan dan mengembalikan isi jawaban.
func (m *LLMMatcher) complete(ctx context.Context, prompt string, maxTokens int) (string, error) {
	if m.apiKey == "" {
		return "", fmt.Errorf("%s: kunci API kosong", m.name)
	}
	body, _ := json.Marshal(map[string]any{
		"model":      m.model,
		"max_tokens": maxTokens,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	if m.bearer {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	} else {
		// AssemblyAI memakai API key apa adanya, tanpa prefix "Bearer".
		req.Header.Set("Authorization", m.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", m.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", &GatewayError{
			Provider:   m.name,
			Status:     resp.StatusCode,
			Reason:     gatewayReason(raw),
			RetryAfter: retryAfter(resp.Header.Get("Retry-After")),
		}
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("%s: decode respons: %w", m.name, err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("%s: respons tidak memiliki choices", m.name)
	}
	return out.Choices[0].Message.Content, nil
}

// gatewayReason mengambil kalimat penyebab dari badan error.
//
// AssemblyAI menaruhnya di metadata.errors ("message" hanya "invalid
// request body"); penyedia bergaya OpenAI (Gemini, Groq) di error.message.
func gatewayReason(raw []byte) string {
	// Gemini membungkus errornya dalam array: [{"error": {...}}].
	var wrapped []json.RawMessage
	if json.Unmarshal(raw, &wrapped) == nil && len(wrapped) > 0 {
		raw = wrapped[0]
	}
	var body struct {
		Message  string          `json:"message"`
		Error    json.RawMessage `json:"error"`
		Metadata struct {
			Errors []string `json:"errors"`
		} `json:"metadata"`
	}
	if json.Unmarshal(raw, &body) == nil {
		if len(body.Metadata.Errors) > 0 {
			return body.Metadata.Errors[0]
		}
		var nested struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body.Error, &nested) == nil && nested.Message != "" {
			return nested.Message
		}
		var flat string
		if json.Unmarshal(body.Error, &flat) == nil && flat != "" {
			return flat
		}
		if body.Message != "" {
			return body.Message
		}
	}
	return strings.TrimSpace(string(raw))
}

// retryAfter membaca header Retry-After dalam detik; 0 kalau tidak ada.
func retryAfter(v string) time.Duration {
	secs, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}
