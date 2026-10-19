package config

import "testing"

func TestDatabaseURLRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil || err.Error() != "DATABASE_URL é obrigatória" {
		t.Fatalf("esperado erro de configuração sem segredos, recebido %v", err)
	}
}
