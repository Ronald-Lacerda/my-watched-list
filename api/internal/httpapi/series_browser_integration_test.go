//go:build integration

package httpapi

import (
	"strings"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
)

func TestBrowserGroupedSeriesAndRatings(t *testing.T) {
	b := browserFixture(t, true)
	p := b.page
	mutate(t, b.f, "PUT", b.ids[3], `{"score":10}`, 1)
	loginBrowser(t, b, 0)
	fill(t, p, "Data inicial", "2026-10-16")
	fill(t, p, "Data final", "2026-10-17")
	records(t, p, 1)
	card := highlightedEvent(t, p, b.ids[2])
	visible(t, card.GetByRole("heading", pw.LocatorGetByRoleOptions{Name: "Série A", Exact: pw.Bool(true)}))
	visible(t, card.GetByText("2 registros no período", pw.LocatorGetByTextOptions{Exact: pw.Bool(true)}))
	watchers, err := card.Locator(".watch-person").AllTextContents()
	if err != nil || len(watchers) != 2 {
		t.Fatal("participantes ausentes", watchers, err)
	}
	watcherNames := map[string]bool{}
	for _, watcher := range watchers {
		watcherNames[watcher] = true
	}
	for _, person := range b.f.users {
		if !watcherNames[person.Name[:1]+person.Name] {
			t.Fatal("participante incorreto", person.Name, watchers)
		}
	}
	if text, _ := card.TextContent(); strings.Contains(text, "episódio") || !strings.Contains(text, "Última vez em 17 de outubro de 2026") {
		t.Fatal("card deve representar série, sem episódio", text)
	}
	openRating(t, p, b.ids[2])
	visible(t, p.GetByText("Sua nota vale para a série inteira.", pw.PageGetByTextOptions{Exact: pw.Bool(true)}))
	saveRating(t, p, "0", "Nota da série")
	waitJS(t, p, `()=>document.querySelector('.featured-watch .couple-score strong')?.textContent==='5'`)
	if text, _ := card.Locator(".couple-score span").TextContent(); text != "Média · 2/2" {
		t.Fatal("contador da série não atualizado", text)
	}
	visible(t, card.GetByText("Sua nota: 0", pw.LocatorGetByTextOptions{Exact: pw.Bool(true)}))
	openRating(t, p, b.ids[2])
	saveRating(t, p, "8.5", "Nota editada da série")
	waitJS(t, p, `()=>document.querySelector('.featured-watch .couple-score strong')?.textContent==='9,25'`)
	openRating(t, p, b.ids[2])
	click(t, p, "Remover minha nota")
	waitJS(t, p, `()=>!document.querySelector('dialog')&&document.querySelector('.featured-watch .couple-score strong')?.textContent==='10'`)
	if text, _ := card.Locator(".couple-score span").TextContent(); text != "Média · 1/2" {
		t.Fatal("remoção afetou contador/nota da parceira", text)
	}
	click(t, p, "Para avaliar")
	records(t, p, 1)
	openRating(t, p, b.ids[2])
	saveRating(t, p, "7", "")
	records(t, p, 0)
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Nenhuma obra para avaliar por aqui."}))
	click(t, p, "Histórico")
	records(t, p, 1)
	choose(t, p, "Pessoa", b.f.users[1].ID)
	records(t, p, 1)
	card = highlightedEvent(t, p, b.ids[3])
	visible(t, card.GetByText("1 registro no período", pw.LocatorGetByTextOptions{Exact: pw.Bool(true)}))
	if text, _ := card.Locator(".watch-meta").TextContent(); !strings.Contains(text, "Gabriela") || strings.Contains(text, "17 de outubro") || !strings.Contains(text, "16 de outubro") {
		t.Fatal("resumo não respeitou participante/data do recorte", text)
	}
	visible(t, card.GetByText("Sua nota: 7", pw.LocatorGetByTextOptions{Exact: pw.Bool(true)}))
	click(t, p, "Limpar filtros")
	records(t, p, 5)
	if err := p.SetViewportSize(320, 900); err != nil {
		t.Fatal(err)
	}
	waitJS(t, p, `()=>document.documentElement.scrollWidth<=window.innerWidth`)
	openRating(t, p, b.ids[2])
	visible(t, p.GetByText("Sua nota vale para a série inteira.", pw.PageGetByTextOptions{Exact: pw.Bool(true)}))
	if err := p.GetByLabel("Sua nota", pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).Press("Escape"); err != nil {
		t.Fatal(err)
	}
	waitJS(t, p, `()=>!document.querySelector('dialog')`)
	screenshot(t, p, "series-agrupadas-mobile.png")
}
