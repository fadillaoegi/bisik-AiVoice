package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bisik/bisik_backend/internal/domain"
	"github.com/bisik/bisik_backend/internal/usecase"
	"github.com/gorilla/websocket"
)

// SessionAuthorizer memutuskan apakah pemegang token boleh menyambung ke
// sebuah sesi, dan dengan peran apa.
type SessionAuthorizer interface {
	AuthorizeSocket(ctx context.Context, token, sessionID string) (domain.Role, error)
}

type Handler struct {
	hub        *Hub
	stt        usecase.SpeechToText
	compliance *usecase.ComplianceUsecase
	authorizer SessionAuthorizer
	log        *slog.Logger
	upgrader   websocket.Upgrader
	nudgeEvery time.Duration
}

func NewHandler(hub *Hub, stt usecase.SpeechToText, c *usecase.ComplianceUsecase, authorizer SessionAuthorizer, allowedOrigins string, nudgeEvery time.Duration, log *slog.Logger) *Handler {
	origins := strings.Split(allowedOrigins, ",")
	return &Handler{
		hub:        hub,
		stt:        stt,
		compliance: c,
		authorizer: authorizer,
		log:        log,
		nudgeEvery: nudgeEvery,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // klien non-browser (app mobile)
				}
				// Deployment satu-origin tidak perlu mengetahui hostname Render
				// sebelumnya. Origin tetap harus http(s) dan host-nya persis sama.
				if parsed, err := url.Parse(origin); err == nil &&
					(parsed.Scheme == "http" || parsed.Scheme == "https") &&
					strings.EqualFold(parsed.Host, r.Host) {
					return true
				}
				for _, o := range origins {
					if strings.TrimSpace(o) == origin {
						return true
					}
				}
				return false
			},
		},
	}
}

// ServeHTTP menangani GET /ws?session_id=...&role=officer|supervisor
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id wajib", http.StatusBadRequest)
		return
	}

	// Token lewat query, bukan header: browser tidak mengizinkan header
	// kustom pada handshake WebSocket. Konsekuensinya token bisa muncul di
	// access log proxy, jadi umurnya harus pendek.
	//
	// Peran TIDAK diambil dari query. Dulu klien menyebutkan sendiri
	// `role=officer`, artinya siapa pun bisa mengaku petugas dan mendorong
	// audio ke sesi orang lain.
	authorizedRole, err := h.authorizer.AuthorizeSocket(
		r.Context(), r.URL.Query().Get("token"), sessionID)
	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, domain.ErrUnauthenticated) {
			status = http.StatusUnauthorized
		}
		http.Error(w, err.Error(), status)
		return
	}
	role := string(authorizedRole)

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("upgrade gagal", "err", err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 64), role: role}
	h.hub.Join(sessionID, client)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go h.writePump(client)

	// Hanya klien petugas yang mengalirkan audio ke upstream.
	if role == "officer" {
		if err := h.startPipeline(ctx, cancel, sessionID); err != nil {
			h.log.Error("pipeline gagal", "session", sessionID, "err", err)
			// Tanpa pemberitahuan ini UI tetap menampilkan "merekam"
			// walaupun tidak ada transkrip yang akan pernah masuk.
			h.publishPipelineError(sessionID, err)
		} else {
			go h.nudgeLoop(ctx, sessionID)
		}
	}

	h.readPump(ctx, sessionID, client)
	h.hub.Leave(sessionID, client)
}

func (h *Handler) startPipeline(ctx context.Context, cancel context.CancelFunc, sessionID string) error {
	events, err := h.stt.Start(ctx, sessionID)
	if err != nil {
		return err
	}
	go func() {
		// Stream upstream mati = tidak ada lagi transkrip. Nudge berkala
		// harus ikut berhenti, bukan terus membisikkan butir pending.
		defer cancel()
		for ev := range events {
			if ev.Err != nil {
				h.log.Error("stream stt berhenti", "session", sessionID, "err", ev.Err)
				h.publishPipelineError(sessionID, ev.Err)
				// Event error memakai kontrak acknowledgement yang sama dengan
				// transcript. Tanpa ack, readLoop upstream menunggu selamanya,
				// channel tidak pernah ditutup, dan cancel/nudge ikut menggantung.
				if ev.Acknowledge != nil {
					ev.Acknowledge()
				}
				continue
			}
			if err := h.compliance.HandleTranscript(ctx, sessionID, ev); err != nil {
				h.log.Warn("handle transcript gagal", "session", sessionID, "err", err)
			}
			if ev.Acknowledge != nil {
				ev.Acknowledge()
			}
		}
	}()
	return nil
}

// publishPipelineError memberi tahu petugas bahwa jalur audio berhenti.
// Supervisor tidak perlu menerimanya: yang harus bertindak adalah petugas.
func (h *Handler) publishPipelineError(sessionID string, err error) {
	h.hub.PublishTo(sessionID, "officer", map[string]any{
		"type":    "session_error",
		"message": "Jalur audio terputus: " + err.Error(),
	})
}

// nudgeLoop mengingatkan butir yang masih pending secara berkala,
// bukan setiap ucapan, supaya tidak berisik di telinga petugas.
func (h *Handler) nudgeLoop(ctx context.Context, sessionID string) {
	t := time.NewTicker(h.nudgeEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if calibrator, ok := h.stt.(usecase.SpeakerRoleCalibrator); ok &&
				calibrator.SpeakerCalibrationActive(sessionID) {
				continue
			}
			if err := h.compliance.RemindPending(ctx, sessionID); err != nil {
				h.log.Warn("nudge gagal", "session", sessionID, "err", err)
			}
		}
	}
}

func (h *Handler) readPump(ctx context.Context, sessionID string, c *Client) {
	defer c.conn.Close()
	c.conn.SetReadLimit(1 << 20)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		// Frame biner = PCM16 dari AudioWorklet / recorder mobile.
		if msgType == websocket.BinaryMessage && c.role == "officer" {
			if err := h.stt.PushAudio(sessionID, data); err != nil {
				h.log.Warn("push audio gagal", "session", sessionID, "err", err)
			}
			continue
		}
		if msgType == websocket.TextMessage && c.role == "officer" {
			h.handleOfficerCommand(sessionID, data)
		}
	}
}

type officerCommand struct {
	Type          string `json:"type"`
	OfficerLabel  string `json:"officer_label"`
	CustomerLabel string `json:"customer_label"`
	// Reason diisi klien saat melaporkan audio tidak layak; kosong = sehat.
	Reason string `json:"reason"`
}

func (h *Handler) handleOfficerCommand(sessionID string, data []byte) {
	var command officerCommand
	if err := json.Unmarshal(data, &command); err != nil {
		return
	}

	// Laporan kualitas audio tidak bergantung pada dukungan kalibrasi STT.
	if command.Type == "audio_quality" {
		h.compliance.SetAudioQuality(sessionID, command.Reason)
		return
	}

	calibrator, ok := h.stt.(usecase.SpeakerRoleCalibrator)
	if !ok {
		h.hub.PublishTo(sessionID, "officer", map[string]any{
			"type": "speaker_calibration_error", "message": "STT tidak mendukung kalibrasi",
		})
		return
	}

	switch command.Type {
	case "begin_speaker_calibration":
		if err := calibrator.BeginSpeakerCalibration(sessionID); err != nil {
			h.publishCalibrationError(sessionID, err)
			return
		}
		h.hub.PublishTo(sessionID, "officer", map[string]any{"type": "speaker_calibration_started"})
	case "confirm_speaker_roles":
		if err := calibrator.ConfirmSpeakerRoles(sessionID, command.OfficerLabel, command.CustomerLabel); err != nil {
			h.publishCalibrationError(sessionID, err)
			return
		}
		h.hub.Publish(sessionID, map[string]any{
			"type":          "speaker_roles_confirmed",
			"officer_label": command.OfficerLabel, "customer_label": command.CustomerLabel,
		})
	}
}

func (h *Handler) publishCalibrationError(sessionID string, err error) {
	h.hub.PublishTo(sessionID, "officer", map[string]any{
		"type": "speaker_calibration_error", "message": err.Error(),
	})
}

func (h *Handler) writePump(c *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
