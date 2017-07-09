package kktoken

import (
	"time"

	"github.com/jackc/pgx"
)

const (
	insert = "INSERT INTO token(token,id,create_at,last_use) VALUES($1,$2,$3,$4)"
)

var dbPool *pgx.ConnPool
var dbLiveSeconds uint32

// DBInfo information for the database
type DBInfo struct {
	DBPool           *pgx.ConnPool
	PersistentSecond uint32
	// DBTableName default: token
	DBTableName string
}

// prepareDB to prepare the database.
func prepareDB() error {
	s := `CREATE TABLE IF NOT EXISTS token (
	token UUID PRIMARY KEY,
	id INTEGER NOT NULL,
    create_at INTEGER NOT NULL,
	last_use INTEGER NOT NULL);`

	_, err := dbPool.Exec(s)
	return err
}

// setToken to set token.
func setToken(token string, userid int32, createAt int32) error {
	_, err := dbPool.Exec(insert, token, userid, createAt, createAt)
	return err
}

// getUserID to get userid from token.
func getUserID(token string) (int32, bool, error) {
	var setAt int32
	var userid int32

	err := dbPool.QueryRow("SELECT userid, set_at FROM token WHERE token=$1", token).Scan(&userid, &setAt)
	if err == pgx.ErrNoRows {
		return 0, false, nil
	}

	if err != nil {
		return 0, false, err
	}

	now := time.Now().Unix()
	// If expire before now, delete the record.
	if uint32(setAt)+dbLiveSeconds < uint32(now) {
		return 0, false, delToken(token)
	}

	return userid, true, nil
}

// delToken to delete a certain token.
func delToken(token string) error {
	_, err := dbPool.Exec("DELETE FROM token WHERE token=$1", token)
	return err
}
