//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"mywatchedlist/internal/auth"
)

type browserApp struct {
	f       appFixture
	server  *httptest.Server
	context pw.BrowserContext
	page    pw.Page
	ids     []string
}

func browserFixture(t *testing.T, withHistory bool) browserApp {
	t.Helper()
	var b browserApp
	if withHistory {
		b.f, b.ids = historyFixture(t)
	} else {
		b.f = fixtureApp(t)
	}
	dist, err := filepath.Abs("../../../web/dist")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(dist, "index.html")); err != nil {
		t.Fatal("execute npm --prefix web run build antes dos testes de navegador")
	}
	b.server = httptest.NewUnstartedServer(nil)
	origin := "http://" + b.server.Listener.Addr().String()
	handler, err := NewApp(b.f.pool, Options{Origins: []string{origin}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", handler)
	mux.Handle("/", http.FileServer(http.Dir(dist)))
	b.server.Config.Handler = mux
	b.server.Start()
	t.Cleanup(b.server.Close)
	driver, err := pw.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { driver.Stop() })
	browser, err := driver.Chromium.Launch(pw.BrowserTypeLaunchOptions{Headless: pw.Bool(true)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { browser.Close() })
	b.context, err = browser.NewContext(pw.BrowserNewContextOptions{Locale: pw.String("pt-BR"), TimezoneId: pw.String("America/Sao_Paulo"), Viewport: &pw.Size{Width: 1440, Height: 960}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.context.Close() })
	b.context.SetDefaultTimeout(8000)
	b.page, err = b.context.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func visible(t *testing.T, l pw.Locator) {
	t.Helper()
	if err := l.WaitFor(); err != nil {
		t.Fatal(err)
	}
}
func screenshot(t *testing.T, p pw.Page, name string) {
	t.Helper()
	dir, err := filepath.Abs("../../../.local")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Screenshot(pw.PageScreenshotOptions{Path: pw.String(filepath.Join(dir, name)), FullPage: pw.Bool(true)}); err != nil {
		t.Fatal(err)
	}
}
func loginBrowser(t *testing.T, b browserApp, person int) {
	t.Helper()
	if _, err := b.page.Goto(b.server.URL); err != nil {
		t.Fatal(err)
	}
	visible(t, b.page.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Bem-vindo de volta."}))
	if err := b.page.GetByLabel("Usuário").Fill(b.f.users[person].Username); err != nil {
		t.Fatal(err)
	}
	if err := b.page.GetByLabel("Senha", pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).Fill(appTestPassword); err != nil {
		t.Fatal(err)
	}
	if err := b.page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Entrar", Exact: pw.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	visible(t, b.page.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Nosso histórico"}))
}

func TestBrowserAuthentication(t *testing.T) {
	b := browserFixture(t, false)
	p := b.page
	if _, err := p.Goto(b.server.URL); err != nil {
		t.Fatal(err)
	}
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Bem-vindo de volta."}))
	screenshot(t, p, "fase2-login.png")
	if err := p.GetByLabel("Usuário").Fill("ronald"); err != nil {
		t.Fatal(err)
	}
	if err := p.GetByLabel("Senha", pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).Fill("incorreta"); err != nil {
		t.Fatal(err)
	}
	if err := p.GetByRole("button", pw.PageGetByRoleOptions{Name: "Entrar", Exact: pw.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	visible(t, p.GetByRole("alert"))
	text, _ := p.GetByRole("alert").TextContent()
	if !strings.Contains(text, "Não foi possível entrar") {
		t.Fatal("falha de login não foi mostrada em português")
	}
	loginBrowser(t, b, 0)
	if _, err := p.Reload(); err != nil {
		t.Fatal(err)
	}
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Nosso histórico"}))
	if name, err := p.Locator(".account-name strong").TextContent(); err != nil || name != "R" {
		t.Fatal("identidade incorreta", name, err)
	}
	value, err := p.Evaluate(`()=>JSON.stringify({local:localStorage.length,session:sessionStorage.length,cookie:document.cookie})`)
	if err != nil {
		t.Fatal(err)
	}
	var storage struct {
		Local, Session int
		Cookie         string
	}
	if err = json.Unmarshal([]byte(value.(string)), &storage); err != nil || storage.Local != 0 || storage.Session != 0 || strings.Contains(storage.Cookie, SessionCookie) {
		t.Fatal("credenciais expostas no armazenamento web")
	}
	if err = p.GetByRole("button", pw.PageGetByRoleOptions{Name: "Sair da conta"}).Click(); err != nil {
		t.Fatal(err)
	}
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Bem-vindo de volta."}))
	token, err := (auth.Service{Pool: b.f.pool}).Invite(context.Background(), b.f.users[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Goto(b.server.URL + "/#ativar=" + token); err != nil {
		t.Fatal(err)
	}
	visible(t, p.GetByLabel("Crie uma senha"))
	if strings.Contains(p.URL(), token) {
		t.Fatal("convite ficou no endereço")
	}
	if err = p.GetByLabel("Crie uma senha").Fill("Uma nova senha sintética 123"); err != nil {
		t.Fatal(err)
	}
	if err = p.GetByLabel("Confirme a senha").Fill("Diferente da primeira 123"); err != nil {
		t.Fatal(err)
	}
	if err = p.GetByRole("button", pw.PageGetByRoleOptions{Name: "Criar minha conta"}).Click(); err != nil {
		t.Fatal(err)
	}
	visible(t, p.GetByRole("alert"))
	text, _ = p.GetByRole("alert").TextContent()
	if !strings.Contains(text, "precisam ser iguais") {
		t.Fatal("confirmação de senha ignorada")
	}
	if err = p.GetByLabel("Confirme a senha").Fill("Uma nova senha sintética 123"); err != nil {
		t.Fatal(err)
	}
	if err = p.GetByRole("button", pw.PageGetByRoleOptions{Name: "Criar minha conta"}).Click(); err != nil {
		t.Fatal(err)
	}
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Nosso histórico"}))
}
