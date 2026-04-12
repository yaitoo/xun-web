package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"strings"

	"github.com/yaitoo/sqle/migrate"
	"github.com/yaitoo/xun"
	"github.com/yaitoo/xun/ext/htmx"
)

//go:embed app
var fsys embed.FS

var (
	flagConfig  = flag.String("config", "app.yml", "config file path")
	flagMigrate = flag.Bool("migrate", false, "run migrations and exit")
	flagAddr    = flag.String("addr", ":8080", "server address")
)

func main() {
	flag.Parse()

	app := xun.New(
		xun.WithFsys(fsys),
		xun.WithWatch(),
		xun.WithHandlerViewers(&xun.JsonViewer{}),
		xun.WithInterceptor(htmx.New()),
		xun.WithBuildAssetURL(func(path string) bool {
			return strings.HasPrefix(path, "/assets/")
		}),
	)

	var err error
	db, err = initDB()
	if err != nil {
		slog.Error("failed to init db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations if requested
	if *flagMigrate {
		if err := runMigrations(); err != nil {
			slog.Error("failed to run migrations", "err", err)
			os.Exit(1)
		}
		slog.Info("migrations completed")
		return
	}

	// Apply middleware
	app.Use(sessionMiddleware)

	// Setup routes
	setupRoutes(app)

	app.Start()

	addr := *flagAddr
	if addr == "" {
		addr = ":8080"
	}

	slog.Info("starting server", "addr", addr)
	// Using nil mux means http.DefaultServeMux
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("server error", "err", err)
	}
}

func runMigrations() error {
	migrator := migrate.New(db)

	if err := migrator.Discover(fsys, migrate.WithModule("github.com/yaitoo/xun-web")); err != nil {
		return fmt.Errorf("failed to discover migrations: %w", err)
	}

	ctx := context.Background()
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to init migrations: %w", err)
	}

	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to migrate: %w", err)
	}

	return nil
}
