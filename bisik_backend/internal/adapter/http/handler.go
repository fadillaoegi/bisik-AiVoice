package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/usecase"
)

type Handler struct {
	sessions    *usecase.SessionUsecase
	sessionRepo domain.SessionRepository
	compliance  domain.ComplianceRepository
	transcripts domain.TranscriptRepository
	auth        *usecase.AuthUsecase
	log         *slog.Logger
}

func NewHandler(
	s *usecase.SessionUsecase,
	sr domain.SessionRepository,
	c domain.ComplianceRepository,
	t domain.TranscriptRepository,
	a *usecase.AuthUsecase,
	log *slog.Logger,
) *Handler {
	return &Handler{
		sessions: s, sessionRepo: sr, compliance: c,
		transcripts: t, auth: a, log: log,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, domain.ErrInvalidInput)
		return
	}
	result, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     result.Token,
		ExpiresAt: result.Expiry.Format(time.RFC3339),
		User: UserResponse{
			ID: result.User.ID, Username: result.User.Username,
			Name: result.User.Name, Role: string(result.User.Role),
		},
	})
}

// Me mengembalikan identitas pemilik token. Dipakai klien untuk memulihkan
// sesi login setelah refresh tanpa menyimpan data pengguna di perangkat.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	identity, ok := IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, domain.ErrUnauthenticated)
		return
	}
	writeJSON(w, http.StatusOK, UserResponse{
		ID: identity.UserID, Name: identity.Name, Role: string(identity.Role),
	})
}

// authorizeSession memastikan pemanggil berhak atas sesi ini.
//
// Supervisor boleh membaca sesi mana pun; petugas hanya sesinya sendiri.
// Tanpa pemeriksaan ini, siapa pun yang mengetahui ID sesi bisa membaca
// seluruh transkrip percakapan nasabah.
func (h *Handler) authorizeSession(r *http.Request, sessionID string) (*domain.Session, error) {
	identity, ok := IdentityFrom(r.Context())
	if !ok {
		return nil, domain.ErrUnauthenticated
	}
	session, err := h.sessionRepo.FindByID(r.Context(), sessionID)
	if err != nil {
		return nil, err
	}
	if identity.Role == domain.RoleSupervisor || session.OfficerID == identity.UserID {
		return session, nil
	}
	return nil, domain.ErrForbidden
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "bisik_backend"})
}

func (h *Handler) Obligations(w http.ResponseWriter, _ *http.Request) {
	obs := domain.DefaultObligations()
	out := make([]ObligationResponse, 0, len(obs))
	for _, ob := range obs {
		out = append(out, ObligationResponse{Code: ob.Code, Label: ob.Label, Status: string(domain.ObligationPending)})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	var req StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, domain.ErrInvalidInput)
		return
	}
	identity, ok := IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, domain.ErrUnauthenticated)
		return
	}
	// Pemilik sesi selalu pemegang token, tidak pernah nilai dari body.
	s, err := h.sessions.Start(r.Context(), identity.UserID, req.ProductID)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	writeJSON(w, http.StatusCreated, toSessionResponse(s))
}

func (h *Handler) EndSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Hanya petugas pemilik sesi yang boleh mengakhirinya. Supervisor
	// memantau dan membaca laporan, bukan menghentikan pekerjaan orang.
	identity, ok := IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, domain.ErrUnauthenticated)
		return
	}
	session, err := h.sessionRepo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	if session.OfficerID != identity.UserID {
		writeError(w, http.StatusForbidden, domain.ErrForbidden)
		return
	}
	s, err := h.sessions.End(r.Context(), id)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	writeJSON(w, http.StatusOK, toSessionResponse(s))
}

// Report adalah laporan kepatuhan berbukti: tiap butir menunjuk
// utterance mana yang memenuhinya.
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	s, err := h.authorizeSession(r, id)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	states, err := h.compliance.ListStates(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	violations, err := h.compliance.ListViolations(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	utterances, err := h.transcripts.ListBySession(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	labels := map[string]string{}
	for _, ob := range domain.DefaultObligations() {
		labels[ob.Code] = ob.Label
	}

	resp := ReportResponse{Session: toSessionResponse(s)}
	for _, st := range states {
		resp.Obligations = append(resp.Obligations, ObligationResponse{
			Code: st.Code, Label: labels[st.Code], Status: string(st.Status),
			Confidence: st.Confidence, EvidenceID: st.EvidenceID,
		})
	}
	for _, v := range violations {
		resp.Violations = append(resp.Violations, ViolationResponse{
			Phrase: v.Phrase, Severity: v.Severity, EvidenceID: v.EvidenceID,
			DetectedAt: v.DetectedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	for _, u := range utterances {
		resp.Transcript = append(resp.Transcript, UtteranceResponse{
			ID: u.ID, Speaker: string(u.Speaker), SourceSpeaker: u.SourceSpeaker,
			Text: u.Text, StartMS: u.StartMS, Revised: u.Revised,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrSessionEnded):
		return http.StatusConflict
	case errors.Is(err, domain.ErrUnauthenticated):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

// ListSessions memberi supervisor daftar sesi terbaru untuk dipantau.
//
// Tanpa ini supervisor harus menyalin ID sesi dari psql — bisa untuk uji
// coba, tidak bisa untuk dipakai orang.
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.sessionRepo.ListRecent(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, toSessionResponse(s))
	}
	writeJSON(w, http.StatusOK, out)
}
