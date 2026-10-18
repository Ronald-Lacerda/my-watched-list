package auth

import (
	"strings"
	"testing"
)

func TestNormalizeUsername(t *testing.T) {
	for _, value := range []string{"ronald", "gabriela", " RONALD ", "user_123", "a" + strings.Repeat("b", 39)} {
		got, ok := NormalizeUsername(value)
		if !ok || got != strings.ToLower(strings.TrimSpace(value)) {
			t.Fatal("username válido rejeitado", value)
		}
	}
	for _, value := range []string{"", "r", "ab", "1ronald", "ronald@example.invalid", "ron ald", "ronáld", "../ronald", "a" + strings.Repeat("b", 40)} {
		if _, ok := NormalizeUsername(value); ok {
			t.Fatal("username inválido aceito", value)
		}
	}
}
