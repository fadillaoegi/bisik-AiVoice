package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
)

func TestHashPasswordMenghasilkanSaltBerbeda(t *testing.T) {
	first, err := HashPassword("rahasia-petugas")
	if err != nil {
		t.Fatalf("HashPassword(): %v", err)
	}
	second, err := HashPassword("rahasia-petugas")
	if err != nil {
		t.Fatalf("HashPassword(): %v", err)
	}
	if first == second {
		t.Fatal("dua hash kata sandi sama persis — salt tidak dipakai")
	}
	if !VerifyPassword("rahasia-petugas", first) || !VerifyPassword("rahasia-petugas", second) {
		t.Fatal("hash yang baru dibuat gagal diverifikasi")
	}
}

func TestVerifyPasswordMenolakYangSalahDanRusak(t *testing.T) {
	hash, err := HashPassword("benar")
	if err != nil {
		t.Fatalf("HashPassword(): %v", err)
	}
	if VerifyPassword("salah", hash) {
		t.Fatal("kata sandi salah diterima")
	}
	if VerifyPassword("benar", "") || VerifyPassword("benar", "bukan-format-hash") {
		t.Fatal("hash rusak diterima")
	}
	// Turunan diubah satu karakter: harus gagal, bukan panic.
	tampered := hash[:len(hash)-1] + "A"
	if VerifyPassword("benar", tampered) && tampered != hash {
		t.Fatal("hash yang diubah diterima")
	}
}

func testUser() *domain.User {
	return &domain.User{ID: "u-1", Name: "Rina", Role: domain.RoleOfficer}
}

func TestTokenBolakBalik(t *testing.T) {
	signer := NewSigner("rahasia-uji", time.Hour)
	raw, expiry, err := signer.Sign(testUser())
	if err != nil {
		t.Fatalf("Sign(): %v", err)
	}
	if !expiry.After(time.Now()) {
		t.Fatal("kedaluwarsa sudah lewat saat diterbitkan")
	}
	token, err := signer.Verify(raw)
	if err != nil {
		t.Fatalf("Verify(): %v", err)
	}
	if token.UserID != "u-1" || token.Role != string(domain.RoleOfficer) {
		t.Fatalf("klaim = %+v", token)
	}
}

func TestTokenDitolakSaatTandaTanganTidakCocok(t *testing.T) {
	signer := NewSigner("rahasia-uji", time.Hour)
	raw, _, err := signer.Sign(testUser())
	if err != nil {
		t.Fatalf("Sign(): %v", err)
	}

	// Payload diubah tanpa menandatangani ulang — ini persis yang dilakukan
	// penyerang yang ingin menaikkan perannya sendiri.
	body, signature, _ := strings.Cut(raw, ".")
	forged := body[:len(body)-2] + "AA." + signature
	if _, err := signer.Verify(forged); err == nil {
		t.Fatal("token yang diubah diterima")
	}

	// Ditandatangani rahasia lain.
	other := NewSigner("rahasia-lain", time.Hour)
	if _, err := other.Verify(raw); err == nil {
		t.Fatal("token dari rahasia berbeda diterima")
	}
}

func TestTokenKedaluwarsaDitolak(t *testing.T) {
	signer := NewSigner("rahasia-uji", -time.Second)
	raw, _, err := signer.Sign(testUser())
	if err != nil {
		t.Fatalf("Sign(): %v", err)
	}
	if _, err := signer.Verify(raw); err == nil {
		t.Fatal("token kedaluwarsa diterima")
	}
}
