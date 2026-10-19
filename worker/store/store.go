// Package store persiste a coleta e mantém a exclusão concorrente por conexão.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"mywatchedlist/internal/session"
	"mywatchedlist/worker/collector"
)

var ErrBusy = errors.New("conexão já está em sincronização")

type Store struct{ Pool *pgxpool.Pool }
type Account struct {
	collector.Connection
	UserID    string
	Name      string
	Status    string
	ErrorCode *string
	LastSync  *string
}
type Lease struct{ Conn *pgxpool.Conn }

func (s Store) Accounts(ctx context.Context, all bool) ([]Account, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text,user_id::text,profile_id,profile_name,session_status,session_ciphertext,last_error_code,last_sync_at::text FROM streaming_accounts WHERE service='netflix' AND ($1 OR (session_status IN ('active','error') AND session_ciphertext IS NOT NULL)) ORDER BY created_at`, all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.UserID, &a.ProfileID, &a.Name, &a.Status, &a.EncryptedSession, &a.ErrorCode, &a.LastSync); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (s Store) Lock(ctx context.Context, id string) (*Lease, error) {
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	var locked bool
	err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1,0))`, id).Scan(&locked)
	if err != nil || !locked {
		conn.Release()
		if err != nil {
			return nil, err
		}
		return nil, ErrBusy
	}
	return &Lease{Conn: conn}, nil
}
func (l *Lease) Close() {
	// Fechar conexão em caso de erro impede devolver ao pool um lock residual.
	ctx, cancel := context.WithTimeout(context.Background(), 5e9)
	defer cancel()
	if _, err := l.Conn.Exec(ctx, `SELECT pg_advisory_unlock_all()`); err != nil {
		_ = l.Conn.Conn().Close(ctx)
	}
	l.Conn.Release()
}
func (l *Lease) Failure(ctx context.Context, id, status, code string) error {
	_, err := l.Conn.Exec(ctx, `UPDATE streaming_accounts SET session_status=$2,last_attempt_at=now(),last_error_code=$3 WHERE id=$1`, id, status, code)
	return err
}
func (l *Lease) Save(ctx context.Context, a Account, events []collector.Event, fresh []byte) (int, error) {
	tx, err := l.Conn.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(context.Background())
	count := 0
	for _, e := range events {
		day, err := time.Parse(time.DateOnly, e.WatchedOn)
		if err != nil {
			return 0, errors.New("data de evento inválida")
		}
		tag, err := tx.Exec(ctx, `INSERT INTO watch_events(user_id,streaming_account_id,source_event_key,source_title_id,source_title,watched_on,watched_at,match_title,media_type_hint,season_number,episode_number,source_work_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT(streaming_account_id,source_event_key) DO UPDATE SET source_title=excluded.source_title,match_title=excluded.match_title,media_type_hint=excluded.media_type_hint,season_number=excluded.season_number,episode_number=excluded.episode_number,source_work_id=excluded.source_work_id`, a.UserID, a.ID, e.SourceKey, e.SourceTitleID, e.SourceTitle, day, e.WatchedAt, e.MatchTitle, e.MediaType, e.SeasonNumber, e.EpisodeNumber, e.SourceWorkID)
		if err != nil {
			return 0, err
		}
		count += int(tag.RowsAffected())
	}
	_, err = tx.Exec(ctx, `UPDATE streaming_accounts SET session_status='active',last_attempt_at=now(),last_sync_at=now(),last_error_code=NULL,session_ciphertext=COALESCE($2,session_ciphertext),session_key_version=1 WHERE id=$1`, a.ID, fresh)
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return count, nil
}

func (s Store) AddUser(ctx context.Context, name, email string) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `INSERT INTO users(name,email,password_hash) VALUES($1,$2,'!disabled') RETURNING id::text`, name, email).Scan(&id)
	return id, err
}
func (s Store) Connect(ctx context.Context, userID, accountRef, profileID, name string, state []byte, c *session.Cipher) (string, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(context.Background())
	var id string
	err = tx.QueryRow(ctx, `SELECT id::text FROM streaming_accounts WHERE user_id=$1 AND service='netflix' AND account_ref=$2 AND profile_id=$3`, userID, accountRef, profileID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&id)
	}
	if err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, id); err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO streaming_accounts(id,user_id,service,account_ref,profile_id,profile_name) VALUES($1,$2,'netflix',$3,$4,$5) ON CONFLICT(user_id,service,account_ref,profile_id) DO NOTHING`, id, userID, accountRef, profileID, name)
	if err != nil {
		return "", err
	}
	sealed, err := c.Seal(id, state)
	if err != nil {
		return "", err
	}
	tag, err := tx.Exec(ctx, `UPDATE streaming_accounts SET profile_name=$3,session_ciphertext=$2,session_key_version=1,session_status='active',last_error_code=NULL WHERE id=$1`, id, sealed, name)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() != 1 {
		return "", ErrBusy
	}
	return id, tx.Commit(ctx)
}
