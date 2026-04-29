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
	ErrUnableToConnect = errors.New("unable to connect to database")
)

type PgxConf struct {
	DB *pgxpool.Pool
}

func (p *PgxConf) GetDBConn() *pgxpool.Pool {
	return p.DB
}

func (p *PgxConf) SetDBConn(conn *pgxpool.Pool) {
	p.DB = conn
}

func InitPgsql(cnf *config.Config) (*PgxConf, error) {
	var pgxConf = &PgxConf{}
	if cnf.GetDatabaseURI() == "" {
		return pgxConf, ErrConnNotFound
	}
	conn, err := pgxpool.New(context.Background(), cnf.GetDatabaseURI())
	if err != nil {
		return pgxConf, ErrUnableToConnect
	}

	log.Println("Connected to database successfully")
	if err := runMigrations(cnf); err != nil {
		return pgxConf, err
	}

	pgxConf.SetDBConn(conn)
	return pgxConf, nil
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