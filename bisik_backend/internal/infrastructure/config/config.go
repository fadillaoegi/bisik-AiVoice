package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                  string
	DatabaseURL           string
	AssemblyAIKey         string
	AssemblyAIWSURL       string
	AssemblyAISpeechModel string
	// Model kedua khusus diarization. Dikosongkan = mode satu stream.
	AssemblyAIDiarizerModel      string
	AssemblyAIRevisionIntervalMS int
	LLMModel                     string
	// LLMProviders: urutan penilai semantik, dipisah koma. Yang pertama
	// dipakai dulu; sisanya cadangan saat kuota habis atau gagal.
	LLMProviders string
	// Kunci boleh lebih dari satu per penyedia, dipisah koma.
	GeminiAPIKey     string
	GeminiModel      string
	GroqAPIKey       string
	GroqModel        string
	OpenRouterAPIKey string
	OpenRouterModel  string
	MistralAPIKey    string
	MistralModel     string
	CerebrasAPIKey   string
	CerebrasModel    string
	NudgeIntervalS   int
	AllowedOrigins   string
	StaticDir        string

	// AuthSecret menandatangani token bearer. Mengubahnya membuat seluruh
	// token yang beredar tidak berlaku — itu memang cara mencabut semuanya.
	AuthSecret string
	AuthTTLH   int

	// Akun awal, hanya dibuat saat tabel users masih kosong.
	SeedOfficerUser      string
	SeedOfficerName      string
	SeedOfficerPassword  string
	SeedSupervisorUser   string
	SeedSupervisorName   string
	SeedSupervisorPasswd string
}

func Load() Config {
	return Config{
		Port:                         env("PORT", "8080"),
		DatabaseURL:                  env("DATABASE_URL", "postgres://bisik:bisik@localhost:5432/bisik?sslmode=disable"),
		AssemblyAIKey:                env("ASSEMBLYAI_API_KEY", ""),
		AssemblyAIWSURL:              env("ASSEMBLYAI_WS_URL", "wss://streaming.assemblyai.com/v3/ws"),
		AssemblyAISpeechModel:        env("ASSEMBLYAI_SPEECH_MODEL", "whisper-rt"),
		AssemblyAIDiarizerModel:      env("ASSEMBLYAI_DIARIZER_MODEL", "universal-streaming-multilingual"),
		AssemblyAIRevisionIntervalMS: envInt("ASSEMBLYAI_SPEAKER_REVISION_INTERVAL_MS", 120000),
		LLMModel:                     env("LLM_MODEL", "claude-sonnet-4-6"),
		// Bawaan tetap AssemblyAI saja, sama dengan perilaku sebelum ada
		// cadangan — .env lama tidak berubah artinya.
		LLMProviders:     env("LLM_PROVIDERS", "assemblyai"),
		GeminiAPIKey:     env("GEMINI_API_KEY", ""),
		GeminiModel:      env("GEMINI_MODEL", ""),
		GroqAPIKey:       env("GROQ_API_KEY", ""),
		GroqModel:        env("GROQ_MODEL", ""),
		OpenRouterAPIKey: env("OPENROUTER_API_KEY", ""),
		OpenRouterModel:  env("OPENROUTER_MODEL", ""),
		MistralAPIKey:    env("MISTRAL_API_KEY", ""),
		MistralModel:     env("MISTRAL_MODEL", ""),
		CerebrasAPIKey:   env("CEREBRAS_API_KEY", ""),
		CerebrasModel:    env("CEREBRAS_MODEL", ""),
		NudgeIntervalS:   envInt("NUDGE_INTERVAL_SECONDS", 45),
		AllowedOrigins:   env("ALLOWED_ORIGINS", "http://localhost:5173"),
		StaticDir:        env("STATIC_DIR", ""),

		AuthSecret: env("AUTH_SECRET", ""),
		AuthTTLH:   envInt("AUTH_TOKEN_TTL_HOURS", 12),

		SeedOfficerUser:      env("SEED_OFFICER_USERNAME", "petugas"),
		SeedOfficerName:      env("SEED_OFFICER_NAME", "Rina Petugas"),
		SeedOfficerPassword:  env("SEED_OFFICER_PASSWORD", ""),
		SeedSupervisorUser:   env("SEED_SUPERVISOR_USERNAME", "supervisor"),
		SeedSupervisorName:   env("SEED_SUPERVISOR_NAME", "Budi Supervisor"),
		SeedSupervisorPasswd: env("SEED_SUPERVISOR_PASSWORD", ""),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// LLMKeys memetakan nama penyedia penilai ke kuncinya (boleh dipisah koma).
func (c Config) LLMKeys() map[string]string {
	return map[string]string{
		"assemblyai": c.AssemblyAIKey,
		"gemini":     c.GeminiAPIKey,
		"groq":       c.GroqAPIKey,
		"openrouter": c.OpenRouterAPIKey,
		"mistral":    c.MistralAPIKey,
		"cerebras":   c.CerebrasAPIKey,
	}
}

// LLMModels memetakan nama penyedia penilai ke modelnya; kosong = bawaan.
func (c Config) LLMModels() map[string]string {
	return map[string]string{
		"assemblyai": c.LLMModel,
		"gemini":     c.GeminiModel,
		"groq":       c.GroqModel,
		"openrouter": c.OpenRouterModel,
		"mistral":    c.MistralModel,
		"cerebras":   c.CerebrasModel,
	}
}
