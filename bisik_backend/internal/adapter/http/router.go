package http

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bisik/bisik_backend/internal/domain"
)

func NewRouter(
	h *Handler, wsHandler http.Handler, verifier TokenVerifier,
	allowedOrigins, staticDir string, log *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	// Siapa pun boleh: health check dan pintu masuk login.
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /api/auth/login", h.Login)

	// Sudah login, peran apa pun.
	loggedIn := requireAuth(verifier)
	mux.HandleFunc("GET /api/auth/me", loggedIn(h.Me))
	mux.HandleFunc("GET /api/obligations", loggedIn(h.Obligations))
	// Otorisasi per sesi dilakukan di dalam handler: supervisor boleh membaca
	// sesi mana pun, petugas hanya miliknya sendiri.
	mux.HandleFunc("GET /api/sessions/{id}/report", loggedIn(h.Report))

	// Khusus petugas: hanya petugas yang menjalankan dan mengakhiri sesi.
	officerOnly := requireAuth(verifier, domain.RoleOfficer)
	mux.HandleFunc("POST /api/sessions", officerOnly(h.StartSession))
	mux.HandleFunc("POST /api/sessions/{id}/end", officerOnly(h.EndSession))

	// Khusus supervisor: daftar sesi untuk dipantau.
	supervisorOnly := requireAuth(verifier, domain.RoleSupervisor)
	mux.HandleFunc("GET /api/sessions", supervisorOnly(h.ListSessions))

	// WebSocket memverifikasi tokennya sendiri: browser tidak bisa
	// menyetel header Authorization pada koneksi WebSocket.
	mux.Handle("GET /ws", wsHandler)
	if staticDir != "" {
		mux.Handle("/", spa(staticDir))
	}

	return withLogging(cors(mux, allowedOrigins), log)
}

// spa menyajikan hasil build React dari container yang sama. Rute frontend
// seperti /officer harus fallback ke index.html, sedangkan API yang tidak
// dikenal tetap mengembalikan 404 dan tidak menyamar sebagai HTML.
func spa(staticDir string) http.Handler {
	files := http.FileServer(http.Dir(staticDir))
	index := filepath.Join(staticDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/ws" {
			http.NotFound(w, r)
			return
		}

		rel := strings.TrimPrefix(filepath.Clean(r.URL.Path), string(filepath.Separator))
		candidate := filepath.Join(staticDir, rel)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}

func cors(next http.Handler, allowedOrigins string) http.Handler {
	origins := strings.Split(allowedOrigins, ",")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, o := range origins {
			if strings.TrimSpace(o) == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
