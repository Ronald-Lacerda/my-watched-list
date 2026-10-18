package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(ping func(context.Context) error) http.Handler {
	return baseRouter(ping)
}

func baseRouter(ping func(context.Context) error) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Get("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		respond(w, http.StatusOK, "ok")
	})
	r.Get("/api/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := ping(ctx); err != nil {
			respond(w, http.StatusServiceUnavailable, "unavailable")
			return
		}
		respond(w, http.StatusOK, "ready")
	})
	return r
}

func respond(w http.ResponseWriter, code int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
}
