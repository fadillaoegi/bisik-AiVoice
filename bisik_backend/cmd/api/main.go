package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	adapterhttp "github.com/bisik/bisik_backend/internal/adapter/http"
	"github.com/bisik/bisik_backend/internal/adapter/repository/postgres"
	"github.com/bisik/bisik_backend/internal/adapter/ws"
	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/infrastructure/assemblyai"
	"github.com/bisik/bisik_backend/internal/infrastructure/auth"
	"github.com/bisik/bisik_backend/internal/infrastructure/banner"
	"github.com/bisik/bisik_backend/internal/infrastructure/config"
	"github.com/bisik/bisik_backend/internal/infrastructure/db"
	"github.com/bisik/bisik_backend/internal/infrastructure/logger"
	"github.com/bisik/bisik_backend/internal/usecase"
)

func main() {
	log := logger.New()
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("koneksi database gagal", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Error("migrasi database gagal", "err", err)
		os.Exit(1)
	}

	// AUTH_SECRET tidak punya nilai bawaan dengan sengaja. Rahasia bawaan
	// yang ikut ter-commit berarti siapa pun bisa menandatangani token
	// petugas mana pun — lebih berbahaya daripada gagal start.
	if cfg.AuthSecret == "" {
		log.Error("AUTH_SECRET wajib diisi; jalankan `make auth-secret` untuk membuatnya")
		os.Exit(1)
	}

	// --- Wiring: infrastructure -> usecase -> adapter ---
	sessionRepo := postgres.NewSessionRepo(pool)
	userRepo := postgres.NewUserRepo(pool)
	transcriptRepo := postgres.NewTranscriptRepo(pool)
	complianceRepo := postgres.NewComplianceRepo(pool)

	transcriber := assemblyai.NewStreamingSTT(
		cfg.AssemblyAIKey,
		cfg.AssemblyAIWSURL,
		cfg.AssemblyAISpeechModel,
		cfg.AssemblyAIRevisionIntervalMS,
		log,
	)

	// Dua stream atas audio yang sama. Terbukti 25 Sep lewat cmd/diarprobe:
	// whisper-rt memberi transkrip Indonesia bagus tetapi diarization-nya
	// tidak berfungsi (semua pembicara dilabeli "A"), sedangkan model
	// multilingual memisahkan pembicara dengan benar tetapi transkrip
	// Indonesianya hancur. Tidak ada satu model pun yang memberi keduanya.
	//
	// Dikosongkan lewat ASSEMBLYAI_DIARIZER_MODEL untuk kembali ke satu
	// stream — berguna kalau kuota menipis atau saat menyelisik masalah.
	var stt usecase.SpeechToText = transcriber
	if cfg.AssemblyAIDiarizerModel != "" {
		diarizer := assemblyai.NewStreamingSTT(
			cfg.AssemblyAIKey,
			cfg.AssemblyAIWSURL,
			cfg.AssemblyAIDiarizerModel,
			cfg.AssemblyAIRevisionIntervalMS,
			log,
		)
		stt = assemblyai.NewDualStreamSTT(transcriber, diarizer, log)
		log.Info("mode dua stream aktif",
			"transkrip", cfg.AssemblyAISpeechModel,
			"diarization", cfg.AssemblyAIDiarizerModel)
	} else {
		log.Warn("mode satu stream: diarization mengikuti model transkrip",
			"model", cfg.AssemblyAISpeechModel)
	}
	matcher := assemblyai.NewLLMMatcher(cfg.AssemblyAIKey, cfg.LLMModel)
	guard := assemblyai.NewPhraseGuard()

	signer := auth.NewSigner(cfg.AuthSecret, time.Duration(cfg.AuthTTLH)*time.Hour)
	authUC := usecase.NewAuthUsecase(userRepo, auth.PasswordHasher{}, signer)
	socketAuth := usecase.NewSocketAuthorizer(signer, sessionRepo)

	created, err := authUC.Seed(ctx, []usecase.SeedAccount{
		{Username: cfg.SeedOfficerUser, Name: cfg.SeedOfficerName,
			Role: domain.RoleOfficer, Password: cfg.SeedOfficerPassword},
		{Username: cfg.SeedSupervisorUser, Name: cfg.SeedSupervisorName,
			Role: domain.RoleSupervisor, Password: cfg.SeedSupervisorPasswd},
	})
	if err != nil {
		log.Error("seed akun gagal", "err", err)
		os.Exit(1)
	}
	if created > 0 {
		log.Info("akun awal dibuat", "jumlah", created,
			"petugas", cfg.SeedOfficerUser, "supervisor", cfg.SeedSupervisorUser)
	}

	hub := ws.NewHub(log)
	nudger := ws.NewHubNudger(hub)

	sessionUC := usecase.NewSessionUsecase(sessionRepo, complianceRepo, stt)
	complianceUC := usecase.NewComplianceUsecase(complianceRepo, transcriptRepo, matcher, guard, nudger, hub)
	complianceUC.SetLogger(log)

	// Uji akses penilai sekali saat menyala. Tidak menghentikan server:
	// transkripsi dan demo tetap berguna tanpa penilai, tetapi kegagalannya
	// harus terlihat di banner, bukan baru ketahuan di tengah uji lapangan.
	llmProblem := ""
	pingCtx, cancelPing := context.WithTimeout(ctx, 10*time.Second)
	if err := matcher.Ping(pingCtx); err != nil {
		llmProblem = err.Error()
		complianceUC.SetScorerProblem(llmProblem)
		log.Error("LLM Gateway tidak bisa dipakai; kewajiban tidak akan pernah terpenuhi",
			"model", cfg.LLMModel, "err", err)
	}
	cancelPing()

	wsHandler := ws.NewHandler(hub, stt, complianceUC, socketAuth, cfg.AllowedOrigins,
		time.Duration(cfg.NudgeIntervalS)*time.Second, log)
	httpHandler := adapterhttp.NewHandler(
		sessionUC, sessionRepo, complianceRepo, transcriptRepo, authUC, log)
	router := adapterhttp.NewRouter(
		httpHandler, wsHandler, signer, cfg.AllowedOrigins, cfg.StaticDir, log)

	// Dicetak sebelum server jalan: model mana yang aktif adalah pertanyaan
	// pertama saat sesuatu tidak beres, dan menjawabnya di sini lebih cepat
	// daripada menyaring log atau membuka `.env`.
	banner.Print(os.Stdout, banner.Config{
		Port:            cfg.Port,
		TranscriptModel: cfg.AssemblyAISpeechModel,
		DiarizerModel:   cfg.AssemblyAIDiarizerModel,
		LLMModel:        cfg.LLMModel,
		LLMProblem:      llmProblem,
		StaticDir:       cfg.StaticDir,
		AuthConfigured:  cfg.AuthSecret != "",
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("bisik_backend jalan", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server berhenti", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutdown...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
