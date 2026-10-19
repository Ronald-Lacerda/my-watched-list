//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"mywatchedlist/internal/auth"
	"mywatchedlist/internal/testdb"
)

const appTestPassword = "Senha sintética para testes 123"

type appFixture struct {
	pool    *pgxpool.Pool
	handler http.Handler
	origin  string
	users   []auth.User
	cookies []*http.Cookie
}

func fixtureApp(t *testing.T) appFixture {
	t.Helper()
	pool, _ := testdb.New(t)
	f := appFixture{pool: pool, origin: "http://app.test"}
	for _, name := range []string{"R", "Gabriela"} {
		var u auth.User
		u.Name = name
		u.Username = "ronald"
		if name == "Gabriela" {
			u.Username = "gabriela"
		}
		if err := pool.QueryRow(context.Background(), `INSERT INTO users(name,email,username,password_hash) VALUES($1,$2,$3,'!disabled') RETURNING id::text`, u.Name, strings.ToLower(name)+"@example.invalid", u.Username).Scan(&u.ID); err != nil {
			t.Fatal(err)
		}
		s := auth.Service{Pool: pool}
		token, err := s.Invite(context.Background(), u.ID)
		if err != nil {
			t.Fatal(err)
		}
		session, err := s.Activate(context.Background(), token, appTestPassword)
		if err != nil {
			t.Fatal(err)
		}
		f.users = append(f.users, u)
		f.cookies = append(f.cookies, &http.Cookie{Name: SessionCookie, Value: session.Token})
	}
	var err error
	f.handler, err = NewApp(pool, Options{Origins: []string{f.origin}})
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f appFixture) call(method, path, body string, person int, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if person >= 0 {
		r.AddCookie(f.cookies[person])
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}

func TestAuthHTTP(t *testing.T) {
	f := fixtureApp(t)
	ctx := context.Background()
	body := fmt.Sprintf(`{"username":"ronald","password":%q}`, appTestPassword)
	w := f.call("POST", "/api/auth/login", body, -1, f.origin)
	if w.Code != 200 || strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), appTestPassword) {
		t.Fatal("login HTTP incorreto", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("cookie ausente")
	}
	c := cookies[0]
	if c.Name != SessionCookie || !c.HttpOnly || c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 1209600 || time.Until(c.Expires) < 14*24*time.Hour-time.Minute {
		t.Fatal("atributos de cookie incorretos", c)
	}
	f.cookies[0] = c
	w = f.call("GET", "/api/auth/me", "", 0, "")
	var result struct{ User auth.User }
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || result.User.ID != f.users[0].ID || result.User.Name != "R" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("me incorreto", w.Code, err)
	}
	if w = f.call("POST", "/api/auth/logout", "{}", 0, "http://evil.test"); w.Code != 403 {
		t.Fatal("logout cross-origin aceito")
	}
	if w = f.call("GET", "/api/auth/me", "", 0, ""); w.Code != 200 {
		t.Fatal("origem inválida revogou sessão")
	}
	w = f.call("POST", "/api/auth/logout", "{}", 0, f.origin)
	expired := w.Result().Cookies()
	if w.Code != 204 || len(expired) != 1 || expired[0].MaxAge >= 0 || expired[0].Value != "" {
		t.Fatal("logout não limpou cookie", w.Code)
	}
	if w = f.call("GET", "/api/auth/me", "", 0, ""); w.Code != 401 {
		t.Fatal("sessão revogada aceita")
	}
	for _, person := range []int{-1, 0} {
		w = f.call("GET", "/api/auth/me", "", person, "")
		if w.Code != 401 || strings.Contains(w.Body.String(), "Gabriela") {
			t.Fatal("dados privados sem sessão")
		}
	}
	s := auth.Service{Pool: f.pool}
	token, err := s.Invite(ctx, f.users[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	activation := fmt.Sprintf(`{"token":%q,"password":%q}`, token, appTestPassword)
	if w = f.call("POST", "/api/auth/activate", activation, -1, f.origin); w.Code != 200 {
		t.Fatal("ativação HTTP falhou", w.Code)
	}
	if w = f.call("POST", "/api/auth/activate", activation, -1, f.origin); w.Code != 401 {
		t.Fatal("convite repetido aceito")
	}
	if _, err = f.pool.Exec(ctx, `UPDATE auth_sessions SET created_at=now()-interval '15 days',expires_at=now()-interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	if w = f.call("GET", "/api/auth/me", "", 1, ""); w.Code != 401 {
		t.Fatal("sessão expirada HTTP aceita")
	}
}

func TestHTTPOriginAndJSON(t *testing.T) {
	f := fixtureApp(t)
	for _, path := range []string{"/api/auth/login", "/api/auth/activate", "/api/auth/logout"} {
		for _, origin := range []string{"", "http://evil.test"} {
			if w := f.call("POST", path, "{}", -1, origin); w.Code != 403 {
				t.Fatalf("origem aceita em %s: %d", path, w.Code)
			}
		}
	}
	for _, test := range []struct {
		body, kind string
		status     int
	}{{"{}", "text/plain", 415}, {"{", "application/json", 400}, {`{"user_id":"outro"}`, "application/json", 400}, {"{}{}", "application/json", 400}, {`{"password":"` + strings.Repeat("x", 65536) + `"}`, "application/json", 413}} {
		r := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(test.body))
		r.Header.Set("Origin", f.origin)
		r.Header.Set("Content-Type", test.kind)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != test.status || strings.Contains(w.Body.String(), "password_hash") {
			t.Fatal("validação HTTP incorreta", test.status, w.Code)
		}
	}
	for i := 0; i < 10; i++ {
		if w := f.call("POST", "/api/auth/login", `{"username":"ronald","password":"incorreta"}`, -1, f.origin); w.Code != 401 {
			t.Fatal("credenciais inválidas sem 401")
		}
	}
	if w := f.call("POST", "/api/auth/login", `{"username":"ronald","password":"incorreta"}`, -1, f.origin); w.Code != 429 || w.Header().Get("Retry-After") != "900" {
		t.Fatal("limite HTTP não aplicado")
	}
}

func TestHTTPSCookieAndOriginConfiguration(t *testing.T) {
	f := fixtureApp(t)
	handler, err := NewApp(f.pool, Options{Origins: []string{"https://app.test"}})
	if err != nil {
		t.Fatal(err)
	}
	f.handler = handler
	w := f.call("POST", "/api/auth/login", fmt.Sprintf(`{"username":"ronald","password":%q}`, appTestPassword), -1, "https://app.test")
	cookies := w.Result().Cookies()
	if w.Code != 200 || len(cookies) != 1 || !cookies[0].Secure {
		t.Fatal("HTTPS sem Secure")
	}
	for _, origin := range []string{"https://app.test/path", "https://user:secret@app.test", "javascript:alert(1)"} {
		if _, err = NewApp(f.pool, Options{Origins: []string{origin}}); err == nil {
			t.Fatal("configuração de origem inválida aceita")
		}
	}
}
