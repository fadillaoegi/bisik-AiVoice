package db

import (
	"context"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationFiles ikut dikompilasi ke binary agar database produksi baru
// tidak bergantung pada volume/init script milik Docker Compose.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// Nilai legacy dipertahankan agar instance sebelum dan sesudah rename tetap
// memakai advisory lock migrasi yang sama.
const migrationLockID int64 = 724_253_747

// Migrate menjalankan migrasi yang belum tercatat menurut urutan nama.
// Setiap file memakai transaksi sendiri, checksum, dan advisory lock supaya
// aman ketika beberapa instance aplikasi mulai pada waktu bersamaan.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("ambil koneksi migrasi: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("kunci migrasi: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, migrationLockID)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			checksum   TEXT        NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("buat schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("baca migrasi: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		body, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("baca %s: %w", entry.Name(), err)
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(body))

		var storedChecksum string
		err = conn.QueryRow(ctx,
			`SELECT checksum FROM schema_migrations WHERE version=$1`, entry.Name(),
		).Scan(&storedChecksum)
		switch {
		case err == nil:
			if storedChecksum != checksum {
				return fmt.Errorf("migrasi %s berubah setelah diterapkan", entry.Name())
			}
			continue
		case !errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("cek versi %s: %w", entry.Name(), err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("mulai transaksi %s: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("jalankan %s: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version, checksum) VALUES ($1,$2)`,
			entry.Name(), checksum,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("catat %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", entry.Name(), err)
		}
	}
	return nil
}
