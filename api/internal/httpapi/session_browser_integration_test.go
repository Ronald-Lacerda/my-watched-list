//go:build integration

package httpapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
)

func highlightedEvent(t *testing.T, p pw.Page, expected string) pw.Locator {
	t.Helper()
	featured := p.Locator(".featured-watch")
	visible(t, featured)
	if n, err := featured.Count(); err != nil || n != 1 {
		t.Fatal("o destaque deve representar um único evento", n, err)
	}
	if id, err := featured.GetAttribute("data-event-id"); err != nil || id != expected {
		t.Fatal("destaque não corresponde à seleção atual", id, expected, err)
	}
	return featured
}

// AC2/3/5/6: destaque real, sem reunir perfis, reage a filtros e à nota zero.
func TestBrowserSessionHighlightAndFilters(t *testing.T) {
	b := browserFixture(t, true)
	p := b.page
	first := b.ids[0]
	person := b.f.users[0].Name
	if b.ids[1] > first {
		first = b.ids[1]
		person = b.f.users[1].Name
	}
	if w := b.f.call("PUT", "/api/events/"+first+"/rating", `{"score":10,"comment":"Opinião da Gabriela"}`, 1, b.f.origin); w.Code != 200 {
		t.Fatal("preparação da nota individual", w.Code, w.Body.String())
	}
	loginBrowser(t, b, 0)
	records(t, p, 6)
	featured := highlightedEvent(t, p, first)
	if text, err := featured.Locator(".couple-score span").TextContent(); err != nil || text != "Média · 1/2" {
		t.Fatal("contador não corresponde à única avaliação disponível", text, err)
	}
	for _, expected := range []string{"Filme A", person, "Netflix", "18 de outubro de 2026"} {
		if text, _ := featured.TextContent(); !strings.Contains(text, expected) {
			t.Fatal("destaque perdeu informação do evento", expected, text)
		}
	}
	for _, id := range b.ids {
		if count, err := eventCard(p, id).Count(); err != nil || count != 1 {
			t.Fatal("evento duplicado ou reunido a outro perfil", id, count, err)
		}
	}
	openRating(t, p, first)
	saveRating(t, p, "0", "Nota zero no destaque")
	waitJS(t, p, `()=>document.querySelector('.featured-watch .couple-score strong')?.textContent==='5'`)
	for _, id := range b.ids[:2] {
		card := eventCard(p, id)
		if text, err := card.Locator(".couple-score strong").TextContent(); err != nil || text != "5" {
			t.Fatal("média da obra não atualizada em todos os eventos", id, text, err)
		}
		if text, err := card.Locator(".couple-score span").TextContent(); err != nil || text != "Média · 2/2" {
			t.Fatal("contador não corresponde às duas avaliações da obra", id, text, err)
		}
		rows, err := card.Locator(".individual-scores span").AllTextContents()
		if err != nil || len(rows) != 2 {
			t.Fatal("notas individuais ausentes", id, rows, err)
		}
		individual := map[string]bool{}
		for _, row := range rows {
			individual[row] = true
		}
		if !individual[b.f.users[0].Name+"0"] || !individual[b.f.users[1].Name+"10"] {
			t.Fatal("nota zero ou nota da parceira alterada", id, rows)
		}
		if text, err := card.GetByRole("button").TextContent(); err != nil || text != "Sua nota: 0" {
			t.Fatal("ação não preservou a nota zero", id, text, err)
		}
	}
	choose(t, p, "Streaming", "prime_video")
	records(t, p, 1)
	highlightedEvent(t, p, b.ids[4])
	choose(t, p, "Pessoa", b.f.users[1].ID)
	records(t, p, 0)
	if n, _ := p.Locator(".featured-watch").Count(); n != 0 {
		t.Fatal("destaque de seleção anterior permaneceu na seleção vazia")
	}
	click(t, p, "Limpar filtros")
	records(t, p, 6)
	choose(t, p, "Pessoa", b.f.users[1].ID)
	choose(t, p, "Streaming", "netflix")
	fill(t, p, "Data inicial", "2026-10-15")
	fill(t, p, "Data final", "2026-10-17")
	records(t, p, 2)
	highlightedEvent(t, p, b.ids[3])
	if !strings.Contains(p.URL(), "person="+b.f.users[1].ID) || !strings.Contains(p.URL(), "service=netflix") || !strings.Contains(p.URL(), "from=2026-10-15") || !strings.Contains(p.URL(), "to=2026-10-17") {
		t.Fatal("redesign perdeu os filtros no endereço", p.URL())
	}
	if _, err := p.Reload(); err != nil {
		t.Fatal(err)
	}
	records(t, p, 2)
	highlightedEvent(t, p, b.ids[3])
	screenshot(t, p, "sessao-filtros.png")
}

// AC3: acrescentar uma página não promove outro evento nem repete o destaque.
func TestBrowserSessionHighlightPagination(t *testing.T) {
	b := browserFixture(t, true)
	p := b.page
	_, err := b.f.pool.Exec(context.Background(), `INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title_id,source_work_id,source_title,match_title,media_type_hint,watched_on) SELECT user_id,streaming_account_id,'sessao-'||n,(500+n)::text,(500+n)::text,'Registro '||n,'Registro '||n,'movie','2026-10-18'::date FROM watch_events CROSS JOIN generate_series(1,55) n WHERE id=$1`, b.ids[0])
	if err != nil {
		t.Fatal(err)
	}
	var first string
	if err = b.f.pool.QueryRow(context.Background(), `SELECT id::text FROM watch_events ORDER BY watched_on DESC,id DESC LIMIT 1`).Scan(&first); err != nil {
		t.Fatal(err)
	}
	loginBrowser(t, b, 0)
	records(t, p, 50)
	highlightedEvent(t, p, first)
	click(t, p, "Carregar mais")
	records(t, p, 61)
	highlightedEvent(t, p, first)
	if value, err := p.Evaluate(`()=>new Set([...document.querySelectorAll('.watch-card')].map(e=>e.dataset.eventId)).size`); err != nil || value != 61 {
		t.Fatal("destaque/paginação duplicou ou perdeu eventos", value, err)
	}
}

// AC4/7: fallback de arte é parte do produto, inclusive para o título longo.
func TestBrowserSessionPosterFallbackAndKeyboard(t *testing.T) {
	b := browserFixture(t, true)
	p := b.page
	longTitle := "Uma história extraordinariamente longa para continuar lembrando depois dos créditos"
	var titleID string
	if err := b.f.pool.QueryRow(context.Background(), `INSERT INTO titles(tmdb_id,media_type,title,release_year,poster_path,genres) VALUES(999991,'movie',$1,2021,'/synthetic-failure.jpg',ARRAY['Drama']) RETURNING id::text`, longTitle).Scan(&titleID); err != nil {
		t.Fatal(err)
	}
	if _, err := b.f.pool.Exec(context.Background(), `UPDATE watch_events SET title_id=$1 WHERE id=$2 OR id=$3`, titleID, b.ids[0], b.ids[1]); err != nil {
		t.Fatal(err)
	}
	var validTitleID string
	if err := b.f.pool.QueryRow(context.Background(), `INSERT INTO titles(tmdb_id,media_type,title,release_year,poster_path) VALUES(999992,'movie','Filme B com pôster',2020,'/synthetic-success.jpg') RETURNING id::text`).Scan(&validTitleID); err != nil {
		t.Fatal(err)
	}
	if _, err := b.f.pool.Exec(context.Background(), `UPDATE watch_events SET title_id=$1 WHERE id=$2`, validTitleID, b.ids[4]); err != nil {
		t.Fatal(err)
	}
	pixel, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j5ZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Route("https://image.tmdb.org/**", func(route pw.Route) {
		if strings.HasSuffix(route.Request().URL(), "/synthetic-success.jpg") {
			route.Fulfill(pw.RouteFulfillOptions{Status: pw.Int(200), ContentType: pw.String("image/png"), Body: pixel})
		} else {
			route.Abort()
		}
	}); err != nil {
		t.Fatal(err)
	}
	loginBrowser(t, b, 0)
	records(t, p, 6)
	first := b.ids[0]
	if b.ids[1] > first {
		first = b.ids[1]
	}
	featured := highlightedEvent(t, p, first)
	visible(t, featured.GetByRole("heading", pw.LocatorGetByRoleOptions{Name: longTitle, Exact: pw.Bool(true)}))
	waitJS(t, p, `()=>!document.querySelector('.featured-watch .poster img')&&!!document.querySelector('.featured-watch .poster-letter')`)
	for _, width := range []int{1280, 390, 320} {
		if err := p.SetViewportSize(width, 900); err != nil {
			t.Fatal(err)
		}
		waitJS(t, p, `()=>document.documentElement.scrollWidth<=window.innerWidth`)
		button := featured.GetByRole("button")
		value, err := button.Evaluate(`e=>{const r=e.getBoundingClientRect();return r.width>=44&&r.height>=44}`, nil)
		if err != nil || value != true {
			t.Fatal("alvo de avaliação pequeno", width, value, err)
		}
		openRating(t, p, first)
		waitJS(t, p, `()=>document.querySelector('dialog').scrollWidth<=window.innerWidth`)
		if err = p.GetByLabel("Sua nota", pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).Press("Escape"); err != nil {
			t.Fatal(err)
		}
		waitJS(t, p, `()=>!document.querySelector('dialog')`)
		value, err = button.Evaluate(`e=>e===document.activeElement&&getComputedStyle(e).outlineStyle!=='none'`, nil)
		if err != nil || value != true {
			t.Fatal("Escape não devolveu foco visível", width, value, err)
		}
		if _, err := p.Evaluate(`()=>window.scrollTo(0,0)`); err != nil {
			t.Fatal(err)
		}
		screenshot(t, p, fmt.Sprintf("sessao-sem-poster-%d.png", width))
	}
	choose(t, p, "Streaming", "prime_video")
	records(t, p, 1)
	featured = highlightedEvent(t, p, b.ids[4])
	visible(t, featured.GetByRole("heading", pw.LocatorGetByRoleOptions{Name: "Filme B com pôster", Exact: pw.Bool(true)}))
	waitJS(t, p, `()=>document.querySelector('.featured-watch .poster img')?.naturalWidth===1`)
	value, err := featured.Locator(".poster img").Evaluate(`e=>e.hasAttribute('width')&&e.hasAttribute('height')&&e.loading==='eager'&&e.fetchPriority==='high'`, nil)
	if err != nil || value != true {
		t.Fatal("pôster válido não reservou espaço/prioridade no destaque", value, err)
	}
}
