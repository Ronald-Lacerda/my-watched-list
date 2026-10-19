package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestInvalidKeyStopsBeforeNetwork(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://unused@127.0.0.1:1/unused")
	t.Setenv("SESSION_ENCRYPTION_KEY", "não-é-chave")
	var out bytes.Buffer
	err := run(context.Background(), []string{"sync"}, strings.NewReader(""), &out)
	if err == nil || err.Error() != "SESSION_ENCRYPTION_KEY deve conter 32 bytes em base64" || out.Len() != 0 {
		t.Fatal("configuração inválida não interrompeu antes da rede", err)
	}
}
