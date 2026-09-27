package usecase

import (
	"context"

	"github.com/bisik/bisik_backend/internal/domain"
)

// SocketAuthorizer memutuskan siapa boleh menyambung ke sebuah sesi realtime
// dan dengan peran apa.
//
// Peran sengaja dikembalikan dari sini, bukan diterima dari klien. Sebelum
// ada lapisan ini, klien menyebutkan sendiri `role=officer` di query — yang
// berarti siapa pun yang mengetahui ID sesi bisa mengaku petugas lalu
// mendorong audio ke percakapan orang lain.
type SocketAuthorizer struct {
	tokens   TokenVerifier
	sessions domain.SessionRepository
}

func NewSocketAuthorizer(t TokenVerifier, s domain.SessionRepository) *SocketAuthorizer {
	return &SocketAuthorizer{tokens: t, sessions: s}
}

func (a *SocketAuthorizer) AuthorizeSocket(
	ctx context.Context, token, sessionID string,
) (domain.Role, error) {
	claims, err := a.tokens.VerifyToken(token)
	if err != nil {
		return "", domain.ErrUnauthenticated
	}

	// Supervisor boleh menyimak sesi mana pun. Dia tidak pernah mengirim
	// audio: adapter WebSocket hanya menerima frame biner dari role officer.
	if claims.Role == domain.RoleSupervisor {
		return domain.RoleSupervisor, nil
	}

	session, err := a.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.OfficerID != claims.UserID {
		return "", domain.ErrForbidden
	}
	return domain.RoleOfficer, nil
}
