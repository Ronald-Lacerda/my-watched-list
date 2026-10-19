//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
)

func TestBrowserPWAOfflinePrivacy(t *testing.T) {
	b := browserFixture(t, true)
	p := b.page
	loginBrowser(t, b, 0)
	records(t, p, 5)
	waitJS(t, p, `()=>!!navigator.serviceWorker.controller`)
	value, err := p.Evaluate(`async()=>{const manifest=await(await fetch('/manifest.webmanifest')).json();const icons=await Promise.all(manifest.icons.map(icon=>new Promise((resolve,reject)=>{const img=new Image();img.onload=()=>resolve([img.width,img.height]);img.onerror=reject;img.src=icon.src})));return JSON.stringify({manifest,icons})}`)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Manifest struct {
			Name, StartURL, Display string
			Icons                   []struct{ Sizes, Purpose string }
		}
		Icons [][]int
	}
	// O nome JSON start_url precisa de um contrato explícito.
	var document map[string]json.RawMessage
	if err = json.Unmarshal([]byte(value.(string)), &document); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(document["manifest"], &result.Manifest); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(document["icons"], &result.Icons); err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	json.Unmarshal(document["manifest"], &manifest)
	if result.Manifest.Name != "My Watched List" || manifest["start_url"] != "/" || result.Manifest.Display != "standalone" || len(result.Icons) != 2 || result.Icons[0][0] != 192 || result.Icons[0][1] != 192 || result.Icons[1][0] != 512 || result.Icons[1][1] != 512 {
		t.Fatal("manifest/ícones não instaláveis", value)
	}
	for _, icon := range result.Manifest.Icons {
		if !strings.Contains(icon.Purpose, "maskable") {
			t.Fatal("ícone sem área de instalação")
		}
	}
	value, err = p.Evaluate(`async()=>{await fetch('/api/history');await fetch('/api/auth/me');const keys=await caches.keys();const urls=[];for(const key of keys){for(const request of await(await caches.open(key)).keys())urls.push(new URL(request.url).pathname)}return JSON.stringify(urls)}`)
	if err != nil {
		t.Fatal(err)
	}
	var cached []string
	if err = json.Unmarshal([]byte(value.(string)), &cached); err != nil || len(cached) < 7 {
		t.Fatal("shell ausente", err, value)
	}
	for _, path := range cached {
		if path != "/index.html" && path != "/manifest.webmanifest" && path != "/favicon.svg" && path != "/icon-192.png" && path != "/icon-512.png" && path != "/fonts/dm-sans-latin.woff2" && path != "/fonts/barlow-condensed-latin.woff2" && !strings.HasPrefix(path, "/assets/") {
			t.Fatal("dado privado/arquivo inesperado no cache", path)
		}
	}
	openRating(t, p, b.ids[0])
	fill(t, p, "Sua nota", "6")
	if err = p.Locator("textarea").Fill("Não salvar offline"); err != nil {
		t.Fatal(err)
	}
	if err = b.context.SetOffline(true); err != nil {
		t.Fatal(err)
	}
	click(t, p, "Salvar nota")
	visible(t, p.GetByRole("dialog").GetByRole("alert"))
	if text, _ := p.GetByRole("dialog").GetByRole("alert").TextContent(); !strings.Contains(text, "Sem conexão") {
		t.Fatal("gravação offline sem orientação")
	}
	var count int
	if err = b.f.pool.QueryRow(context.Background(), `SELECT count(*) FROM ratings`).Scan(&count); err != nil || count != 0 {
		t.Fatal("nota gravada offline", count, err)
	}
	click(t, p, "Cancelar")
	if _, err = p.Reload(); err != nil {
		t.Fatal("shell não abriu offline", err)
	}
	visible(t, p.Locator(".offline-banner"))
	visible(t, p.GetByRole("button", pw.PageGetByRoleOptions{Name: "Tentar novamente"}))
	if n, _ := p.Locator(".watch-card").Count(); n != 0 {
		t.Fatal("histórico privado reapareceu offline")
	}
	if err = b.context.SetOffline(false); err != nil {
		t.Fatal(err)
	}
	click(t, p, "Tentar novamente")
	records(t, p, 5)
	click(t, p, "Sair da conta")
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Bem-vindo de volta."}))
}
