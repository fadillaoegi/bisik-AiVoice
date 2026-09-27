package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/usecase"
)

// Token adalah token bearer bertanda tangan: `<payload-b64>.<hmac-b64>`.
//
// Formatnya sengaja dibuat sendiri dan tidak memakai JWT. Yang dibutuhkan di
// sini hanya satu algoritma dan tiga klaim, sedangkan JWT membawa serta
// negosiasi algoritma — sumber kerentanan klasik `alg: none` — beserta satu
// dependensi lagi. Token ini stateless: server tidak menyimpan sesi login.
//
// Konsekuensi yang disadari: token tidak bisa dicabut sebelum kedaluwarsa.
// Untuk produksi, tambahkan daftar cabut atau umur token yang jauh lebih
// pendek plus mekanisme refresh.
type Token struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	Name   string `json:"name"`
	Expiry int64  `json:"exp"`
}

type Signer struct {
	secret []byte
	ttl    time.Duration
}

func NewSigner(secret string, ttl time.Duration) *Signer {
	return &Signer{secret: []byte(secret), ttl: ttl}
}

func (s *Signer) Sign(user *domain.User) (string, time.Time, error) {
	expiry := time.Now().Add(s.ttl).UTC()
	payload, err := json.Marshal(Token{
		UserID: user.ID,
		Role:   string(user.Role),
		Name:   user.Name,
		Expiry: expiry.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	return body + "." + s.sign(body), expiry, nil
}

// Verify menolak token yang tanda tangannya salah atau sudah kedaluwarsa.
func (s *Signer) Verify(raw string) (*Token, error) {
	body, signature, found := strings.Cut(raw, ".")
	if !found {
		return nil, domain.ErrUnauthenticated
	}
	// Tanda tangan diperiksa lebih dulu, sebelum payload di-decode: jangan
	// pernah mengurai data yang belum terbukti berasal dari kita.
	if subtle.ConstantTimeCompare([]byte(signature), []byte(s.sign(body))) != 1 {
		return nil, domain.ErrUnauthenticated
	}
	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, domain.ErrUnauthenticated
	}
	var token Token
	if err := json.Unmarshal(payload, &token); err != nil {
		return nil, domain.ErrUnauthenticated
	}
	if time.Now().UTC().Unix() >= token.Expiry {
		return nil, fmt.Errorf("%w: token kedaluwarsa", domain.ErrUnauthenticated)
	}
	if !domain.Role(token.Role).Valid() {
		return nil, domain.ErrUnauthenticated
	}
	return &token, nil
}

// VerifyToken mengadaptasi Verify ke kontrak usecase.TokenVerifier.
func (s *Signer) VerifyToken(raw string) (*usecase.TokenClaims, error) {
	token, err := s.Verify(raw)
	if err != nil {
		return nil, err
	}
	return &usecase.TokenClaims{
		UserID: token.UserID,
		Name:   token.Name,
		Role:   domain.Role(token.Role),
	}, nil
}

func (s *Signer) sign(body string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
