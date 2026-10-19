package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log/slog"
	"mywatchedlist/internal/config"
	"mywatchedlist/internal/session"
	"mywatchedlist/worker/netflix"
	"mywatchedlist/worker/store"
	"mywatchedlist/worker/syncer"
	"mywatchedlist/worker/tmdb"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

type browserClient interface {
	syncer.Fetcher
	LoginWithState(context.Context, string, []byte, func() error) (netflix.Profile, []byte, error)
}

func run(ctx context.Context, args []string, input io.Reader, output io.Writer) error {
	return runWithBrowser(ctx, args, input, output, func(zone *time.Location, channel string) browserClient {
		return netflix.Browser{Zone: zone, LoginChannel: channel}
	})
}

// O navegador é a fronteira externa substituída nos testes da CLI; banco,
// criptografia, comandos e orquestração continuam sendo os componentes reais.
func runWithBrowser(ctx context.Context, args []string, input io.Reader, output io.Writer, newBrowser func(*time.Location, string) browserClient) error {
	command := "serve"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}
	if command != "serve" && command != "sync" && command != "status" && command != "user-add" && command != "login" {
		return errors.New("comandos: serve, sync, status, user-add, login")
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	name := flags.String("name", "", "nome da pessoa")
	email := flags.String("email", "", "identificador local da pessoa")
	uid := flags.String("user-id", "", "UUID do usuário")
	profile := flags.String("profile", "", "nome exato do perfil Netflix")
	ref := flags.String("account", "principal", "referência da conta")
	id := flags.String("connection", "", "UUID da conexão")
	channel := flags.String("channel", "", "canal do navegador local para login")
	reuse := flags.String("reuse-connection", "", "reaproveitar autenticação local de outra conexão")
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return errors.New("argumentos inválidos; consulte README")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	var cipher *session.Cipher
	if command != "status" && command != "user-add" {
		cipher, err = session.New(os.Getenv("SESSION_ENCRYPTION_KEY"))
		if err != nil {
			return err
		}
	}
	zone, err := time.LoadLocation(value("SYNC_TIMEZONE", "America/Sao_Paulo"))
	if err != nil {
		return errors.New("SYNC_TIMEZONE inválido")
	}
	expression := value("SYNC_CRON", "0 4 * * *")
	if command == "serve" {
		if _, err = syncer.Schedule(expression, zone); err != nil {
			return errors.New("SYNC_CRON inválido")
		}
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return errors.New("configuração de banco inválida")
	}
	defer pool.Close()
	s := store.Store{Pool: pool}
	switch command {
	case "user-add":
		if *name == "" || *email == "" {
			return errors.New("user-add exige --name e --email")
		}
		user, err := s.AddUser(ctx, *name, *email)
		if err != nil {
			return errors.New("não foi possível criar usuário; identificador deve ser único")
		}
		fmt.Fprintln(output, user)
		return nil
	case "status":
		accounts, err := s.Accounts(ctx, true)
		if err != nil {
			return errors.New("falha ao listar conexões")
		}
		for _, a := range accounts {
			last := "nunca"
			if a.LastSync != nil {
				last = *a.LastSync
			}
			code := "-"
			if a.ErrorCode != nil {
				code = *a.ErrorCode
			}
			fmt.Fprintf(output, "%s\t%s\t%s\t%s\t%s\n", a.ID, a.Name, a.Status, last, code)
		}
		return nil
	case "login":
		if *uid == "" || *profile == "" {
			return errors.New("login exige --user-id e --profile")
		}
		b := newBrowser(zone, *channel)
		var initial []byte
		if *reuse != "" {
			list, e := s.Accounts(ctx, true)
			if e != nil {
				return errors.New("falha ao ler conexão de origem")
			}
			for _, a := range list {
				if a.ID == *reuse {
					initial, e = cipher.Open(a.ID, a.EncryptedSession)
					if e != nil {
						return errors.New("sessão de origem inválida")
					}
				}
			}
			if len(initial) == 0 {
				return errors.New("conexão de origem não encontrada")
			}
		}
		p, state, err := b.LoginWithState(ctx, *profile, initial, func() error {
			fmt.Fprintf(output, "Na janela do coletor, faça login e selecione o perfil %s. Mantenha a janela aberta e pressione Enter neste terminal; o coletor fechará a janela após salvar a sessão.\n", *profile)
			_, e := bufio.NewReader(input).ReadString('\n')
			return e
		})
		if err != nil {
			return fmt.Errorf("login não validado: %w", err)
		}
		connection, err := s.Connect(ctx, *uid, *ref, p.ID, p.Name, state, cipher)
		if err != nil {
			return errors.New("não foi possível salvar conexão")
		}
		fmt.Fprintln(output, "Conexão salva:", connection)
		return nil
	}
	r := syncer.Runner{Store: s, Cipher: cipher, Fetcher: newBrowser(zone, "")}
	if token := os.Getenv("TMDB_READ_ACCESS_TOKEN"); token != "" {
		r.Matcher = tmdb.New(token)
	} else {
		slog.Warn("TMDB_READ_ACCESS_TOKEN ausente; metadados permanecerão pendentes")
	}
	if command == "sync" {
		return r.All(ctx, *id)
	}
	slog.Info("worker ativo", "agenda", expression, "fuso", zone.String())
	return r.Serve(ctx, expression, zone)
}
func value(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}
