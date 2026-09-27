package main

import (
	"context"
	"log"

	"github.com/bisik/bisik_backend/internal/infrastructure/config"
	"github.com/bisik/bisik_backend/internal/infrastructure/db"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("koneksi database gagal: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrasi gagal: %v", err)
	}
	log.Println("migrasi database selesai")
}
