package netflix

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

	"mywatchedlist/worker/collector"
)

func TestNormalize(t *testing.T) {
	body, err := os.ReadFile("testdata/history.synthetic.json")
	if err != nil {
		t.Fatal(err)
	}
	zone, _ := time.LoadLocation("America/Sao_Paulo")
	events, err := Normalize(body, zone)
	if err != nil || len(events) != 2 {
		t.Fatalf("lote: %v %v", events, err)
	}
	first := events[0]
	if first.SourceTitleID != "101" || first.SourceTitle != "Filme sintético" || first.SourceKey != "netflix:101:1725233400000" || first.WatchedAt.Format(time.RFC3339) != "2024-09-01T23:30:00Z" || first.WatchedOn != "2024-09-01" || first.MediaType != "movie" || first.MatchTitle != "Filme sintético" {
		t.Fatalf("filme: %+v", first)
	}
	second := events[1]
	if second.WatchedAt.Format(time.RFC3339) != "2024-09-02T00:00:00Z" || second.WatchedOn != "2024-09-01" {
		t.Fatal("dia local incorreto na virada UTC")
	}
	if second.MediaType != "tv" || second.MatchTitle != "Série sintética" || *second.SeasonNumber != 0 || *second.EpisodeNumber != 1 {
		t.Fatalf("episódio: %+v", second)
	}
	translated := strings.ReplaceAll(string(body), "Filme sintético", "Outra tradução")
	again, err := Normalize([]byte(translated), zone)
	if err != nil || again[0].SourceKey != first.SourceKey {
		t.Fatal("tradução alterou identidade")
	}
	reversed := `{"viewedItems":[{"movieID":202,"title":"Ep","date":1725235200000},{"movieID":101,"title":"Outro","date":1725233400000}]}`
	again, err = Normalize([]byte(reversed), zone)
	if err != nil || again[1].SourceKey != first.SourceKey {
		t.Fatal("ordem alterou identidade")
	}
}

func TestInvalidPayloadIsAtomic(t *testing.T) {
	zone := time.UTC
	for _, body := range []string{`{}`, `{"viewedItems":null}`, `{"viewedItems":{}}`, `<html>login</html>`, `{"viewedItems":[{"movieID":1,"title":"Ok","date":1725233400000},{"movieID":0,"title":"Inválido","date":1725233400000}]}`, `{"viewedItems":[{"movieID":1,"title":"X","date":1725233400}]}`, `{"viewedItems":[{"movieID":1,"date":1725233400000}]}`, `{"viewedItems":[{"movieID":1,"title":"X","date":1725233400000,"seasonNumber":1}]}`} {
		events, err := Normalize([]byte(body), zone)
		if !errors.Is(err, collector.ErrParserChanged) || events != nil {
			t.Fatalf("payload inválido aceito: %s", body)
		}
	}
	events, err := Normalize([]byte(`{"viewedItems":[]}`), zone)
	if err != nil || len(events) != 0 {
		t.Fatal("histórico vazio rejeitado")
	}
}

// AC 3–4: título explicitamente em branco é um registro indisponível válido;
// ausência/null continuam sendo mudança de contrato. Decisão confirmada no uso real.
func TestUnavailableTitle(t *testing.T) {
	events, err := Normalize([]byte(`{"viewedItems":[{"movieID":42,"title":" ","date":1725233400000}]}`), time.UTC)
	if err != nil || len(events) != 1 {
		t.Fatalf("registro indisponível: %v %v", events, err)
	}
	e := events[0]
	if e.SourceTitle != "Título indisponível na Netflix" || e.SourceTitleID != "42" || e.SourceKey != "netflix:42:1725233400000" || e.MatchTitle != "" || e.WatchedAt.Format(time.RFC3339) != "2024-09-01T23:30:00Z" {
		t.Fatalf("registro perdeu identidade: %+v", e)
	}
	events, err = Normalize([]byte(`{"viewedItems":[{"movieID":42,"title":null,"date":1725233400000}]}`), time.UTC)
	if !errors.Is(err, collector.ErrParserChanged) || events != nil {
		t.Fatal("null aceito como título indisponível")
	}
}
