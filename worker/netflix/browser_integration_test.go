//go:build integration

package netflix

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mywatchedlist/worker/collector"
)

// Site controlado: comprova Chromium e captura de rede, não simula validação
// da conta Netflix real. O contrato externo ainda exige captura autorizada.
func TestBrowserCapture(t *testing.T) {
	for _, mode := range []string{"normal", "perfil-errado", "parser", "repetida", "login"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/login" {
					fmt.Fprint(w, "login")
					return
				}
				if r.URL.Path == "/api/viewingactivity" {
					w.Header().Set("Content-Type", "application/json")
					if mode == "parser" {
						fmt.Fprint(w, `{"outro":[]}`)
						return
					}
					stamp := "1725233400000"
					if r.URL.Query().Get("page") == "2" && mode != "repetida" {
						stamp = "1725319800000"
					}
					fmt.Fprintf(w, `{"viewedItems":[{"movieID":1,"title":"Teste","date":%s}]}`, stamp)
					return
				}
				if mode == "login" {
					http.Redirect(w, r, "/login", 302)
					return
				}
				fmt.Fprint(w, `<script>window.netflix={reactContext:{models:{memberContext:{data:{userInfo:{guid:"perfil-1",name:"R"}}}}}}; fetch('/api/viewingactivity').then(r=>r.json()).then(()=>{document.querySelector('button').hidden=false});</script><button hidden onclick="fetch('/api/viewingactivity?page=2').then(r=>r.json()).then(()=>this.remove())">Mostrar mais</button>`)
			}))
			defer server.Close()
			expected := "perfil-1"
			if mode == "perfil-errado" {
				expected = "outro"
			}
			b := Browser{entryURL: server.URL + "/viewingactivity", Zone: time.UTC, MaxPages: 3}
			batch, err := b.Fetch(context.Background(), nil, expected)
			if mode == "normal" {
				if err != nil || len(batch.Events) != 2 || batch.Events[0].SourceKey == batch.Events[1].SourceKey || len(batch.State) == 0 {
					t.Fatalf("captura: %+v %v", batch, err)
				}
				return
			}
			want := collector.ErrParserChanged
			if mode == "perfil-errado" || mode == "login" {
				want = collector.ErrLoginRequired
			}
			if !errors.Is(err, want) || batch.Events != nil {
				t.Fatalf("modo %s: %v", mode, err)
			}
		})
	}
}
