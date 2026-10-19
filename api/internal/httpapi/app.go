package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"mywatchedlist/internal/auth"
)

const SessionCookie = "mwl_session"

type Options struct{ Origins []string }
type app struct {
	auth    auth.Service
	pool    *pgxpool.Pool
	origins map[string]bool
	secure  bool
}
type userKey struct{}

func NewApp(pool *pgxpool.Pool, options Options) (http.Handler, error) {
	origins := options.Origins
	if len(origins) == 0 || (len(origins) == 1 && strings.TrimSpace(origins[0]) == "") {
		origins = []string{"http://127.0.0.1:8080", "http://localhost:8080"}
	}
	a := &app{auth: auth.Service{Pool: pool}, pool: pool, origins: map[string]bool{}}
	var scheme string
	for _, origin := range origins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (scheme != "" && scheme != u.Scheme) {
			return nil, errors.New("APP_ORIGINS inválido; use origens HTTP ou HTTPS com o mesmo protocolo")
		}
		scheme = u.Scheme
		a.origins[origin] = true
	}
	a.secure = scheme == "https"
	r := baseRouter(pool.Ping)
	r.Post("/api/auth/login", a.login)
	r.Post("/api/auth/activate", a.activate)
	r.Post("/api/auth/logout", a.logout)
	r.With(a.requireUser).Get("/api/auth/me", func(w http.ResponseWriter, r *http.Request) { reply(w, 200, map[string]any{"user": currentUser(r)}) })
	r.With(a.requireUser).Get("/api/filters", a.filters)
	r.With(a.requireUser).Get("/api/history", a.history)
	r.With(a.requireUser).Get("/api/connections", a.connections)
	r.With(a.requireUser).Put("/api/events/{id}/rating", a.rating)
	r.With(a.requireUser).Delete("/api/events/{id}/rating", a.rating)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		apiError(w, 404, "not_found", "Não encontramos esta página.")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		apiError(w, 405, "method_not_allowed", "Método não permitido.")
	})
	return middleware.Timeout(10 * time.Second)(privateHeaders(r)), nil
}

func privateHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}
func apiError(w http.ResponseWriter, status int, code, message string) {
	reply(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func (a *app) origin(w http.ResponseWriter, r *http.Request) bool {
	if !a.origins[r.Header.Get("Origin")] {
		apiError(w, 403, "origin_not_allowed", "A origem desta solicitação não é permitida.")
		return false
	}
	return true
}

func readJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || kind != "application/json" {
		apiError(w, 415, "json_required", "Envie os dados em JSON.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	var document json.RawMessage
	if err = decoder.Decode(&document); err == nil {
		if len(document) == 0 || document[0] != '{' {
			err = errors.New("objeto JSON obrigatório")
		} else {
			object := json.NewDecoder(bytes.NewReader(document))
			object.DisallowUnknownFields()
			err = object.Decode(out)
		}
	}
	if err == nil {
		var extra any
		if e := decoder.Decode(&extra); e != io.EOF {
			if e == nil {
				err = errors.New("mais de um objeto")
			} else {
				err = e
			}
		}
	}
	if err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			apiError(w, 413, "body_too_large", "Os dados enviados ultrapassam o limite permitido.")
		} else {
			apiError(w, 400, "invalid_json", "Confira os dados enviados e tente novamente.")
		}
		return false
	}
	return true
}
func cookieToken(r *http.Request) string {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}
func currentUser(r *http.Request) auth.User {
	user, _ := r.Context().Value(userKey{}).(auth.User)
	return user
}
func (a *app) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.auth.Me(r.Context(), cookieToken(r))
		if err != nil {
			a.authError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}
func (a *app) authError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrUnauthorized):
		apiError(w, 401, "unauthorized", "Não foi possível entrar. Verifique seus dados ou o link de acesso.")
	case errors.Is(err, auth.ErrThrottled):
		w.Header().Set("Retry-After", "900")
		apiError(w, 429, "too_many_attempts", auth.ErrThrottled.Error())
	case errors.Is(err, auth.ErrPassword):
		apiError(w, 400, "invalid_password", auth.ErrPassword.Error())
	default:
		apiError(w, 500, "unavailable", "Não foi possível concluir agora. Tente novamente.")
	}
}
func (a *app) sessionCookie(w http.ResponseWriter, s auth.Session) {
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: s.Token, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode, MaxAge: 14 * 24 * 60 * 60, Expires: s.ExpiresAt})
}
func (a *app) login(w http.ResponseWriter, r *http.Request) {
	if !a.origin(w, r) {
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s, err := a.auth.Login(r.Context(), body.Username, body.Password)
	if err != nil {
		a.authError(w, err)
		return
	}
	a.sessionCookie(w, s)
	reply(w, 200, map[string]any{"user": s.User})
}
func (a *app) activate(w http.ResponseWriter, r *http.Request) {
	if !a.origin(w, r) {
		return
	}
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s, err := a.auth.Activate(r.Context(), body.Token, body.Password)
	if err != nil {
		a.authError(w, err)
		return
	}
	a.sessionCookie(w, s)
	reply(w, 200, map[string]any{"user": s.User})
}
func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	if !a.origin(w, r) {
		return
	}
	if r.ContentLength != 0 {
		var empty struct{}
		if !readJSON(w, r, &empty) {
			return
		}
	}
	if err := a.auth.Logout(r.Context(), cookieToken(r)); err != nil {
		a.authError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	reply(w, 204, nil)
}
