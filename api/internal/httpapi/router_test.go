package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestHealthAndReadiness(t *testing.T) {
	for _, tt := range []struct {
		name, path, body string
		pingErr          error
		code             int
	}{
		{"processo vivo independe do banco", "/api/healthz", "{\"status\":\"ok\"}\n", errors.New("indisponível"), 200},
		{"banco disponível", "/api/readyz", "{\"status\":\"ready\"}\n", nil, 200},
		{"banco indisponível", "/api/readyz", "{\"status\":\"unavailable\"}\n", errors.New("senha-secreta"), 503},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := New(func(context.Context) error { return tt.pingErr })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
			if w.Code != tt.code || w.Body.String() != tt.body {
				t.Fatalf("resposta = %d %q; esperado %d %q", w.Code, w.Body.String(), tt.code, tt.body)
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q", got)
			}
		})
	}
}
