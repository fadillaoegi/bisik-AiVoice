package assemblyai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
)

// LLMMatcher memakai AssemblyAI LLM Gateway untuk menilai apakah satu
// ucapan sudah memenuhi satu butir kewajiban.
type LLMMatcher struct {
	apiKey string
	model  string
	url    string
	client *http.Client
}

func NewLLMMatcher(apiKey, model string) *LLMMatcher {
	return newLLMMatcher(apiKey, model, "https://llm-gateway.assemblyai.com/v1/chat/completions")
}

func newLLMMatcher(apiKey, model, endpoint string) *LLMMatcher {
	return &LLMMatcher{
		apiKey: apiKey,
		model:  model,
		url:    endpoint,
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

const matchPrompt = `Kamu adalah auditor kepatuhan. Tugasmu menilai SATU kalimat petugas.

Butir kewajiban: %s
Definisi: %s

Kalimat petugas: "%s"

Jawab HANYA dengan JSON: {"matched": true|false, "confidence": 0.0-1.0}
matched=true hanya jika kalimat itu benar-benar memenuhi butir di atas secara substantif, bukan sekadar menyinggung topiknya.`

func (m *LLMMatcher) Match(ctx context.Context, utterance string, ob domain.Obligation) (bool, float64, error) {
	if m.apiKey == "" {
		return false, 0, nil
	}
	body, _ := json.Marshal(map[string]any{
		"model":      m.model,
		"max_tokens": 64,
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf(matchPrompt, ob.Label, ob.Description, utterance)},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		m.url, bytes.NewReader(body))
	if err != nil {
		return false, 0, err
	}
	// AssemblyAI memakai API key apa adanya, tanpa prefix "Bearer".
	req.Header.Set("Authorization", m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return false, 0, fmt.Errorf("LLM Gateway HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, 0, fmt.Errorf("decode respons LLM Gateway: %w", err)
	}
	if len(out.Choices) == 0 {
		return false, 0, fmt.Errorf("respons LLM Gateway tidak memiliki choices")
	}

	var verdict struct {
		Matched    bool    `json:"matched"`
		Confidence float64 `json:"confidence"`
	}
	content := strings.TrimSpace(out.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(strings.TrimPrefix(content, "```"), "```")
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &verdict); err != nil {
		return false, 0, fmt.Errorf("verdict LLM Gateway bukan JSON valid: %w", err)
	}
	return verdict.Matched, verdict.Confidence, nil
}
