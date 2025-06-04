package main

import (
	"github.com/authenticvision/tmppg"
	"github.com/authenticvision/tmppg/util/logutil"
	"log/slog"
	"os"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "--" {
		slog.Error("usage: tmppg -- command [args...]")
		os.Exit(1)
	}
	args := os.Args[2:]
	if err := tmppg.RunWithPostgresql(args); err != nil {
		slog.Error("uncaught error", logutil.Err(err))
		os.Exit(1)
	}
}
