package ws

import (
	"net/http/httptest"
	"testing"
)

func TestWebSocketMenerimaSameOriginDeployment(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, "http://localhost:5173", 0, nil)
	req := httptest.NewRequest("GET", "https://bisik.onrender.com/ws", nil)
	req.Host = "bisik.onrender.com"
	req.Header.Set("Origin", "https://bisik.onrender.com")
	if !h.upgrader.CheckOrigin(req) {
		t.Fatal("same-origin production harus diterima")
	}
}

func TestWebSocketMenolakOriginAsing(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, "http://localhost:5173", 0, nil)
	req := httptest.NewRequest("GET", "https://bisik.onrender.com/ws", nil)
	req.Host = "bisik.onrender.com"
	req.Header.Set("Origin", "https://evil.example")
	if h.upgrader.CheckOrigin(req) {
		t.Fatal("origin asing harus ditolak")
	}
}
