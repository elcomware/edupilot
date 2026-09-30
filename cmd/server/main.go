// Command server is the EduPilot Site Server (Mode B).
//
// The site server owns the school database on the school LAN. Desktop clients
// call the same application services the desktop host uses, reached through an
// HTTP transport adapter instead of Wails.
//
// A SQLite database file is never opened directly by clients on a network
// drive: clients must talk to this process.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/elcomware/edupilot/internal/app"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address for the site server")
	dataDir := flag.String("data", "", "EduPilot data root (defaults to the platform data directory)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	root := *dataDir
	if root == "" {
		resolved, err := defaultDataRoot()
		if err != nil {
			logger.Error("cannot resolve the EduPilot data directory", "error", err)
			os.Exit(1)
		}
		root = resolved
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The site server owns the SQLite file: clients never open it from a
	// network drive. Migrations run before the listener starts, so no client
	// can reach a half-built schema.
	startupContext, cancelStartup := app.StartupContext()
	defer cancelStartup()

	edupilot, err := app.New(startupContext, app.DefaultSettings(root), logger)
	if err != nil {
		logger.Error("cannot open the school database", "error", err)
		os.Exit(1)
	}
	defer func() { _ = edupilot.Close() }()

	srv := &http.Server{
		Addr:              *addr,
		Handler:           newHTTPTransport(logger, edupilot),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("edupilot site server listening", "addr", *addr, "data", root)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("site server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}

	logger.Info("edupilot site server stopped")
}

// defaultDataRoot resolves the production data root, kept separate from the
// application files.
func defaultDataRoot() (string, error) {
	if root := os.Getenv("EDUPILOT_DATA_DIR"); root != "" {
		return root, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "EduPilot"), nil
}
