package db

import (
	"FoPQer/go-fermart/internal/config"
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrConnNotFound = errors.New("connection to database not found")
	errUnableToConnect = errors.New("unable to connect to database")
)

func InitPgsql(ctx context.Context, cnf *config.Config) (*pgxpool.Pool, error) {
	if cnf.GetDatabaseURI() == "" {
		return nil, ErrConnNotFound
	}
	conn, err := pgxpool.New(ctx, cnf.GetDatabaseURI())
	if err != nil {
		return nil, errUnableToConnect
	}

	slog.Info("Connected to database successfully")
	if err := runMigrations(cnf); err != nil {
		return nil, err
	}

	return conn, nil
}

func runMigrations(cnf *config.Config) error {
	m, err := migrate.New(
		"file://migrations",
		cnf.GetDatabaseURI(),
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}