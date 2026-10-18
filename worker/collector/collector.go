// Package collector define os contratos da coleta. Adaptadores entram na Fase 1.
package collector

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrLoginRequired = errors.New("a conexão precisa de login")
	ErrParserChanged = errors.New("o formato do histórico mudou")
)

// Connection referencia uma sessão criptografada; não transporta cookies em logs.
type Connection struct {
	ID               string
	ProfileID        string
	EncryptedSession []byte
}

// Event conserva a identidade do provedor mesmo antes do enriquecimento TMDB.
// SourceKey deve ser estável entre sincronizações, sem índice de paginação.
// WatchedOn usa YYYY-MM-DD; WatchedAt fica nil quando a origem só informa o dia.
type Event struct {
	SourceKey     string     `json:"source_key"`
	SourceTitleID string     `json:"source_title_id"`
	SourceTitle   string     `json:"source_title"`
	WatchedOn     string     `json:"watched_on"`
	WatchedAt     *time.Time `json:"watched_at,omitempty"`
	SeasonNumber  *int       `json:"season_number,omitempty"`
	EpisodeNumber *int       `json:"episode_number,omitempty"`
}

type Collector interface {
	Collect(context.Context, Connection) ([]Event, error)
}

// AssistedExtractor será injetado apenas quando o fallback for habilitado na
// Fase 4. A saída deverá passar pela mesma validação da coleta determinística.
// A página deve ser sanitizada e não pode conter cookies nem credenciais.
type AssistedExtractor interface {
	Extract(ctx context.Context, sanitizedPage string, expectedSchema json.RawMessage) ([]Event, error)
}
