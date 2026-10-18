//go:build integration

package httpapi

import (
	"fmt"
	"strings"
	"testing"
)

func TestUsernameLoginContract(t *testing.T) {
	f := fixtureApp(t)
	for _, test := range []struct {
		username string
		actor    int
	}{{" RONALD ", 0}, {"gabriela", 1}} {
		w := f.call("POST", "/api/auth/login", fmt.Sprintf(`{"username":%q,"password":%q}`, test.username, appTestPassword), -1, f.origin)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"id":"`+f.users[test.actor].ID+`"`) || !strings.Contains(w.Body.String(), `"username":"`+f.users[test.actor].Username+`"`) || strings.Contains(w.Body.String(), `"email"`) {
			t.Fatal("login por username não identificou a pessoa", w.Code, w.Body.String())
		}
	}
	for _, username := range []string{"r@example.invalid", "inexistente", "ron ald", "ab"} {
		if w := f.call("POST", "/api/auth/login", fmt.Sprintf(`{"username":%q,"password":%q}`, username, appTestPassword), -1, f.origin); w.Code != 401 {
			t.Fatal("identificador inválido não recebeu erro genérico", w.Code)
		}
	}
	if w := f.call("POST", "/api/auth/login", fmt.Sprintf(`{"email":"r@example.invalid","password":%q}`, appTestPassword), -1, f.origin); w.Code != 400 {
		t.Fatal("contrato antigo de e-mail ainda aceito", w.Code)
	}
}
