//go:build integration

package main

import (
	"bytes"
	"context"
	"mywatchedlist/internal/auth"
	"mywatchedlist/internal/testdb"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvitationCommand(t *testing.T) {
	ctx := context.Background()
	pool, dsn := testdb.New(t)
	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("APP_ORIGINS", "http://127.0.0.1:8080")
	if _, err := pool.Exec(ctx, `INSERT INTO users(name,email,password_hash) VALUES('R','r@example.invalid','!disabled')`); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "acesso.html")
	var out bytes.Buffer
	if err := run(ctx, []string{"invite", "--email", "r@example.invalid", "--output", path}, &out); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "#ativar=", 2)
	if len(parts) != 2 || len(parts[1]) < 43 {
		t.Fatal("arquivo não contém link de ativação")
	}
	token := parts[1][:43]
	if strings.Contains(out.String(), token) || !strings.Contains(out.String(), path) {
		t.Fatal("convite exposto na saída")
	}
	if _, err := (auth.Service{Pool: pool}).Activate(ctx, token, "Senha sintética para testes 123"); err != nil {
		t.Fatal("convite não funcional", err)
	}
	if err = run(ctx, []string{"invite", "--email", "r@example.invalid", "--output", path}, &out); err == nil {
		t.Fatal("arquivo existente sobrescrito")
	}
	again, _ := os.ReadFile(path)
	if !bytes.Equal(body, again) {
		t.Fatal("arquivo foi modificado")
	}
}
