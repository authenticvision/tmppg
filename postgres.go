package tmppg

import (
	"context"
	"io"
	"log/slog"
	"os/exec"
)

type postgres struct {
	stdout, stderr io.Writer
}

type Option func(pg *postgres)

func newPostgres(opts ...Option) *postgres {
	pg := &postgres{}
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

func WithLogOutput(logger *slog.Logger, level slog.Level) Option {
	return func(pg *postgres) {
		pg.stdout = slogOut{logger.With(slog.String("output", "stdout")), level}
		pg.stderr = slogOut{logger.With(slog.String("output", "stderr")), level}
	}
}

func WithOutput(stdout, stderr io.Writer) Option {
	return func(pg *postgres) {
		pg.stdout = stdout
		pg.stderr = stderr
	}
}

func (pg *postgres) makeCmd(args ...string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = pg.stdout
	cmd.Stderr = pg.stderr
	return cmd
}
