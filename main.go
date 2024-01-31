package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/simonswine/grafana-agent-cnc/app"
)

func init() {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})
	logger := slog.New(h)
	slog.SetDefault(logger)
}

func main() {
	ctx := context.Background()
	a := app.New()
	a.Run(ctx, os.Args...)
}
