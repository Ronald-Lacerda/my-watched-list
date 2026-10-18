package netflix

import (
	"testing"
	"time"
)

func TestWorkIdentity(t *testing.T) {
	body := []byte(`{"viewedItems":[{"movieID":101,"title":"Filme","date":1725233400000},{"movieID":201,"title":"Série: 1","series":200,"seriesTitle":"Série","date":1725233400000},{"movieID":202,"title":"Outra tradução: 2","series":200,"seriesTitle":"Outra tradução","date":1725319800000}]}`)
	e, err := Normalize(body, time.UTC)
	if err != nil || len(e) != 3 || e[0].SourceWorkID != "101" || e[1].SourceWorkID != "200" || e[2].SourceWorkID != "200" || e[1].SourceTitleID == e[2].SourceTitleID {
		t.Fatalf("identidade de obra incorreta: %+v %v", e, err)
	}
}
