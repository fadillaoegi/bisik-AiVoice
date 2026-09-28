package main

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/bisik/bisik_backend/internal/infrastructure/config"
)

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// Urutan LLM_PROVIDERS dipertahankan, kunci ganda diberi nomor, penyedia
// tanpa kunci atau tak dikenal dilewati tanpa menggagalkan startup.
func TestNewScorerMenyusunUrutanDanKunci(t *testing.T) {
	cfg := config.Config{
		LLMProviders:  " gemini , groq,openai, assemblyai",
		GeminiAPIKey:  "g1, g2",
		GroqAPIKey:    "",
		AssemblyAIKey: "a1",
		LLMModel:      "claude-sonnet-4-6",
	}
	st := newScorer(cfg, quietLog()).PingAll(canceled())

	var names []string
	for _, s := range st {
		names = append(names, s.Name+"="+s.Model)
	}
	want := "gemini#1=gemini-3.6-flash,gemini#2=gemini-3.6-flash,assemblyai=claude-sonnet-4-6"
	if got := strings.Join(names, ","); got != want {
		t.Fatalf("penyedia = %s\nmau        %s", got, want)
	}
}

func TestBawaanTetapAssemblyAISaja(t *testing.T) {
	cfg := config.Config{LLMProviders: "assemblyai", AssemblyAIKey: "a1", LLMModel: "claude-sonnet-4-6"}
	if st := newScorer(cfg, quietLog()).PingAll(canceled()); len(st) != 1 || st[0].Name != "assemblyai" {
		t.Fatalf("status = %+v, mau hanya assemblyai", st)
	}
}

// Ping dengan context yang sudah dibatalkan: tidak ada panggilan jaringan
// sungguhan dari test, yang diuji hanya susunan penyedianya.
func canceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
