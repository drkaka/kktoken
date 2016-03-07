package kktoken

import (
	"testing"

	"github.com/jackc/pgx"
)

func testTableGeneration(t *testing.T) {
	var dbname pgx.NullString
	if err := dbPool.QueryRow("SELECT 'public.token'::regclass;").Scan(&dbname); err != nil {
		t.Fatal(err)
	}

	if dbname.String != "token" {
		t.Fatal("dbname is not correct.")
	}
}
