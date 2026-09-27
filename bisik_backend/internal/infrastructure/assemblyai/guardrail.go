package assemblyai

import (
	"context"
	"strings"

	"github.com/bisik/bisik_backend/internal/domain"
)

// PhraseGuard adalah gate deterministik untuk janji terlarang.
//
// Sengaja deterministik, bukan LLM: pelanggaran kepatuhan tidak boleh
// bergantung pada sampling model. Ini juga membuat demo reproducible.
type PhraseGuard struct{ phrases []string }

func NewPhraseGuard() *PhraseGuard {
	return &PhraseGuard{phrases: domain.ForbiddenPhrases()}
}

func (g *PhraseGuard) Inspect(_ context.Context, utterance string) (string, string, bool, error) {
	low := strings.ToLower(utterance)
	for _, p := range g.phrases {
		if strings.Contains(low, p) {
			return p, "high", true, nil
		}
	}
	return "", "", false, nil
}
