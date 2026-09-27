package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/usecase"
)

type contextKey struct{}

var identityKey contextKey

// Identity adalah hasil verifikasi token, ditempelkan ke context request.
type Identity struct {
	UserID string
	Name   string
	Role   domain.Role
}

// IdentityFrom mengambil identitas yang sudah diverifikasi middleware.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}

// WithIdentity dipakai adapter lain — mis. WebSocket — yang memverifikasi
// token lewat jalurnya sendiri tetapi tetap ingin memakai helper yang sama.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// TokenVerifier di-alias dari use case supaya adapter ini tidak perlu
// mengimpor paket infrastructure mana pun.
type TokenVerifier = usecase.TokenVerifier

// requireAuth menolak request tanpa token yang sah. `roles` kosong berarti
// peran apa pun boleh, asalkan sudah login.
func requireAuth(v TokenVerifier, roles ...domain.Role) func(http.HandlerFunc) http.HandlerFunc {
	allowed := make(map[domain.Role]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token, err := v.VerifyToken(bearerToken(r))
			if err != nil {
				writeError(w, http.StatusUnauthorized, domain.ErrUnauthenticated)
				return
			}
			identity := Identity{
				UserID: token.UserID,
				Name:   token.Name,
				Role:   token.Role,
			}
			if len(allowed) > 0 && !allowed[identity.Role] {
				writeError(w, http.StatusForbidden, domain.ErrForbidden)
				return
			}
			next(w, r.WithContext(WithIdentity(r.Context(), identity)))
		}
	}
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if value, found := strings.CutPrefix(header, "Bearer "); found {
		return strings.TrimSpace(value)
	}
	return ""
}
