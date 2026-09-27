package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/bisik/bisik_backend/internal/domain"
)

// Parameter PBKDF2-HMAC-SHA256 mengikuti rekomendasi OWASP.
//
// PBKDF2 dipilih karena ada di pustaka standar Go 1.24+, jadi tidak perlu
// dependensi tambahan. Argon2id sebenarnya lebih disukai untuk kata sandi,
// tetapi hanya tersedia lewat golang.org/x/crypto.
const (
	pbkdf2Iterations = 600_000
	pbkdf2KeyLength  = 32
	saltLength       = 16
	hashPrefix       = "pbkdf2-sha256"
)

// HashPassword mengembalikan hash berformat
// `pbkdf2-sha256$<iterasi>$<salt-b64>$<turunan-b64>`.
//
// Jumlah iterasi ikut disimpan supaya hash lama tetap bisa diverifikasi
// ketika parameternya dinaikkan di kemudian hari.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("%w: kata sandi kosong", domain.ErrInvalidInput)
	}
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, pbkdf2KeyLength)
	if err != nil {
		return "", err
	}
	return strings.Join([]string{
		hashPrefix,
		strconv.Itoa(pbkdf2Iterations),
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	}, "$"), nil
}

// VerifyPassword membandingkan kata sandi dengan hash tersimpan.
//
// Perbandingannya memakai waktu tetap supaya lama proses tidak membocorkan
// seberapa banyak byte awal yang sudah cocok.
func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != hashPrefix {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

// PasswordHasher mengadaptasi fungsi di atas ke kontrak usecase.PasswordHasher.
type PasswordHasher struct{}

func (PasswordHasher) Hash(password string) (string, error) { return HashPassword(password) }

func (PasswordHasher) Verify(password, encoded string) bool {
	return VerifyPassword(password, encoded)
}
