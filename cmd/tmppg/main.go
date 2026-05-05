package main

import (
	"log/slog"
	"os"

	"github.com/authenticvision/tmppg"
	"github.com/authenticvision/tmppg/internal/logutil"
)

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	opts := []tmppg.PostgresOption{tmppg.WithOutput(os.Stdout, os.Stderr)}
	if os.Args[1] == "--" {
		// tmppg -- cmd...
		args := os.Args[2:]
		if err := tmppg.RunWithPostgresql(args, opts...); err != nil {
			slog.Error("uncaught error", logutil.Err(err))
			os.Exit(1)
		}
	} else if os.Args[2] == "--" {
		// tmppg PGDATA -- cmd...
		dir := os.Args[1]
		args := os.Args[3:]
		if err := tmppg.RunWithPostgresqlDir(dir, args, opts...); err != nil {
			slog.Error("uncaught error", logutil.Err(err))
			os.Exit(1)
		}
	} else {
		usage()
	}
}

func usage() {
	slog.Error("usage: tmppg [PGDATA] -- command [args...]")
	os.Exit(1)
}
