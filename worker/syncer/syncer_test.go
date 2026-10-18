package syncer

import (
	"errors"
	"mywatchedlist/internal/session"
	"mywatchedlist/worker/collector"
	"testing"
	"time"
	_ "time/tzdata"
)

func TestFailureCodes(t *testing.T) {
	for _, tt := range []struct {
		err          error
		status, code string
	}{{collector.ErrLoginRequired, "needs_login", "login_required"}, {collector.ErrParserChanged, "error", "parser_changed"}, {session.ErrInvalid, "error", "session_invalid"}, {errors.New("cookie=segredo"), "error", "sync_failed"}} {
		status, code := Failure(tt.err)
		if status != tt.status || code != tt.code {
			t.Fatalf("classificação incorreta %s %s", status, code)
		}
	}
}
func TestDailySchedule(t *testing.T) {
	zone, _ := time.LoadLocation("America/Sao_Paulo")
	s, err := Schedule("0 4 * * *", zone)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 18, 4, 0, 0, 0, zone)
	if got := s.Next(now); !got.Equal(now.Add(24 * time.Hour)) {
		t.Fatal("cron diário incorreto", got)
	}
	if _, err = Schedule("inválido", zone); err == nil {
		t.Fatal("cron inválido aceito")
	}
}
