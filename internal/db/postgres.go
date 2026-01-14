package db

import (
	"context"
	"github.com/jackc/pgx/v5"
)

var Conn *pgx.Conn
var ctx = context.Background()

func Connect(url string) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	Conn = conn
	return nil
}
