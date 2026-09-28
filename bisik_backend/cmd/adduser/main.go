// Command adduser menambah satu akun petugas atau supervisor ke database
// yang sudah berjalan.
//
// Akun awal dari SEED_* hanya dibuat saat tabel users masih kosong, jadi
// akun berikutnya dibuat lewat perintah ini — memakai hasher dan aturan yang
// sama dengan login, bukan INSERT manual yang harus menebak format hash.
//
// Kata sandi dibaca dari BISIK_NEW_PASSWORD, bukan dari flag, supaya tidak
// tertinggal di riwayat shell. Kalau kosong, kata sandi acak dibuat dan
// dicetak sekali.
//
//	make add-user USERNAME=handoko NAME="Handoko"
//	BISIK_NEW_PASSWORD=... go run ./cmd/adduser -username handoko -name Handoko
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/bisik/bisik_backend/internal/adapter/repository/postgres"
	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/infrastructure/auth"
	"github.com/bisik/bisik_backend/internal/infrastructure/config"
	"github.com/bisik/bisik_backend/internal/infrastructure/db"
	"github.com/bisik/bisik_backend/internal/usecase"
)

// Tanpa huruf yang mudah tertukar saat dibacakan: 0/O, 1/l/I.
const passwordAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func main() {
	username := flag.String("username", "", "username untuk login (wajib)")
	name := flag.String("name", "", "nama yang tampil di aplikasi dan laporan (wajib)")
	role := flag.String("role", string(domain.RoleOfficer), "officer atau supervisor")
	flag.Parse()

	password := os.Getenv("BISIK_NEW_PASSWORD")
	generated := password == ""
	if generated {
		password = randomPassword(12)
	}

	ctx := context.Background()
	cfg := config.Load()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("koneksi database gagal: %v", err)
	}
	defer pool.Close()
	// Database yang benar-benar baru belum punya tabel users.
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrasi gagal: %v", err)
	}

	// Signer tidak dipakai: CreateUser tidak menerbitkan token.
	uc := usecase.NewAuthUsecase(postgres.NewUserRepo(pool), auth.PasswordHasher{}, nil)
	user, err := uc.CreateUser(ctx, usecase.SeedAccount{
		Username: *username,
		Name:     *name,
		Role:     domain.Role(*role),
		Password: password,
	})
	if err != nil {
		log.Fatalf("akun gagal dibuat: %v", err)
	}

	fmt.Printf("akun dibuat\n  username : %s\n  nama     : %s\n  peran    : %s\n", user.Username, user.Name, user.Role)
	if generated {
		fmt.Printf("  sandi    : %s   (dibuat acak, catat sekarang — tidak ditampilkan lagi)\n", password)
	}
}

func randomPassword(n int) string {
	out := make([]byte, n)
	max := big.NewInt(int64(len(passwordAlphabet)))
	for i := range out {
		k, err := rand.Int(rand.Reader, max)
		if err != nil {
			log.Fatalf("sumber acak gagal: %v", err)
		}
		out[i] = passwordAlphabet[k.Int64()]
	}
	return string(out)
}
