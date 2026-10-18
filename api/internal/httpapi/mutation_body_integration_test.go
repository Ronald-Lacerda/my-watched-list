//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOptionalMutationBodies(t *testing.T) {
	for _, endpoint := range []string{"logout", "rating"} {
		for _, chunked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/chunked=%t", endpoint, chunked), func(t *testing.T) {
				f, ids := historyFixture(t)
				for actor, score := range []int{7, 4} {
					if w := f.call("PUT", "/api/events/"+ids[0]+"/rating", fmt.Sprintf(`{"score":%d}`, score), actor, f.origin); w.Code != 200 {
						t.Fatal("nota inicial falhou")
					}
				}
				method, path := "POST", "/api/auth/logout"
				if endpoint == "rating" {
					method, path = "DELETE", "/api/events/"+ids[0]+"/rating"
				}
				for _, tc := range []struct {
					name, body, kind string
					status           int
				}{
					{"syntax", "{", "application/json", 400},
					{"type", "{}", "text/plain", 415},
					{"size", strings.Repeat(" ", 65537), "application/json", 413},
					{"actor", fmt.Sprintf(`{"user_id":%q}`, f.users[1].ID), "application/json", 400},
					{"null", "null", "application/json", 400},
					{"array", "[]", "application/json", 400},
					{"extra", "{} {}", "application/json", 400},
				} {
					t.Run(tc.name, func(t *testing.T) {
						r := httptest.NewRequest(method, path, strings.NewReader(tc.body))
						r.Header.Set("Content-Type", tc.kind)
						r.Header.Set("Origin", f.origin)
						r.AddCookie(f.cookies[0])
						if chunked {
							r.ContentLength = -1
							r.TransferEncoding = []string{"chunked"}
						}
						w := httptest.NewRecorder()
						f.handler.ServeHTTP(w, r)
						if w.Code != tc.status {
							t.Fatalf("corpo inválido aceito: status=%d esperado=%d", w.Code, tc.status)
						}
						if w := f.call("GET", "/api/auth/me", "", 0, ""); w.Code != 200 {
							t.Fatal("corpo inválido revogou sessão")
						}
						var count int
						var sum float64
						if err := f.pool.QueryRow(context.Background(), `SELECT count(*),sum(score) FROM ratings`).Scan(&count, &sum); err != nil || count != 2 || sum != 11 {
							t.Fatal("corpo inválido alterou notas", count, sum, err)
						}
					})
				}
				r := httptest.NewRequest(method, path, strings.NewReader("{}"))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Origin", f.origin)
				r.AddCookie(f.cookies[0])
				if chunked {
					r.ContentLength = -1
					r.TransferEncoding = []string{"chunked"}
				}
				w := httptest.NewRecorder()
				f.handler.ServeHTTP(w, r)
				expected := 204
				if endpoint == "rating" {
					expected = 200
				}
				if w.Code != expected {
					t.Fatal("objeto vazio válido rejeitado", w.Code)
				}
			})
		}
	}
}
