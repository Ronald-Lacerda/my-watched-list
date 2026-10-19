// Package syncer orquestra coleta, persistência e enriquecimento por perfil.
package syncer

import (
	"context"
	"errors"
	"github.com/robfig/cron/v3"
	"log/slog"
	"mywatchedlist/internal/session"
	"mywatchedlist/worker/collector"
	"mywatchedlist/worker/netflix"
	"mywatchedlist/worker/store"
	"mywatchedlist/worker/tmdb"
	"time"
)

type Fetcher interface {
	Fetch(context.Context, []byte, string) (netflix.Batch, error)
}
type Matcher interface {
	Match(context.Context, collector.Event) (store.Title, error)
}
type Runner struct {
	Store   store.Store
	Cipher  *session.Cipher
	Fetcher Fetcher
	Matcher Matcher
}

func Failure(err error) (string, string) {
	switch {
	case errors.Is(err, collector.ErrLoginRequired):
		return "needs_login", "login_required"
	case errors.Is(err, collector.ErrParserChanged):
		return "error", "parser_changed"
	case errors.Is(err, session.ErrInvalid):
		return "error", "session_invalid"
	default:
		return "error", "sync_failed"
	}
}
func (r Runner) Sync(ctx context.Context, a store.Account) error {
	lease, err := r.Store.Lock(ctx, a.ID)
	if err != nil {
		return err
	}
	defer lease.Close()
	var status string
	if err = lease.Conn.QueryRow(ctx, `SELECT session_ciphertext,session_status FROM streaming_accounts WHERE id=$1`, a.ID).Scan(&a.EncryptedSession, &status); err != nil {
		return err
	}
	if status == "needs_login" || status == "disabled" {
		return collector.ErrLoginRequired
	}
	raw, err := r.Cipher.Open(a.ID, a.EncryptedSession)
	if err != nil {
		return r.fail(ctx, lease, a.ID, err)
	}
	batch, err := r.Fetcher.Fetch(ctx, raw, a.ProfileID)
	if err != nil {
		return r.fail(ctx, lease, a.ID, err)
	}
	sealed, err := r.Cipher.Seal(a.ID, batch.State)
	if err != nil {
		return r.fail(ctx, lease, a.ID, err)
	}
	if _, err = lease.Save(ctx, a, batch.Events, sealed); err != nil {
		return r.fail(ctx, lease, a.ID, err)
	}
	if r.Matcher != nil {
		pending, err := lease.Pending(ctx, a.ID)
		if err != nil {
			return err
		}
		for _, p := range pending {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			title, e := r.Matcher.Match(ctx, p.Event)
			if e != nil {
				state := "pending"
				if errors.Is(e, tmdb.ErrNotFound) {
					state = "not_found"
				}
				if errors.Is(e, tmdb.ErrAmbiguous) {
					state = "ambiguous"
				}
				if err = lease.DeferMatch(ctx, p.ID, state); err != nil {
					return err
				}
				if state == "pending" {
					slog.Warn("enriquecimento adiado", "código", "tmdb_unavailable")
					break
				}
				continue
			}
			if err = lease.Link(ctx, p, title); err != nil {
				return err
			}
		}
	}
	slog.Info("sincronização concluída", "conexão", a.ID, "eventos_recebidos", len(batch.Events))
	return nil
}
func (r Runner) fail(ctx context.Context, l *store.Lease, id string, cause error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	status, code := Failure(cause)
	if err := l.Failure(ctx, id, status, code); err != nil {
		return errors.New("não foi possível registrar falha da coleta")
	}
	return errors.New(code)
}
func (r Runner) All(ctx context.Context, onlyID string) error {
	accounts, err := r.Store.Accounts(ctx, false)
	if err != nil {
		return errors.New("falha ao listar conexões")
	}
	var failed bool
	found := onlyID == ""
	for _, a := range accounts {
		if onlyID != "" && a.ID != onlyID {
			continue
		}
		found = true
		if err = r.Sync(ctx, a); err != nil && !errors.Is(err, store.ErrBusy) {
			slog.Warn("sincronização não concluída", "conexão", a.ID)
			failed = true
		}
	}
	if !found {
		return errors.New("conexão não encontrada ou requer login")
	}
	if failed {
		return errors.New("uma ou mais conexões falharam")
	}
	return nil
}
func Schedule(expression string, zone *time.Location) (cron.Schedule, error) {
	if zone == nil {
		return nil, errors.New("fuso inválido")
	}
	return cron.ParseStandard("CRON_TZ=" + zone.String() + " " + expression)
}
func (r Runner) Serve(ctx context.Context, expression string, zone *time.Location) error {
	schedule, err := Schedule(expression, zone)
	if err != nil {
		return errors.New("SYNC_CRON inválido")
	}
	return r.serve(ctx, schedule)
}

func (r Runner) serve(ctx context.Context, schedule cron.Schedule) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := r.All(ctx, ""); err != nil && ctx.Err() == nil {
			slog.Warn("ciclo terminou com falhas")
		}
		next := schedule.Next(time.Now())
		if next.IsZero() {
			return errors.New("agenda sem próxima execução")
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}
