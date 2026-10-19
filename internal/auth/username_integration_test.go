//go:build integration

package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/stdlib"
	"mywatchedlist/api/migrations"
	"mywatchedlist/internal/testdb"
)

func TestUsernameMigrationPreservesCredentials(t *testing.T) {
	ctx := context.Background()
	pool, _ := testdb.At(t, 3)
	encoded, err := HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	var id, otherID string
	if err = pool.QueryRow(ctx, `INSERT INTO users(name,email,password_hash) VALUES('R','r@example.invalid',$1) RETURNING id::text`, encoded).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO users(name,email,password_hash) VALUES('Gabriela','gabriela@example.invalid',$1) RETURNING id::text`, encoded).Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	token, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO auth_sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '14 days')`, hash, id); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO ratings(user_id,source_key,score,comment) VALUES($1,'netflix:work:101',0,'Preservar')`, id); err != nil {
		t.Fatal(err)
	}
	var eventID string
	if err = pool.QueryRow(ctx, `WITH a AS (INSERT INTO streaming_accounts(user_id,service,account_ref,profile_id,profile_name) VALUES($1,'netflix','principal','r','R') RETURNING id) INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title,watched_on) SELECT $1,id,'anterior','Filme','2026-10-18' FROM a RETURNING id::text`, id).Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer db.Close()
	provider, err := migrations.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	var defaults int
	if err = pool.QueryRow(ctx, `SELECT count(DISTINCT username) FROM users WHERE username ~ '^user_[0-9a-f]{32}$'`).Scan(&defaults); err != nil || defaults != 2 {
		t.Fatal("logins provisórios não são únicos", defaults, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE users SET username='ronald' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE users SET username='gabriela' WHERE id=$1`, otherID); err != nil {
		t.Fatal(err)
	}
	s := Service{Pool: pool}
	me, err := s.Me(ctx, token)
	if err != nil || me.ID != id || me.Username != "ronald" {
		t.Fatal("sessão anterior perdida", err)
	}
	for _, tc := range []struct{ username, id string }{{" RONALD ", id}, {"gabriela", otherID}} {
		session, err := s.Login(ctx, tc.username, testPassword)
		if err != nil || session.User.ID != tc.id {
			t.Fatal("senha/identidade alterada pela migração", err)
		}
	}
	var stored, comment, event string
	var score float64
	if err = pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1`, id).Scan(&stored); err != nil || stored != encoded {
		t.Fatal("hash da senha modificado", err)
	}
	if err = pool.QueryRow(ctx, `SELECT score,comment FROM ratings WHERE user_id=$1`, id).Scan(&score, &comment); err != nil || score != 0 || comment != "Preservar" {
		t.Fatal("nota anterior alterada", err)
	}
	if err = pool.QueryRow(ctx, `SELECT id::text FROM watch_events WHERE id=$1 AND user_id=$2`, eventID, id).Scan(&event); err != nil || event != eventID {
		t.Fatal("histórico anterior perdido", err)
	}
	for _, value := range []string{"ronald", "Gabriela", "a", "bad name", "email@example.invalid"} {
		if _, err = pool.Exec(ctx, `UPDATE users SET username=$2 WHERE id=$1`, otherID, value); err == nil {
			t.Fatal("username duplicado/inválido aceito", value)
		}
	}
}

func TestUsernameCanonicalThrottle(t *testing.T) {
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
		value := "ronald"
		if i%2 == 0 {
			value = " RONALD "
		}
		if _, err = s.Login(ctx, value, "Senha incorreta 123"); !errors.Is(err, ErrUnauthorized) {
			t.Fatal("tentativa incorreta não recebeu rejeição", i, err)
		}
	}
	if _, err = s.Login(ctx, "RONALD", testPassword); !errors.Is(err, ErrThrottled) {
		t.Fatal("caixa/espaços contornaram o limite", err)
	}
	var count int
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM auth_attempts`).Scan(&count); err != nil || count != 1 {
		t.Fatal("identidades duplicadas no throttle", count, err)
	}
}
