//go:build integration

package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Cada execução usa um schema aleatório em banco descartável, nunca public.
func TestSchema(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL é obrigatória para os testes de integração")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal("TEST_DATABASE_URL inválida")
	}
	admin := stdlib.OpenDB(*cfg)
	defer admin.Close()
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)
	defer db.Close()
	p, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Down(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if applied, err := p.Up(ctx); err != nil || len(applied) != 0 {
		t.Fatalf("reaplicação: %v %v", applied, err)
	}
	for _, table := range []string{"users", "streaming_accounts", "titles", "episodes", "watch_events", "ratings"} {
		var name string
		if err := db.QueryRow("SELECT to_regclass($1)::text", table).Scan(&name); err != nil || name != table {
			t.Fatalf("tabela %s: %q %v", table, name, err)
		}
	}

	t.Run("replay preserva evento e permite revisita", func(t *testing.T) {
		tx := fixture(t, db)
		insert := eventSQL + " ON CONFLICT (streaming_account_id, source_event_key) DO NOTHING"
		for _, key := range []string{"evento-1", "evento-1", "evento-2"} {
			mustExec(t, tx, insert, key)
		}
		var count int
		if err := tx.QueryRow("SELECT count(*) FROM watch_events").Scan(&count); err != nil || count != 2 {
			t.Fatalf("eventos = %d, esperado 2: %v", count, err)
		}
		var title string
		var day string
		var titleID, episodeID, instant sql.NullString
		err := tx.QueryRow("SELECT source_title, watched_on::text, title_id::text, episode_id::text, watched_at::text FROM watch_events WHERE source_event_key='evento-1'").Scan(&title, &day, &titleID, &episodeID, &instant)
		if err != nil || title != "Título de origem" || day != "2026-10-18" || titleID.Valid || episodeID.Valid || instant.Valid {
			t.Fatalf("evento sem metadados ou horário não foi preservado: %v", err)
		}
	})
	for _, score := range []string{"0", "0.5", "10", "-0.5", "10.5", "7.2", "8.01"} {
		t.Run("nota "+score, func(t *testing.T) {
			tx := fixture(t, db)
			_, err := tx.Exec("INSERT INTO ratings (user_id,title_id,score) VALUES ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000003',$1)", score)
			valid := score == "0" || score == "0.5" || score == "10"
			if valid && err != nil {
				t.Fatal(err)
			}
			if !valid && err == nil {
				t.Fatalf("nota inválida %s aceita", score)
			}
		})
	}
	t.Run("pessoa precisa ser dona da conexão", func(t *testing.T) {
		tx := fixture(t, db)
		mustExec(t, tx, "INSERT INTO users(id,name,email,password_hash) VALUES ('00000000-0000-0000-0000-000000000009','Outra pessoa','outra@example.invalid','hash-de-teste')")
		_, err := tx.Exec("INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title,watched_on) VALUES ('00000000-0000-0000-0000-000000000009','00000000-0000-0000-0000-000000000002','chave','Título','2026-10-18')")
		if err == nil {
			t.Fatal("evento de pessoa diferente aceito")
		}
	})
	t.Run("episódio precisa pertencer ao título", func(t *testing.T) {
		tx := fixture(t, db)
		mustExec(t, tx, "INSERT INTO titles(id,tmdb_id,media_type,title) VALUES ('00000000-0000-0000-0000-000000000005',2,'tv','Outra série')")
		_, err := tx.Exec(eventSQL+" RETURNING id", "episódio-errado")
		if err != nil {
			t.Fatal(err)
		}
		_, err = tx.Exec("UPDATE watch_events SET title_id='00000000-0000-0000-0000-000000000005',episode_id='00000000-0000-0000-0000-000000000004'")
		if err == nil {
			t.Fatal("episódio de outro título aceito")
		}
	})
	t.Run("episódio não pode pertencer a filme", func(t *testing.T) {
		tx := fixture(t, db)
		mustExec(t, tx, "INSERT INTO titles(id,tmdb_id,media_type,title) VALUES ('00000000-0000-0000-0000-000000000005',2,'movie','Filme')")
		_, err := tx.Exec("INSERT INTO episodes(title_id,season_number,episode_number) VALUES ('00000000-0000-0000-0000-000000000005',1,1)")
		if err == nil {
			t.Fatal("episódio vinculado a filme aceito")
		}
	})
}

const eventSQL = `INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title,watched_on) VALUES ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002',$1,'Título de origem','2026-10-18')`

func fixture(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	mustExec(t, tx, "INSERT INTO users(id,name,email,password_hash) VALUES ('00000000-0000-0000-0000-000000000001','Pessoa','pessoa@example.invalid','hash-de-teste')")
	mustExec(t, tx, "INSERT INTO streaming_accounts(id,user_id,service,account_ref,profile_id,profile_name) VALUES ('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','netflix','conta','perfil','Pessoa')")
	mustExec(t, tx, "INSERT INTO titles(id,tmdb_id,media_type,title) VALUES ('00000000-0000-0000-0000-000000000003',1,'tv','Série')")
	mustExec(t, tx, "INSERT INTO episodes(id,title_id,season_number,episode_number) VALUES ('00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000003',1,1)")
	return tx
}

func mustExec(t *testing.T, tx *sql.Tx, query string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
