package tmppg

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunWithPostgresql(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := require.New(t)
	err := RunWithPostgresql(
		[]string{"psql", "-d", "postgres", "-c", "SELECT 1"},
		WithLogOutput(log, slog.LevelInfo),
	)
	r.NoError(err)
}
