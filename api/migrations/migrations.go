// Package migrations incorpora o schema no executável para evitar divergência
// entre o binário e arquivos SQL presentes no servidor.
package migrations

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

func New(db *sql.DB) (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, db, files)
}
