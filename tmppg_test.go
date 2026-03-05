package tmppg

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunWithPostgresql(t *testing.T) {
	logOutput := &bytes.Buffer{}
	log := slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := require.New(t)
	err := RunWithPostgresql(
		[]string{"psql", "-Atc", "SELECT 'ok'"},
		WithLogOutput(log, slog.LevelInfo),
		WithLogStatement("all"),
	)
	r.NoError(err)
	r.Regexp(`level=INFO .* statement: SELECT 'ok'`, logOutput.String())
}
