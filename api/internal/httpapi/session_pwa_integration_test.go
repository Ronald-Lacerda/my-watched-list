//go:build integration

package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
)

// AC8: a identidade visual continua disponível no shell, sem CDN de fontes.
func TestBrowserSessionLocalFonts(t *testing.T) {
	b := browserFixture(t, true)
	p := b.page
	loginBrowser(t, b, 0)
	records(t, p, 5)
	waitJS(t, p, `()=>!!navigator.serviceWorker.controller`)
	value, err := p.Evaluate(`async()=>{
		await document.fonts.load('400 16px "DM Sans"');
		await document.fonts.load('500 24px "Barlow Condensed"');
		const manifest=await(await fetch('/manifest.webmanifest')).json();
		const paths=[];
		for(const key of await caches.keys())for(const req of await(await caches.open(key)).keys())paths.push(new URL(req.url).pathname);
		const fontSources=[...document.styleSheets].flatMap(sheet=>[...sheet.cssRules]).filter(rule=>rule.type===CSSRule.FONT_FACE_RULE).map(rule=>rule.style.getPropertyValue('src'));
		return JSON.stringify({paths,fontSources,theme:manifest.theme_color,background:manifest.background_color,meta:document.querySelector('meta[name="theme-color"]').content,scheme:getComputedStyle(document.documentElement).colorScheme});
	}`)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Paths, FontSources              []string
		Theme, Background, Meta, Scheme string
	}
	if err = json.Unmarshal([]byte(value.(string)), &result); err != nil {
		t.Fatal(err)
	}
	if result.Theme != result.Meta || result.Background != result.Theme || result.Scheme != "dark" {
		t.Fatal("shell não mantém o mesmo tema escuro", value)
	}
	for _, expected := range []string{"/fonts/dm-sans-latin.woff2", "/fonts/barlow-condensed-latin.woff2"} {
		found := false
		for _, path := range result.Paths {
			if path == expected {
				found = true
			}
		}
		if !found {
			t.Fatal("fonte pública fora do shell offline", expected, result.Paths)
		}
	}
	if len(result.FontSources) < 2 {
		t.Fatal("tipografia local não declarada", value)
	}
	for _, source := range result.FontSources {
		if source == "" || containsRemoteFont(source) {
			t.Fatal("fonte depende de rede externa", source)
		}
	}
	if err = b.context.SetOffline(true); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Reload(); err != nil {
		t.Fatal(err)
	}
	visible(t, p.Locator(".offline-banner"))
	value, err = p.Evaluate(`async()=>{
		const a=await document.fonts.load('400 16px "DM Sans"');
		const b=await document.fonts.load('500 24px "Barlow Condensed"');
		return a.length>0&&b.length>0&&a.every(f=>f.status==='loaded')&&b.every(f=>f.status==='loaded');
	}`)
	if err != nil || value != true {
		t.Fatal("fontes não disponíveis após reabrir offline", value, err)
	}
}

func containsRemoteFont(source string) bool {
	// CSSOM normaliza aspas; analisar a origem evita aceitar um host externo.
	return strings.Contains(source, "http:") || strings.Contains(source, "https:") || strings.Contains(source, "url(\"//") || strings.Contains(source, "url('//")
}
