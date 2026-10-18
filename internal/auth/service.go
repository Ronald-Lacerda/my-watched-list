package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
)

var (
	ErrUnauthorized = errors.New("credenciais ou link inválidos")
	ErrThrottled    = errors.New("muitas tentativas; tente novamente em 15 minutos")
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}
type Session struct {
	User      User
	Token     string
	ExpiresAt time.Time
}
type Service struct{ Pool *pgxpool.Pool }

func NewToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, hash[:], nil
}
func tokenHash(token string) ([]byte, bool) {
	if len(token) != 43 {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(raw) != 32 {
		return nil, false
	}
	hash := sha256.Sum256([]byte(token))
	return hash[:], true
}

func (s Service) Invite(ctx context.Context, userID string) (string, error) {
	token, hash, err := NewToken()
	if err != nil {
		return "", err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO auth_invitations(user_id,token_hash,expires_at) VALUES($1,$2,now()+interval '1 hour') ON CONFLICT(user_id) DO UPDATE SET token_hash=excluded.token_hash,created_at=now(),expires_at=excluded.expires_at`, userID, hash)
	if err != nil {
		return "", err
	}
	return token, nil
}

func insertSession(ctx context.Context, tx pgx.Tx, user User) (Session, error) {
	token, hash, err := NewToken()
	if err != nil {
		return Session{}, err
	}
	session := Session{User: user, Token: token}
	err = tx.QueryRow(ctx, `INSERT INTO auth_sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '14 days') RETURNING expires_at`, hash, user.ID).Scan(&session.ExpiresAt)
	return session, err
}

func (s Service) Activate(ctx context.Context, token, password string) (Session, error) {
	hash, ok := tokenHash(token)
	if !ok {
		return Session{}, ErrUnauthorized
	}
	encoded, err := HashPassword(password)
	if err != nil {
		return Session{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(context.Background())
	var user User
	err = tx.QueryRow(ctx, `SELECT u.id::text,u.name,u.email FROM auth_invitations i JOIN users u ON u.id=i.user_id WHERE i.token_hash=$1 AND i.expires_at>now() FOR UPDATE OF i,u`, hash).Scan(&user.ID, &user.Name, &user.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrUnauthorized
	}
	if err != nil {
		return Session{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, user.ID, encoded); err != nil {
		return Session{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM auth_sessions WHERE user_id=$1`, user.ID); err != nil {
		return Session{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM auth_invitations WHERE user_id=$1`, user.ID); err != nil {
		return Session{}, err
	}
	result, err := insertSession(ctx, tx, user)
	if err != nil {
		return Session{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return result, nil
}

func (s Service) Login(ctx context.Context, email, password string) (Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) == 0 || len(email) > 254 {
		return Session{}, ErrUnauthorized
	}
	key := sha256.Sum256([]byte(email))
	var attempts int
	_, err := s.Pool.Exec(ctx, `DELETE FROM auth_attempts WHERE window_start < now()-interval '1 day'`)
	if err != nil {
		return Session{}, err
	}
	err = s.Pool.QueryRow(ctx, `INSERT INTO auth_attempts(identifier_hash,attempts) VALUES($1,1) ON CONFLICT(identifier_hash) DO UPDATE SET attempts=CASE WHEN auth_attempts.window_start<=now()-interval '15 minutes' THEN 1 ELSE auth_attempts.attempts+1 END,window_start=CASE WHEN auth_attempts.window_start<=now()-interval '15 minutes' THEN now() ELSE auth_attempts.window_start END RETURNING attempts`, key[:]).Scan(&attempts)
	if err != nil {
		return Session{}, err
	}
	if attempts > 10 {
		return Session{}, ErrThrottled
	}
	if !ValidPassword(password) {
		return Session{}, ErrUnauthorized
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(context.Background())
	var user User
	var encoded string
	err = tx.QueryRow(ctx, `SELECT id::text,name,email,password_hash FROM users WHERE lower(email)=$1 FOR UPDATE`, email).Scan(&user.ID, &user.Name, &user.Email, &encoded)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !strings.HasPrefix(encoded, passwordPrefix)) {
		_ = argon2.IDKey([]byte(password), []byte("dummy-salt-16byt"), 2, 19456, 1, 32)
		return Session{}, ErrUnauthorized
	}
	if err != nil {
		return Session{}, err
	}
	if !VerifyPassword(encoded, password) {
		return Session{}, ErrUnauthorized
	}
	if _, err = tx.Exec(ctx, `DELETE FROM auth_attempts WHERE identifier_hash=$1`, key[:]); err != nil {
		return Session{}, err
	}
	result, err := insertSession(ctx, tx, user)
	if err != nil {
		return Session{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return result, nil
}

func (s Service) Me(ctx context.Context, token string) (User, error) {
	hash, ok := tokenHash(token)
	if !ok {
		return User{}, ErrUnauthorized
	}
	var user User
	err := s.Pool.QueryRow(ctx, `SELECT u.id::text,u.name,u.email FROM auth_sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.password_hash<>'!disabled'`, hash).Scan(&user.ID, &user.Name, &user.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUnauthorized
	}
	return user, err
}
func (s Service) Logout(ctx context.Context, token string) error {
	hash, ok := tokenHash(token)
	if !ok {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `DELETE FROM auth_sessions WHERE token_hash=$1`, hash)
	return err
}
