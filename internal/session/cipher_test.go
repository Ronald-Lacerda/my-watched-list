package session

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"
)

func TestEnvelope(t *testing.T) {
	c, err := New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte(`{"cookies":[{"value":"segredo-sintético"}]}`)
	a, err := c.Seal("conexão-1", plain)
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Seal("conexão-1", plain)
	if err != nil {
		t.Fatal(err)
	}
	if a[0] != 1 || bytes.Equal(a, b) || bytes.Contains(a, plain) {
		t.Fatal("envelope deve ser versionado, aleatório e cifrado")
	}
	got, err := c.Open("conexão-1", a)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatal("estado não preservado")
	}
	t.Run("conexão diferente", func(t *testing.T) {
		if _, err := c.Open("conexão-2", a); !errors.Is(err, ErrInvalid) {
			t.Fatal("contexto não autenticado")
		}
	})
	t.Run("adulteração", func(t *testing.T) {
		bad := bytes.Clone(a)
		bad[len(bad)-1] ^= 1
		if _, err := c.Open("conexão-1", bad); !errors.Is(err, ErrInvalid) {
			t.Fatal("adulteração aceita")
		}
	})
	t.Run("versão desconhecida", func(t *testing.T) {
		bad := bytes.Clone(a)
		bad[0] = 2
		if _, err := c.Open("conexão-1", bad); !errors.Is(err, ErrInvalid) {
			t.Fatal("versão aceita")
		}
	})
	t.Run("outra chave", func(t *testing.T) {
		other, _ := New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{8}, 32)))
		if _, err := other.Open("conexão-1", a); !errors.Is(err, ErrInvalid) {
			t.Fatal("chave errada aceita")
		}
	})
	t.Run("truncado", func(t *testing.T) {
		if _, err := c.Open("conexão-1", a[:5]); !errors.Is(err, ErrInvalid) {
			t.Fatal("envelope truncado aceito")
		}
	})
}

func TestInvalidKey(t *testing.T) {
	for _, key := range []string{"", "não-base64", base64.StdEncoding.EncodeToString(make([]byte, 16))} {
		if _, err := New(key); err == nil || err.Error() != "SESSION_ENCRYPTION_KEY deve conter 32 bytes em base64" {
			t.Fatal("chave inválida não rejeitada com erro seguro")
		}
	}
}
