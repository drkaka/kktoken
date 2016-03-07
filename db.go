package kktoken

import (
	"github.com/jackc/pgx"
)

var dbPool *pgx.ConnPool

// prepareDB to prepare the database.
func prepareDB() error {
	s := `CREATE TABLE IF NOT EXISTS token (
	token uuid primary key,
	userid integer,
    expire integer);`

	_, err := dbPool.Exec(s)
	return err
}
