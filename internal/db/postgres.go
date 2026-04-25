package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var Pool *pgxpool.Pool

func Connect(url string) error {
	var err error
	Pool, err = pgxpool.New(context.Background(), url)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %v", err)
	}

	return Pool.Ping(context.Background())
}

func RunMigrations(url string, migrationsPath string) error {
	m, err := migrate.New(
		"file://"+migrationsPath,
		url,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %v", err)
	}
	return nil
}
