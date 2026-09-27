package http

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// withLogging mencatat setiap request beserta status jawabannya.
//
// Tanpa ini, kegagalan seperti 401 atau 404 hanya terlihat di panel Network
// browser — dan kalau klien menelan errornya, tidak terlihat sama sekali.
// Level dibedakan supaya masalah nyata tidak tenggelam: 4xx adalah peringatan
// (klien salah), 5xx adalah error (kita yang salah).
func withLogging(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		// Health check dipanggil terus-menerus oleh Render; mencatat yang
		// berhasil hanya menenggelamkan log yang berguna.
		if r.URL.Path == "/health" && recorder.status < http.StatusBadRequest {
			return
		}

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"ms", time.Since(started).Milliseconds(),
		}
		switch {
		case recorder.status >= http.StatusInternalServerError:
			log.Error("request gagal", attrs...)
		case recorder.status >= http.StatusBadRequest:
			log.Warn("request ditolak", attrs...)
		default:
			log.Info("request", attrs...)
		}
	})
}

// statusRecorder mengingat status yang ditulis handler.
//
// Hijack WAJIB diteruskan: upgrade WebSocket menuntut ResponseWriter
// mengimplementasikan http.Hijacker, dan pembungkus yang menelannya akan
// mematikan seluruh jalur realtime — gejalanya koneksi gagal tanpa pesan
// yang jelas.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
