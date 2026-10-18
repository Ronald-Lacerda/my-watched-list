//go:build integration

package auth

import (
	"context"
	"errors"
	"mywatchedlist/internal/testdb"
	"sync"
	"testing"
	"time"
)

const testPassword = "Senha sintética para testes 123"

func userFixture(t *testing.T) (Service, string) {
	t.Helper()
	pool, _ := testdb.New(t)
	s := Service{Pool: pool}
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO users(name,email,password_hash) VALUES('R','r@example.invalid','!disabled') RETURNING id::text`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return s, id
}

func TestCredentialLifecycle(t *testing.T) {
	ctx := context.Background()
	s, id := userFixture(t)
	if _, err := s.Login(ctx, "r@example.invalid", testPassword); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("usuário desabilitado autenticou")
	}
	token, err := s.Invite(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	var remaining float64
	if err = s.Pool.QueryRow(ctx, `SELECT extract(epoch from(expires_at-now())) FROM auth_invitations WHERE user_id=$1`, id).Scan(&remaining); err != nil || remaining < 3590 || remaining > 3600 {
		t.Fatal("TTL de convite incorreto", remaining, err)
	}
	first, err := s.Activate(ctx, token, testPassword)
	if err != nil || first.User.ID != id || time.Until(first.ExpiresAt) < 14*24*time.Hour-time.Minute || time.Until(first.ExpiresAt) > 14*24*time.Hour+time.Minute {
		t.Fatal("ativação incorreta", err)
	}
	hash, _ := tokenHash(first.Token)
	var stored []byte
	if err = s.Pool.QueryRow(ctx, `SELECT token_hash FROM auth_sessions WHERE user_id=$1`, id).Scan(&stored); err != nil || string(stored) != string(hash) || string(stored) == first.Token {
		t.Fatal("token não cifrado por hash", err)
	}
	if _, err = s.Activate(ctx, token, testPassword); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("convite reutilizado")
	}
	me, err := s.Me(ctx, first.Token)
	if err != nil || me.ID != id || me.Name != "R" {
		t.Fatal("sessão não autenticou", err)
	}
	logged, err := s.Login(ctx, " R@EXAMPLE.INVALID ", testPassword)
	if err != nil || logged.User.ID != id {
		t.Fatal("login correto rejeitado", err)
	}
	if err = s.Logout(ctx, logged.Token); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Me(ctx, logged.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("logout não revogou")
	}
	newInvite, err := s.Invite(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	newSession, err := s.Activate(ctx, newInvite, "Outra senha sintética 123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Me(ctx, first.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("senha nova não revogou sessão anterior")
	}
	if _, err = s.Login(ctx, "r@example.invalid", testPassword); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("senha anterior ainda válida")
	}
	if _, err = s.Pool.Exec(ctx, `UPDATE auth_sessions SET created_at=now()-interval '15 days',expires_at=now()-interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Me(ctx, newSession.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("sessão expirada aceita")
	}
	if _, err = s.Me(ctx, "inválido"); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("token inválido aceito")
	}
}

func TestInvitationConcurrencyAndExpiry(t *testing.T) {
	ctx := context.Background()
	s, id := userFixture(t)
	old, err := s.Invite(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.Invite(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Activate(ctx, old, testPassword); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("convite substituído aceito")
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Activate(ctx, token, testPassword); results <- err }()
	}
	wg.Wait()
	close(results)
	success, denied := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrUnauthorized) {
			denied++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || denied != 1 {
		t.Fatal("convite consumido mais de uma vez", success, denied)
	}
	token, err = s.Invite(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Pool.Exec(ctx, `UPDATE auth_invitations SET created_at=now()-interval '2 hours',expires_at=now()-interval '1 hour'`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Activate(ctx, token, testPassword); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("convite expirado aceito")
	}
}

func TestLoginThrottle(t *testing.T) {
	ctx := context.Background()
	s, id := userFixture(t)
	token, err := s.Invite(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Activate(ctx, token, testPassword); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err = s.Login(ctx, "r@example.invalid", "Senha incorreta 123"); !errors.Is(err, ErrUnauthorized) {
			t.Fatal("tentativa inválida sem erro genérico", i, err)
		}
	}
	if _, err = s.Login(ctx, "r@example.invalid", testPassword); !errors.Is(err, ErrThrottled) {
		t.Fatal("11ª tentativa não bloqueada", err)
	}
	if _, err = s.Pool.Exec(ctx, `UPDATE auth_attempts SET window_start=now()-interval '16 minutes'`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Login(ctx, "r@example.invalid", testPassword); err != nil {
		t.Fatal("janela não expirou", err)
	}
	var count int
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM auth_attempts`).Scan(&count); err != nil || count != 0 {
		t.Fatal("sucesso não zerou tentativas", count, err)
	}
	if _, err = s.Login(ctx, "inexistente@example.invalid", testPassword); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("usuário desconhecido não recebeu erro genérico", err)
	}
}
