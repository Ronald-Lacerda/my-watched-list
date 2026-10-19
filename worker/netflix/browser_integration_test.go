//go:build integration

package netflix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// Contrato SSR + Falcor observado na Netflix; dados inteiramente sintéticos.
// A identidade da atividade prevalece sobre o userInfo da página browse.
func TestBrowserFalcor(t *testing.T) {
	for _, mode := range []string{"normal", "vazio", "perfil-na-paginacao", "parser-na-paginacao", "limite", "total-incompleto"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/aui/pathEvaluator/web/test" {
					w.Header().Set("Content-Type", "application/json")
					if mode == "parser-na-paginacao" {
						fmt.Fprint(w, `{"jsonGraph":{"aui":{}}}`)
						return
					}
					guid := "atividade-r"
					if mode == "perfil-na-paginacao" {
						guid = "atividade-outra-pessoa"
					}
					fmt.Fprintf(w, `{"jsonGraph":{"aui":{"viewingActivity":{"$type":"atom","value":{"vhSize":2,"profileInfo":{"guid":%q,"profileName":"R"},"viewedItems":[{"movieID":2,"title":"Segundo filme","date":1725319800000}]}}}}}`, guid)
					return
				}
				items := `[{movieID:1,title:"Primeiro filme",date:1725233400000}]`
				total := 2
				if mode == "vazio" {
					items = `[]`
					total = 0
				}
				fmt.Fprintf(w, `<script>window.netflix={reactContext:{models:{userInfo:{data:{guid:"usuario-diferente",name:"R"}},vaModel:{data:{vhSize:%d,profileInfo:{guid:"atividade-r",profileName:"R"},viewedItems:%s}}}}};</script>`, total, items)
				if mode != "vazio" && mode != "total-incompleto" {
					fmt.Fprint(w, `<button onclick="fetch('/api/aui/pathEvaluator/web/test?paths=%5B%22viewingActivity%22%5D').then(r=>r.json()).then(()=>this.disabled=true)">Mostrar mais</button>`)
				}
			}))
			defer server.Close()
			limit := 3
			if mode == "limite" {
				limit = 1
			}
			batch, err := (Browser{entryURL: server.URL + "/viewingactivity", Zone: time.UTC, MaxPages: limit}).Fetch(context.Background(), nil, "atividade-r")
			if mode == "normal" {
				if err != nil || len(batch.Events) != 2 || batch.Events[0].SourceTitleID != "1" || batch.Events[1].SourceTitleID != "2" || len(batch.State) == 0 {
					t.Fatalf("SSR/Falcor: %+v %v", batch, err)
				}
				return
			}
			if mode == "vazio" {
				if err != nil || len(batch.Events) != 0 || len(batch.State) == 0 {
					t.Fatalf("histórico vazio: %+v %v", batch, err)
				}
				return
			}
			want := collector.ErrParserChanged
			if mode == "perfil-na-paginacao" {
				want = collector.ErrLoginRequired
			}
			if !errors.Is(err, want) || batch.Events != nil {
				t.Fatalf("lote parcial aceito no modo %s: %+v %v", mode, batch, err)
			}
		})
	}
}

// AC 5: o navegador precisa aplicar o estado fornecido, sem herdar outra sessão.
func TestBrowserUsesConnectionState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			fmt.Fprint(w, "login")
			return
		}
		cookie, err := r.Cookie("synthetic-session")
		if err != nil || (cookie.Value != "session-r" && cookie.Value != "session-g") {
			http.Redirect(w, r, "/login", 302)
			return
		}
		guid := "perfil-r"
		if cookie.Value == "session-g" {
			guid = "perfil-g"
		}
		fmt.Fprintf(w, `<script>window.netflix={reactContext:{models:{vaModel:{data:{vhSize:1,profileInfo:{guid:%q,profileName:"Sintético"},viewedItems:[{movieID:1,title:"Teste",date:1725233400000}]}}}}};</script>`, guid)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	for _, value := range []string{"session-r", "session-g", ""} {
		t.Run("cookie-"+value, func(t *testing.T) {
			var state []byte
			if value != "" {
				state, _ = json.Marshal(map[string]any{"cookies": []any{map[string]any{"name": "synthetic-session", "value": value, "domain": u.Hostname(), "path": "/", "expires": -1, "httpOnly": true, "secure": false, "sameSite": "Lax"}}, "origins": []any{}})
			}
			batch, err := (Browser{entryURL: server.URL + "/viewingactivity", Zone: time.UTC}).Fetch(context.Background(), state, "perfil-r")
			if value == "session-r" {
				if err != nil || len(batch.Events) != 1 || batch.Events[0].SourceTitleID != "1" {
					t.Fatalf("sessão da conexão não aplicada: %+v %v", batch, err)
				}
			} else if !errors.Is(err, collector.ErrLoginRequired) || batch.Events != nil {
				t.Fatalf("sessão ausente/de outro perfil aceita: %+v %v", batch, err)
			}
		})
	}
}
