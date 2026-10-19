package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestPasswords(t *testing.T) {
	password := "Minha senha de teste 123"
	a, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	b, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if a == b || !strings.HasPrefix(a, passwordPrefix) || strings.Contains(a, password) || !VerifyPassword(a, password) || VerifyPassword(a, "Outra senha de teste 123") {
		t.Fatal("hash/salt/verificação incorretos")
	}
	for _, bad := range []string{"curta", strings.Repeat("x", 129), string([]byte{0xff, 0xfe})} {
		if _, err = HashPassword(bad); err != ErrPassword {
			t.Fatal("senha fora dos limites aceita")
		}
	}
	for _, bad := range []string{"!disabled", a + "$", strings.Replace(a, "m=19456", "m=999999999", 1), passwordPrefix + "bad$bad"} {
		if VerifyPassword(bad, password) {
			t.Fatal("hash inválido aceito")
		}
	}
	for _, good := range []string{strings.Repeat("x", 12), strings.Repeat("á", 128)} {
		h, e := HashPassword(good)
		if e != nil || !VerifyPassword(h, good) {
			t.Fatal("limite válido rejeitado")
		}
	}
	token, hash, err := NewToken()
	other, _, _ := NewToken()
	if err != nil || len(token) != 43 || len(hash) != 32 || token == other || bytes.Contains(hash, []byte(token)) {
		t.Fatal("token inseguro")
	}
}
