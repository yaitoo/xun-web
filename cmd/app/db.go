package main

import (
	"context"
	"embed"
	"fmt"

	"github.com/cnlangzi/sqlite"
	"github.com/spf13/viper"
	"github.com/yaitoo/sqle"
	"github.com/yaitoo/sqle/migrate"
)

//go:embed migrations
var migrations embed.FS

// Global database handle
var db *sqlite.DB

func setupSQLite(ctx context.Context) (*sqlite.DB, error) {
	dsn := viper.GetString("db.dsn")
	if dsn == "" {
		dsn = "./app.db"
	}

	// Assign to the package-level handle so migrateSQLite (which closes
	// over `db`) sees the same instance. Using `=` rather than `:=` here
	// is intentional — `:=` would create a local var that shadows the
	// global and leave migrateSQLite reading a nil pointer.
	var err error
	db, err = sqlite.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create db: %w", err)
	}

	if err := migrateSQLite(ctx); err != nil {
		return nil, fmt.Errorf("failed to migrate db: %w", err)
	}

	return db, nil
}

func migrateSQLite(ctx context.Context) error {
	d := sqle.Open(db.Writer.DB)
	migrator := migrate.New(d)
	if err := migrator.Discover(migrations); err != nil {
		return fmt.Errorf("failed to discover migrations: %w", err)
	}

	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to init migrations: %w", err)
	}

	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to migrate db: %w", err)
	}

	return nil
}
