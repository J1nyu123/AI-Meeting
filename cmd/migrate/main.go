package main

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		slog.Error("MYSQL_DSN is required")
		os.Exit(1)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		slog.Error("open mysql", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		slog.Error("create migration driver", "error", err)
		os.Exit(1)
	}
	m, err := migrate.NewWithDatabaseInstance("file:///app/migrations", "mysql", driver)
	if err != nil {
		slog.Error("load migrations", "error", err)
		os.Exit(1)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error("apply migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations_applied")
}
