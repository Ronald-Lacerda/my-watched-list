//go:build integration

package main

import (
	"bytes"
	"context"
	"mywatchedlist/internal/testdb"
	"strings"
	"testing"
	"time"
)

func TestEnrichCLIWithoutBrowserOrSession(t *testing.T) {
	_, dsn := testdb.New(t)
	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("SESSION_ENCRYPTION_KEY", "")
	t.Setenv("TMDB_READ_ACCESS_TOKEN", "synthetic-token")
	var out bytes.Buffer
	err := runWithBrowser(context.Background(), []string{"enrich", "--retry-unmatched"}, strings.NewReader(""), &out, func(*time.Location, string) browserClient { t.Fatal("enrich abriu navegador"); return nil })
	if err != nil || out.Len() != 0 {
		t.Fatal("CLI enrich depende de sessão", err, out.String())
	}
}
