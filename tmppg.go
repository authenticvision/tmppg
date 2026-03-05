package tmppg

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/authenticvision/tmppg/internal/logutil"
)

// WithPostgresql runs fn after spawning a temporary Postgresql instance
func WithPostgresql(fn func(socketDir string) error, opts ...Option) error {
	dir, err := os.MkdirTemp("", "tmppg")
	if err != nil {
		return fmt.Errorf("setup temporary directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			slog.Error("failed to remove temporary directory", logutil.Err(err))
		}
	}()
	opts = append(opts, WithoutSync())
	return WithPostgresqlDir(dir, fn, opts...)
}

// WithPostgresqlDir runs fn after spawning a Postgresql instance using dir as
// its data and socket directory. The cluster will be created in dir if it
// doesn't exist or is empty
func WithPostgresqlDir(dir string, fn func(socketDir string) error, opts ...Option) error {
	pg := NewPostgres(opts...)
	c, err := OpenOrCreateCluster(pg, dir)
	if err != nil {
		return fmt.Errorf("open or create cluster: %w", err)
	}

	if err := c.Start(); err != nil {
		return fmt.Errorf("start cluster: %w", err)
	}

	// run database removal deferred, so the database also gets removed on
	// runtime.Goexit() and t.FailNow()
	defer func() {
		if err := c.Stop(); err != nil {
			slog.Error("postgres failed", logutil.Err(err))
		}
	}()

	return fn(c.dir)
}

// RunWithPostgresql runs the given command with a temporary PostgreSQL instance available.
// Connection information is available via the standard PG* environment variables.
// See https://www.postgresql.org/docs/current/libpq-envars.html
func RunWithPostgresql(args []string, opts ...Option) error {
	return WithPostgresql(runCmd(args), opts...)
}

// RunWithPostgresqlDir runs the given command with a PostgreSQL instance available.
// Data is persisted in the given directory. A cluster is initialized if it doesn't exist.
// Connection information is available via the standard PG* environment variables.
// See https://www.postgresql.org/docs/current/libpq-envars.html
func RunWithPostgresqlDir(dir string, args []string, opts ...Option) error {
	return WithPostgresqlDir(dir, runCmd(args), opts...)
}

func runCmd(args []string) func(socketDir string) error {
	return func(socketDir string) error {
		wrapped := exec.Command(args[0], args[1:]...)
		wrapped.Stdin = os.Stdin
		wrapped.Stdout = os.Stdout
		wrapped.Stderr = os.Stderr
		wrapped.Env = append(os.Environ(), "PGHOST="+socketDir, "PGDATABASE=postgres")
		if err := wrapped.Run(); err != nil {
			return fmt.Errorf("%v: %v", wrapped.Args, err)
		}
		return nil
	}
}
