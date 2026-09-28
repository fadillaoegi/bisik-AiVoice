package assemblyai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bisik/bisik_backend/internal/domain"
)

func TestLLMMatcherMemakaiAuthTanpaBearer(t *testing.T) {
	matcher := newLLMMatcher("test-key", "claude-sonnet-4-6", "https://gateway.test/chat/completions")
	matcher.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "test-key" {
			t.Errorf("Authorization = %q, mau test-key", got)
		}
		requestBody, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(requestBody), `"model":"claude-sonnet-4-6"`) {
			t.Errorf("request tidak memuat model yang diharapkan: %s", requestBody)
		}
		return jsonResponse(http.StatusOK, `{"choices":[{"message":{"content":"{\"matched\":true,\"confidence\":0.93}"}}]}`), nil
	})}

	matched, confidence, err := matcher.Match(context.Background(), "Bunganya dua persen.", domain.DefaultObligations()[1])
	if err != nil {
		t.Fatalf("Match(): %v", err)
	}
	if !matched || confidence != 0.93 {
		t.Fatalf("verdict = %v %.2f, mau true 0.93", matched, confidence)
	}
}

func TestLLMMatcherMengembalikanStatusUpstream(t *testing.T) {
	matcher := newLLMMatcher("test-key", "model-salah", "https://gateway.test/chat/completions")
	matcher.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, "model tidak tersedia"), nil
	})}
	_, _, err := matcher.Match(context.Background(), "teks", domain.DefaultObligations()[0])
	if err == nil {
		t.Fatal("mengharapkan error untuk respons non-2xx")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// Ping harus melaporkan kalimat penyebab dari metadata.errors, bukan
// "invalid request body" yang tidak menjelaskan apa pun.
func TestPingMelaporkanPenyebabDariGateway(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"metadata":{"errors":["Your account does not have access to this LLM Gateway model"]},"message":"invalid request body","code":400}`))
	}))
	defer srv.Close()

	err := newLLMMatcher("kunci", "claude-sonnet-4-6", srv.URL).Ping(context.Background())
	if err == nil || !strings.Contains(err.Error(), "does not have access") || !strings.Contains(err.Error(), "400") {
		t.Fatalf("Ping() = %v, mau menyebut HTTP 400 dan penyebabnya", err)
	}
}

func TestPingBerhasilSaatGatewayMenerima(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()
	if err := newLLMMatcher("kunci", "m", srv.URL).Ping(context.Background()); err != nil {
		t.Fatalf("Ping() = %v, mau nil", err)
	}
}
