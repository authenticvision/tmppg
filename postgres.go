package tmppg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
)

type Postgres struct {
	stdout, stderr io.Writer
	noSync         bool
	logStatement   string
}

type PostgresOption func(pg *Postgres)

func NewPostgres(opts ...PostgresOption) *Postgres {
	pg := &Postgres{
		logStatement: "none",
	}
	for _, opt := range opts {
		opt(pg)
	}
	return pg
}

type slogOut struct {
	logger *slog.Logger
	level  slog.Level
}

func (s slogOut) Write(p []byte) (n int, err error) {
	s.logger.Log(context.Background(), s.level, string(p))
	return len(p), nil
}

func WithLogOutput(logger *slog.Logger, level slog.Level) PostgresOption {
	return func(pg *Postgres) {
		pg.stdout = slogOut{logger.With(slog.String("output", "stdout")), level}
		pg.stderr = slogOut{logger.With(slog.String("output", "stderr")), level}
	}
}

func WithOutput(stdout, stderr io.Writer) PostgresOption {
	return func(pg *Postgres) {
		pg.stdout = stdout
		pg.stderr = stderr
	}
}

// WithoutSync allows disabling syncing if data integrity isn't important
func WithoutSync() PostgresOption {
	return func(pg *Postgres) {
		pg.noSync = true
	}
}

// WithLogStatement configures postgres' log_statement setting.
// Possible values are none, ddl, mod and all. Defaults to none.
func WithLogStatement(setting string) PostgresOption {
	return func(pg *Postgres) {
		pg.logStatement = setting
	}
}

func (pg *Postgres) makeCmd(args ...string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = pg.stdout
	cmd.Stderr = pg.stderr
	return cmd
}

func (pg *Postgres) initDB(dir string) error {
	args := []string{"initdb", "-D", dir, "--no-instructions"}
	if pg.noSync {
		args = append(args, "--no-sync")
	}
	cmd := pg.makeCmd(args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}

func (pg *Postgres) start(dir string) (*exec.Cmd, error) {
	args := []string{
		"postgres", "-D", dir,
		"--listen_addresses=",
		"--unix_socket_directories=" + dir,
		"--log_line_prefix=[%p] ",
		"--log_statement=" + pg.logStatement,
	}
	if pg.noSync {
		args = append(
			args,
			"--fsync=off",
			"--synchronous_commit=off",
			"--full_page_writes=off",
		)
	}
	pgCmd := pg.makeCmd(args...)
	if err := pgCmd.Start(); err != nil {
		return nil, fmt.Errorf("start postgres with args %v: %w", pgCmd.Args, err)
	}
	return pgCmd, nil
}
