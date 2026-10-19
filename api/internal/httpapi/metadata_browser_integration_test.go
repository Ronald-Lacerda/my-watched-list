//go:build integration

package httpapi

import (
	pw "github.com/mxschmitt/playwright-go"
	"strings"
	"testing"
)

func TestMetadataBrowser(t *testing.T) {
	stub := &metadataStub{fail: true}
	b := browserFixtureOptions(t, true, Options{Metadata: stub, NetflixMetadata: stub})
	p := b.page
	// Imagens sintéticas; o teste não consulta TMDB/Netflix reais.
	if err := b.context.Route("https://image.tmdb.org/**", func(route pw.Route) {
		route.Fulfill(pw.RouteFulfillOptions{Status: pw.Int(200), ContentType: pw.String("image/svg+xml"), Body: "<svg xmlns='http://www.w3.org/2000/svg' width='80' height='120'><rect width='80' height='120' fill='#345'/></svg>"})
	}); err != nil {
		t.Fatal(err)
	}
	if err := b.context.Route("https://*.nflxso.net/**", func(route pw.Route) {
		route.Fulfill(pw.RouteFulfillOptions{Status: pw.Int(200), ContentType: pw.String("image/svg+xml"), Body: "<svg xmlns='http://www.w3.org/2000/svg' width='80' height='120'><rect width='80' height='120' fill='#654'/></svg>"})
	}); err != nil {
		t.Fatal(err)
	}
	for actor, score := range []string{"0", "10"} {
		if w := b.f.call("PUT", "/api/events/"+b.ids[2]+"/rating", `{"score":`+score+`,"comment":"Preservada"}`, actor, b.f.origin); w.Code != 200 {
			t.Fatal("fixture de notas", w.Code)
		}
	}
	loginBrowser(t, b, 0)
	card := eventCard(p, b.ids[2])
	visible(t, card)
	info := card.Locator(".metadata-button")
	if err := info.Click(); err != nil {
		t.Fatal(err)
	}
	dialog := p.GetByRole("dialog")
	visible(t, dialog)
	focused, err := dialog.GetByLabel("Título da obra").Evaluate(`e=>e===document.activeElement`, nil)
	if err != nil || focused != true {
		t.Fatal("foco não foi para busca", focused, err)
	}
	if err := dialog.GetByLabel("Título da obra").Fill("Minha busca"); err != nil {
		t.Fatal(err)
	}
	click(t, p, "Buscar no TMDB")
	visible(t, dialog.GetByRole("alert"))
	if value, err := dialog.GetByLabel("Título da obra").InputValue(); err != nil || value != "Minha busca" {
		t.Fatal("erro apagou consulta", value, err)
	}
	stub.fail = false
	click(t, p, "Buscar no TMDB")
	choose := dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Usar Série identificada, 2019", Exact: pw.Bool(true)})
	visible(t, choose)
	if value, err := card.Locator(".metadata-missing").TextContent(); err != nil || value != "Sinopse ainda não identificada." {
		t.Fatal("busca associou antes da escolha", value, err)
	}
	if err := choose.Click(); err != nil {
		t.Fatal(err)
	}
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Série identificada", Exact: pw.Bool(true)}))
	card = eventCard(p, b.ids[2])
	visible(t, card.Locator(".watch-overview"))
	if text, err := card.Locator(".watch-overview").TextContent(); err != nil || text != "Sinopse oficial" {
		t.Fatal("sinopse não está no card", text, err)
	}
	if text, err := card.Locator(".metadata-source").TextContent(); err != nil || text != "Sinopse do TMDB" {
		t.Fatal("fonte ausente", text, err)
	}
	visible(t, card.Locator(".poster img"))
	if src, err := card.Locator(".poster img").GetAttribute("src"); err != nil || !strings.HasSuffix(src, "/official.jpg") {
		t.Fatal("capa não atualizou", src, err)
	}
	if text, err := card.Locator(".rating-button").TextContent(); err != nil || text != "Sua nota: 0" {
		t.Fatal("correção apagou zero", text, err)
	}
	if err := p.SetViewportSize(390, 844); err != nil {
		t.Fatal(err)
	}
	if err := card.Locator(".metadata-button").Click(); err != nil {
		t.Fatal(err)
	}
	visible(t, dialog)
	overflow, err := p.Evaluate(`()=>document.documentElement.scrollWidth>innerWidth`)
	if err != nil || overflow != false {
		t.Fatal("overflow celular", overflow, err)
	}
	if err := p.Keyboard().Press("Escape"); err != nil {
		t.Fatal(err)
	}
	if err := dialog.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden}); err != nil {
		t.Fatal(err)
	}
	focused, err = card.Locator(".metadata-button").Evaluate(`e=>e===document.activeElement`, nil)
	if err != nil || focused != true {
		t.Fatal("Escape perdeu foco", focused, err)
	}
	if err := card.Locator(".metadata-button").Click(); err != nil {
		t.Fatal(err)
	}
	click(t, p, "Usar informações oficiais da Netflix")
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Série oficial Netflix", Exact: pw.Bool(true)}))
	card = eventCard(p, b.ids[2])
	visible(t, card.Locator(".watch-overview"))
	if text, err := card.Locator(".watch-overview").TextContent(); err != nil || text != "Sinopse da Netflix" {
		t.Fatal("fallback não apareceu", text, err)
	}
	if src, err := card.Locator(".poster img").GetAttribute("src"); err != nil || src != "https://occ-0.nflxso.net/official.jpg" {
		t.Fatal("imagem Netflix ausente", src, err)
	}
	openRating(t, p, b.ids[2])
	visible(t, dialog.GetByText("Sua nota vale para a série inteira.", pw.LocatorGetByTextOptions{Exact: pw.Bool(true)}))
	if value, err := p.GetByLabel("Sua nota", pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).InputValue(); err != nil || value != "0" {
		t.Fatal("formulário perdeu nota", value, err)
	}
	if text, err := dialog.Locator(".partner-score").TextContent(); err != nil || text != "10/10" {
		t.Fatal("nota parceira alterada", text, err)
	}
	click(t, p, "Cancelar")
	screenshot(t, p, "capas-sinopses-celular.png")
	cached, err := p.Evaluate(`async()=>{const urls=[];for(const name of await caches.keys())for(const r of await (await caches.open(name)).keys())urls.push(r.url);return urls.some(u=>u.includes('/api/')||u.includes('nflxso.net')||u.includes('image.tmdb.org'))}`)
	if err != nil || cached != false {
		t.Fatal("metadados privados persistidos no cache", cached, err)
	}
}
