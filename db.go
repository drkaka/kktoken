package kktoken

import (
	"time"

	"github.com/jackc/pgx"
)

const (
	insert = "INSERT INTO token(token,userid,expire) VALUES($1,$2,$3)"
)

var dbPool *pgx.ConnPool
var dbLiveSeconds uint32

// prepareDB to prepare the database.
func prepareDB() error {
	s := `CREATE TABLE IF NOT EXISTS token (
	token uuid primary key,
	userid integer,
    expire integer);`

	_, err := dbPool.Exec(s)
	return err
}

// setToken to set token.
func setToken(token string, userid int32, exp uint32) error {
	_, err := dbPool.Exec(insert, token, userid, exp)
	return err
}

// getUserID to get userid from token.
func getUserID(token string) (int32, bool, error) {
	var exp int32
	var userid int32

	err := dbPool.QueryRow("SELECT userid, expire FROM token WHERE token=$1", token).Scan(&userid, &exp)
	if err == pgx.ErrNoRows {
		return 0, false, nil
	}

	if err != nil {
		return 0, false, err
	}

	now := time.Now().Unix()
	// If expire before now, delete the record.
	if exp < int32(now) {
		return 0, false, delToken(token)
	}

	return userid, true, nil
}

// delToken to delete a certain token.
func delToken(token string) error {
	_, err := dbPool.Exec("DELETE FROM token WHERE token=$1", token)
	return err
}
