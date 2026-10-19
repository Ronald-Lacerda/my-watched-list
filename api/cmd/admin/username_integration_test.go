//go:build integration

package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"mywatchedlist/internal/auth"
	"mywatchedlist/internal/testdb"
)

func TestUsernameAdministration(t *testing.T) {
	ctx := context.Background()
	pool, dsn := testdb.New(t)
	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("APP_ORIGINS", "http://app.test")
	var id, other string
	for i, email := range []string{"r@example.invalid", "gabriela@example.invalid"} {
		var uid string
		if err := pool.QueryRow(ctx, `INSERT INTO users(name,email,password_hash) VALUES('Pessoa',$1,'!disabled') RETURNING id::text`, email).Scan(&uid); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			id = uid
		} else {
			other = uid
		}
	}
	var out bytes.Buffer
	if err := run(ctx, []string{"username", "--email", "r@example.invalid", "--username", " RONALD "}, &out); err != nil {
		t.Fatal(err)
	}
	if err := run(ctx, []string{"username", "--email", "gabriela@example.invalid", "--username", "ronald"}, &out); err == nil {
		t.Fatal("colisão de username aceita")
	}
	if err := run(ctx, []string{"username", "--email", "r@example.invalid", "--username", "bad name"}, &out); err == nil {
		t.Fatal("username inválido aceito")
	}
	var username, password string
	if err := pool.QueryRow(ctx, `SELECT username,password_hash FROM users WHERE id=$1`, id).Scan(&username, &password); err != nil || username != "ronald" || password != "!disabled" {
		t.Fatal("alteração de login afetou credenciais", err)
	}
	if err := run(ctx, []string{"username", "--email", "gabriela@example.invalid", "--username", "gabriela"}, &out); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "convite.html")
	if err := run(ctx, []string{"invite", "--username", "gabriela", "--output", path}, &out); err != nil {
		t.Fatal(err)
	}
	var invited string
	if err := pool.QueryRow(ctx, `SELECT user_id::text FROM auth_invitations`).Scan(&invited); err != nil || invited != other {
		t.Fatal("convite de username associou pessoa errada", err)
	}
	if _, err := (auth.Service{Pool: pool}).Login(ctx, "ronald", "Senha sintética para testes 123"); err == nil {
		t.Fatal("trocar username ativou senha desabilitada")
	}
}
