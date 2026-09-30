// Command desktop is the EduPilot standalone desktop host (Mode A).
//
// It runs the React frontend inside a native Windows window using WebView2, with
// the Go application services compiled into the same executable and a SQLite
// database stored separately under the EduPilot data root.
//
// Wails is a delivery adapter only. Application services, domain and repository
// ports are shared verbatim with cmd/server and the future cloud API.
package main

import (
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/elcomware/edupilot/internal/app"
)

// assets holds the production frontend build. `wails3 build` generates
// cmd/desktop/frontend/dist before this package is compiled.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	root, err := dataRoot()
	if err != nil {
		logger.Error("the EduPilot data root could not be resolved", "error", err)
		os.Exit(1)
	}

	startupContext, cancelStartup := app.StartupContext()
	defer cancelStartup()

	// Migrations run before the window opens, so the interface never renders
	// against a schema that does not exist yet.
	edupilot, err := app.New(startupContext, app.DefaultSettings(root), logger)
	if err != nil {
		logger.Error("EduPilot could not start", "error", err)
		os.Exit(1)
	}

	service := newApp(edupilot)

	wails := application.New(application.Options{
		Name:        "EduPilot",
		Description: "EduPilot — modular school operating system",
		Logger:      logger,
		LogLevel:    slog.LevelInfo,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.edupilot.desktop",
		},
		Services: []application.Service{
			// The Wails adapter exposes exactly the operations the HTTP
			// adapter exposes, because both call the same application services.
			application.NewService(service),
		},
		OnShutdown: func() {
			// Pending outbox events are flushed by the dispatcher; the
			// database is closed last so a final write still lands.
			if err := edupilot.Close(); err != nil {
				logger.Error("the database could not be closed cleanly", "error", err)
			}
		},
		PanicHandler: func(details *application.PanicDetails) {
			logger.Error("panic", "error", details.Error)
		},
	})

	wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "EduPilot",
		URL:              "/",
		Width:            1440,
		Height:           900,
		BackgroundColour: application.RGBA{Red: 255, Green: 255, Blue: 255, Alpha: 255},
	})

	if err := wails.Run(); err != nil {
		logger.Error("edupilot exited with an error", "error", err)
		_ = edupilot.Close()
		os.Exit(1)
	}
}

// dataRoot resolves the production data root, kept separate from the
// application files: <root>/database/edupilot.db, <root>/files,
// <root>/backups, <root>/logs, <root>/runtime.
func dataRoot() (string, error) {
	if root := os.Getenv("EDUPILOT_DATA_DIR"); root != "" {
		return root, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(base, "EduPilot"), nil
}
