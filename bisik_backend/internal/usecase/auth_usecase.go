package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/google/uuid"
)

// PasswordHasher memisahkan use case dari pilihan algoritma kata sandi.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encoded string) bool
}

// TokenSigner menerbitkan token bearer.
type TokenSigner interface {
	Sign(user *domain.User) (token string, expiry time.Time, err error)
}

// TokenClaims adalah isi token yang sudah terbukti sah.
type TokenClaims struct {
	UserID string
	Name   string
	Role   domain.Role
}

// TokenVerifier memeriksa token bearer. Kontraknya diletakkan di sini, bukan
// di infrastructure, supaya adapter HTTP dan WebSocket cukup bergantung pada
// use case dan tidak perlu tahu token itu JWT, HMAC, atau apa pun.
type TokenVerifier interface {
	VerifyToken(raw string) (*TokenClaims, error)
}

type AuthUsecase struct {
	users  domain.UserRepository
	hasher PasswordHasher
	signer TokenSigner
}

func NewAuthUsecase(u domain.UserRepository, h PasswordHasher, s TokenSigner) *AuthUsecase {
	return &AuthUsecase{users: u, hasher: h, signer: s}
}

type LoginResult struct {
	Token  string
	Expiry time.Time
	User   *domain.User
}

// Login memeriksa kredensial dan menerbitkan token.
//
// Username yang tidak ada dan kata sandi yang salah menghasilkan error yang
// sama persis. Membedakan keduanya akan memberi tahu penyerang akun mana yang
// benar-benar ada.
func (uc *AuthUsecase) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, domain.ErrUnauthenticated
	}

	user, err := uc.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthenticated
		}
		return nil, err
	}
	if !uc.hasher.Verify(password, user.PasswordHash) {
		return nil, domain.ErrUnauthenticated
	}

	token, expiry, err := uc.signer.Sign(user)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, Expiry: expiry, User: user}, nil
}

// MinPasswordLength berlaku untuk akun yang dibuat lewat CreateUser.
const MinPasswordLength = 8

// CreateUser menambah satu akun ke database yang sudah berjalan.
//
// Seed hanya bekerja saat tabel users kosong, jadi akun berikutnya — petugas
// baru, misalnya — lewat jalur ini. Username dicek tanpa membedakan huruf
// besar-kecil, sama seperti saat login: "Handoko" dan "handoko" adalah akun
// yang sama.
func (uc *AuthUsecase) CreateUser(ctx context.Context, account SeedAccount) (*domain.User, error) {
	account.Username = strings.TrimSpace(account.Username)
	account.Name = strings.TrimSpace(account.Name)
	if account.Username == "" || account.Name == "" {
		return nil, fmt.Errorf("%w: username dan nama wajib diisi", domain.ErrInvalidInput)
	}
	if !account.Role.Valid() {
		return nil, fmt.Errorf("%w: role %q", domain.ErrInvalidInput, account.Role)
	}
	if len(account.Password) < MinPasswordLength {
		return nil, fmt.Errorf("%w: kata sandi minimal %d karakter", domain.ErrInvalidInput, MinPasswordLength)
	}

	switch _, err := uc.users.FindByUsername(ctx, account.Username); {
	case err == nil:
		return nil, fmt.Errorf("%w: username %q sudah dipakai", domain.ErrConflict, account.Username)
	case !errors.Is(err, domain.ErrNotFound):
		return nil, err
	}

	hash, err := uc.hasher.Hash(account.Password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		ID:           uuid.NewString(),
		Username:     account.Username,
		Name:         account.Name,
		Role:         account.Role,
		PasswordHash: hash,
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// SeedAccount adalah akun awal yang dibuat saat tabel users masih kosong.
type SeedAccount struct {
	Username string
	Name     string
	Role     domain.Role
	Password string
}

// Seed membuat akun awal HANYA ketika belum ada satu pun pengguna.
//
// Sengaja tidak memperbarui apa pun kalau tabelnya sudah terisi: kalau tidak,
// mengubah variabel lingkungan akan diam-diam menimpa kata sandi yang sudah
// dipakai orang, dan kata sandi bawaan akan hidup kembali selamanya.
func (uc *AuthUsecase) Seed(ctx context.Context, accounts []SeedAccount) (int, error) {
	existing, err := uc.users.Count(ctx)
	if err != nil {
		return 0, err
	}
	if existing > 0 {
		return 0, nil
	}

	created := 0
	for _, account := range accounts {
		if account.Username == "" || account.Password == "" {
			continue
		}
		if !account.Role.Valid() {
			return created, fmt.Errorf("%w: role %q", domain.ErrInvalidInput, account.Role)
		}
		hash, err := uc.hasher.Hash(account.Password)
		if err != nil {
			return created, err
		}
		if err := uc.users.Create(ctx, &domain.User{
			ID:           uuid.NewString(),
			Username:     account.Username,
			Name:         account.Name,
			Role:         account.Role,
			PasswordHash: hash,
		}); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}
