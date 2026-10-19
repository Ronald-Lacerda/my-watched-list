package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"mywatchedlist/internal/auth"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) > 0 && args[0] == "username" {
		return assignUsername(ctx, args[1:], out)
	}
	if len(args) == 0 || args[0] != "invite" {
		return errors.New("uso: admin invite --username USUARIO --output ARQUIVO.html; admin username --email EMAIL --username USUARIO")
	}
	flags := flag.NewFlagSet("invite", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	email := flags.String("email", "", "e-mail da pessoa existente")
	username := flags.String("username", "", "login da pessoa existente")
	path := flags.String("output", "", "novo arquivo local do convite")
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 || (*email == "") == (*username == "") || *path == "" {
		return errors.New("invite exige --username (ou --email legado) e --output")
	}
	if *username != "" {
		var valid bool
		*username, valid = auth.NormalizeUsername(*username)
		if !valid {
			return errors.New("username inválido: use 3 a 40 letras, números ou _, começando com letra")
		}
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL obrigatória")
	}
	origin := strings.Split(os.Getenv("APP_ORIGINS"), ",")[0]
	if origin == "" {
		origin = "http://127.0.0.1:8080"
	}
	u, err := url.Parse(strings.TrimRight(origin, "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return errors.New("APP_ORIGINS deve conter origens HTTP/HTTPS válidas")
	}
	f, err := os.OpenFile(*path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("não foi possível criar arquivo novo do convite; não sobrescreva um arquivo existente")
	}
	complete := false
	defer func() {
		f.Close()
		if !complete {
			os.Remove(*path)
		}
	}()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return errors.New("configuração de banco inválida")
	}
	defer pool.Close()
	var uid, name string
	if err = pool.QueryRow(ctx, `SELECT id::text,name FROM users WHERE username=$1 OR lower(email)=lower($2)`, *username, strings.TrimSpace(*email)).Scan(&uid, &name); err != nil {
		return errors.New("não foi possível localizar a pessoa")
	}
	token, err := (auth.Service{Pool: pool}).Invite(ctx, uid)
	if err != nil {
		return errors.New("não foi possível gerar convite")
	}
	u.Fragment = "ativar=" + token
	link := html.EscapeString(u.String())
	_, err = fmt.Fprintf(f, `<!doctype html><html lang="pt-BR"><meta charset="utf-8"><meta name="referrer" content="no-referrer"><meta http-equiv="refresh" content="0;url=%s"><title>Primeiro acesso — My Watched List</title><p>Primeiro acesso de %s.</p><a href="%s">Criar minha senha</a><p>Este convite expira em 1 hora e só pode ser usado uma vez.</p></html>`, link, html.EscapeString(name), link)
	if err != nil {
		return errors.New("não foi possível gravar convite")
	}
	if err = f.Close(); err != nil {
		return errors.New("não foi possível concluir arquivo")
	}
	complete = true
	fmt.Fprintf(out, "Convite salvo em %s. Abra o arquivo no navegador em até 1 hora; apague-o após criar sua senha.\n", *path)
	return nil
}

// Este comando administrativo só altera o identificador de login.
func assignUsername(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("username", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	email := flags.String("email", "", "identificador legado da pessoa")
	value := flags.String("username", "", "novo login")
	if flags.Parse(args) != nil || flags.NArg() != 0 || strings.TrimSpace(*email) == "" {
		return errors.New("username exige --email da conta existente e --username")
	}
	username, valid := auth.NormalizeUsername(*value)
	if !valid {
		return errors.New("username inválido: use 3 a 40 letras, números ou _, começando com letra")
	}
	if os.Getenv("DATABASE_URL") == "" {
		return errors.New("DATABASE_URL obrigatória")
	}
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return errors.New("configuração de banco inválida")
	}
	defer pool.Close()
	result, err := pool.Exec(ctx, `UPDATE users SET username=$2 WHERE lower(email)=lower($1)`, strings.TrimSpace(*email), username)
	if err != nil || result.RowsAffected() != 1 {
		return errors.New("não foi possível atribuir o username; confira a conta e se o login já está em uso")
	}
	fmt.Fprintf(out, "Login atualizado para %s.\n", username)
	return nil
}
