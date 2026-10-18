//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"strings"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
)

func click(t *testing.T, p pw.Page, name string) {
	t.Helper()
	if err := p.GetByRole("button", pw.PageGetByRoleOptions{Name: name, Exact: pw.Bool(true)}).Filter(pw.LocatorFilterOptions{Visible: pw.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
}
func fill(t *testing.T, p pw.Page, label, value string) {
	t.Helper()
	if err := p.GetByLabel(label, pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).Fill(value); err != nil {
		t.Fatal(err)
	}
}
func choose(t *testing.T, p pw.Page, label, value string) {
	t.Helper()
	if _, err := p.GetByLabel(label, pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).SelectOption(pw.SelectOptionValues{Values: &[]string{value}}); err != nil {
		t.Fatal(err)
	}
}
func waitJS(t *testing.T, p pw.Page, expression string) {
	t.Helper()
	if _, err := p.WaitForFunction(expression, nil); err != nil {
		t.Fatal(err)
	}
}
func records(t *testing.T, p pw.Page, count int) {
	t.Helper()
	waitJS(t, p, fmt.Sprintf(`()=>document.querySelectorAll('.watch-card').length===%d&&!document.querySelector('.feed-loading')`, count))
}
func eventCard(p pw.Page, id string) pw.Locator { return p.Locator(`[data-event-id="` + id + `"]`) }
func openRating(t *testing.T, p pw.Page, id string) {
	t.Helper()
	if err := eventCard(p, id).GetByRole("button").Click(); err != nil {
		t.Fatal(err)
	}
	visible(t, p.GetByRole("dialog"))
}
func saveRating(t *testing.T, p pw.Page, score, comment string) {
	t.Helper()
	fill(t, p, "Sua nota", score)
	if err := p.Locator("textarea").Fill(comment); err != nil {
		t.Fatal(err)
	}
	click(t, p, "Salvar nota")
	waitJS(t, p, `()=>!document.querySelector('dialog')`)
	visible(t, p.GetByText("Sua nota foi salva.", pw.PageGetByTextOptions{Exact: pw.Bool(true)}))
}

func TestBrowserJournalAndIndividualRatings(t *testing.T) {
	b := browserFixture(t, true)
	p := b.page
	loginBrowser(t, b, 0)
	records(t, p, 6)
	visible(t, p.GetByText("6 registros", pw.PageGetByTextOptions{Exact: pw.Bool(true)}))
	choose(t, p, "Pessoa", b.f.users[0].ID)
	choose(t, p, "Streaming", "netflix")
	fill(t, p, "Data inicial", "2026-10-17")
	fill(t, p, "Data final", "2026-10-18")
	records(t, p, 2)
	visible(t, eventCard(p, b.ids[0]))
	visible(t, eventCard(p, b.ids[2]))
	click(t, p, "Limpar filtros")
	records(t, p, 6)
	fill(t, p, "Data inicial", "2027-01-01")
	records(t, p, 0)
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Nenhuma história neste recorte."}))
	click(t, p, "Limpar filtros")
	records(t, p, 6)
	screenshot(t, p, "fase2-history.png")
	openRating(t, p, b.ids[0])
	screenshot(t, p, "fase2-rating.png")
	if err := p.GetByLabel("Sua nota", pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).Press("Escape"); err != nil {
		t.Fatal(err)
	}
	waitJS(t, p, `()=>!document.querySelector('dialog')`)
	focused, err := eventCard(p, b.ids[0]).GetByRole("button").Evaluate(`e=>e===document.activeElement`, nil)
	if err != nil || focused != true {
		t.Fatal("Escape não devolveu foco", err)
	}
	openRating(t, p, b.ids[0])
	saveRating(t, p, "0", "<script>window.executed=true</script>")
	if _, err = p.Reload(); err != nil {
		t.Fatal(err)
	}
	records(t, p, 6)
	visible(t, eventCard(p, b.ids[0]).GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Editar nota de Filme A"}))
	if text, _ := eventCard(p, b.ids[1]).TextContent(); !strings.Contains(text, "Sua nota: 0") {
		t.Fatal("nota zero não se propagou à obra compartilhada", text)
	}
	openRating(t, p, b.ids[0])
	if value, _ := p.GetByLabel("Sua nota", pw.PageGetByLabelOptions{Exact: pw.Bool(true)}).InputValue(); value != "0" {
		t.Fatal("zero perdido após reload")
	}
	click(t, p, "Cancelar")
	click(t, p, "Sair da conta")
	loginBrowser(t, b, 1)
	records(t, p, 6)
	openRating(t, p, b.ids[1])
	visible(t, p.GetByText("<script>window.executed=true</script>", pw.PageGetByTextOptions{Exact: pw.Bool(true)}))
	if v, _ := p.Evaluate(`()=>window.executed===true`); v == true {
		t.Fatal("comentário executado")
	}
	saveRating(t, p, "10", "Minha opinião independente")
	waitJS(t, p, `()=>document.querySelector('.couple-score strong')?.textContent==='5'`)
	openRating(t, p, b.ids[1])
	fill(t, p, "Sua nota", "8.5")
	click(t, p, "Salvar nota")
	waitJS(t, p, `()=>!document.querySelector('dialog')`)
	waitJS(t, p, `()=>document.querySelector('.couple-score strong')?.textContent==='4,25'`)
	openRating(t, p, b.ids[1])
	click(t, p, "Remover minha nota")
	waitJS(t, p, `()=>!document.querySelector('dialog')`)
	waitJS(t, p, `()=>document.querySelector('.couple-score strong')?.textContent==='0'`)
	if w := b.f.call("GET", "/api/history", "", 0, ""); !strings.Contains(w.Body.String(), `"score":0`) {
		t.Fatal("excluir nota da parceira afetou nota zero")
	}
	click(t, p, "Para avaliar")
	records(t, p, 3)
	click(t, p, "Avaliar Filme A")
	visible(t, p.GetByRole("dialog"))
	saveRating(t, p, "0", "")
	records(t, p, 2)
	click(t, p, "Conexões")
	visible(t, p.GetByRole("heading", pw.PageGetByRoleOptions{Name: "Nossas conexões"}))
	visible(t, p.GetByText("Precisa de login", pw.PageGetByTextOptions{Exact: pw.Bool(true)}))
	if txt, _ := p.Locator("body").TextContent(); strings.Contains(txt, "DO-NOT-EXPOSE") {
		t.Fatal("sessão vazou na interface")
	}
	click(t, p, "Histórico")
	records(t, p, 6)
	if err = p.SetViewportSize(390, 844); err != nil {
		t.Fatal(err)
	}
	waitJS(t, p, `()=>document.documentElement.scrollWidth<=window.innerWidth`)
	screenshot(t, p, "fase2-mobile.png")
	openRating(t, p, b.ids[0])
	waitJS(t, p, `()=>document.querySelector('dialog').scrollWidth<=window.innerWidth`)
	screenshot(t, p, "fase2-mobile-rating.png")
}

func TestBrowserPaginationAndRecovery(t *testing.T) {
	b := browserFixture(t, true)
	p := b.page
	_, err := b.f.pool.Exec(context.Background(), `INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title_id,source_work_id,source_title,match_title,media_type_hint,watched_on) SELECT user_id,streaming_account_id,'extra-'||n, (500+n)::text,(500+n)::text,'Extra '||n,'Extra '||n,'movie','2026-10-18'::date FROM watch_events CROSS JOIN generate_series(1,55) n WHERE id=$1`, b.ids[0])
	if err != nil {
		t.Fatal(err)
	}
	loginBrowser(t, b, 0)
	records(t, p, 50)
	click(t, p, "Carregar mais")
	records(t, p, 61)
	if value, err := p.Evaluate(`()=>new Set([...document.querySelectorAll('.watch-card')].map(e=>e.dataset.eventId)).size`); err != nil || value != 61 {
		t.Fatal("paginação duplicou/perdeu eventos", value, err)
	}
	choose(t, p, "Streaming", "prime_video")
	records(t, p, 1)
	visible(t, eventCard(p, b.ids[4]))
	if err = p.Route("**/api/history?*", func(route pw.Route) {
		route.Fulfill(pw.RouteFulfillOptions{Status: pw.Int(500), ContentType: pw.String("application/json"), Body: `{"error":{"message":"Falha sintética recuperável"}}`})
	}); err != nil {
		t.Fatal(err)
	}
	click(t, p, "Limpar filtros")
	visible(t, p.GetByRole("alert"))
	if err = p.Unroute("**/api/history?*"); err != nil {
		t.Fatal(err)
	}
	click(t, p, "Tentar novamente")
	records(t, p, 50)
	click(t, p, "Carregar mais")
	records(t, p, 61)
	openRating(t, p, b.ids[0])
	fill(t, p, "Sua nota", "7.5")
	if err = p.Locator("textarea").Fill("Preservado durante falha"); err != nil {
		t.Fatal(err)
	}
	if err = p.Route("**/api/events/*/rating", func(route pw.Route) {
		route.Fulfill(pw.RouteFulfillOptions{Status: pw.Int(500), ContentType: pw.String("application/json"), Body: `{"error":{"message":"Não foi possível salvar agora"}}`})
	}); err != nil {
		t.Fatal(err)
	}
	click(t, p, "Salvar nota")
	visible(t, p.GetByRole("dialog").GetByRole("alert"))
	if value, _ := p.Locator("textarea").InputValue(); value != "Preservado durante falha" {
		t.Fatal("erro apagou formulário")
	}
	if err = p.Unroute("**/api/events/*/rating"); err != nil {
		t.Fatal(err)
	}
	click(t, p, "Salvar nota")
	waitJS(t, p, `()=>!document.querySelector('dialog')`)
}
