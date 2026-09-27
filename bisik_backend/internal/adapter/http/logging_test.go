package http

import (
	"bufio"
	"bytes"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// hijackableRecorder meniru ResponseWriter server sungguhan, yang selalu
// mendukung Hijack. httptest.ResponseRecorder sendiri tidak mendukungnya.
type hijackableRecorder struct {
	*httptest.ResponseRecorder
	hijacked bool
}

func (h *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, nil
}

// Upgrade WebSocket menuntut ResponseWriter mengimplementasikan
// http.Hijacker. Pembungkus logging yang menelannya akan mematikan seluruh
// jalur realtime, dan gejalanya cuma koneksi gagal tanpa pesan jelas.
func TestLoggingMeneruskanHijackUntukWebSocket(t *testing.T) {
	var reached bool
	handler := withLogging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("ResponseWriter kehilangan http.Hijacker — WebSocket akan mati")
		}
		if _, _, err := hijacker.Hijack(); err != nil {
			t.Fatalf("Hijack(): %v", err)
		}
		reached = true
	}), slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)))

	recorder := &hijackableRecorder{ResponseRecorder: httptest.NewRecorder()}
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ws", nil))

	if !reached || !recorder.hijacked {
		t.Fatal("handler tidak sempat melakukan hijack")
	}
}

func TestLoggingMencatatStatusDanLevel(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := withLogging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}), log)
	handler.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/api/sessions", nil))

	out := buf.String()
	if !strings.Contains(out, "status=401") {
		t.Fatalf("status tidak tercatat:\n%s", out)
	}
	// 4xx berarti klien yang salah, bukan server — jangan dilaporkan ERROR.
	if !strings.Contains(out, "level=WARN") {
		t.Fatalf("4xx harus dicatat sebagai WARN:\n%s", out)
	}
}

func TestLoggingMelewatiHealthCheckYangBerhasil(t *testing.T) {
	var buf bytes.Buffer
	handler := withLogging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), slog.New(slog.NewTextHandler(&buf, nil)))

	handler.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/health", nil))

	if buf.Len() != 0 {
		t.Fatalf("health check yang sehat tidak perlu dicatat:\n%s", buf.String())
	}
}
