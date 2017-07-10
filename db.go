package kktoken

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx"
)

var (
	dbPool             *pgx.ConnPool
	dbPersistentSecond uint32

	insertTokenStm      string
	updateLastUseStm    string
	deleteTokenStm      string
	getUserIDStm        string
	getUserIDWithEXPStm string
	queryTokenStm       string
)

// DBInfo information for the database
type DBInfo struct {
	Pool *pgx.ConnPool
	// PersistentSecond to delete the record after how many seconds from last_use, 0 means never expire
	PersistentSecond uint32
	// DBTableName default: token
	TableName string
	// DBEXPCheckSecond the frequency to check expiration, default: 300
	EXPCheckSecond uint32
}

// TokenInfo of a single token
type TokenInfo struct {
	// UUID token
	Token string
	// the attached information
	Info     map[string]interface{}
	UserID   int32
	CreateAt int32
	LastUse  int32
}

// prepareDB to prepare the database.
func prepareDB(info *DBInfo) error {
	if info.Pool == nil {
		return errors.New("dbInfo Pool can't be nil")
	}
	// setup the info
	dbPool = info.Pool
	dbPersistentSecond = info.PersistentSecond

	tableName := info.TableName
	if tableName == "" {
		tableName = "token"
	}

	// create db if not exist
	s := `CREATE TABLE IF NOT EXISTS %s (
	token UUID PRIMARY KEY,
	user_id INTEGER NOT NULL,
	info JSONB,
    create_at INTEGER NOT NULL,
	last_use INTEGER NOT NULL);`
	if _, err := dbPool.Exec(fmt.Sprintf(s, tableName)); err != nil {
		return err
	}

	// create index if not exist for user_id
	s = "CREATE INDEX IF NOT EXISTS %s_user_id_index ON %s USING btree (user_id);"
	if _, err := dbPool.Exec(fmt.Sprintf(s, tableName, tableName)); err != nil {
		return err
	}

	// create index if not exist for last_use
	s = "CREATE INDEX IF NOT EXISTS %s_last_use_index ON %s USING btree (last_use);"
	if _, err := dbPool.Exec(fmt.Sprintf(s, tableName, tableName)); err != nil {
		return err
	}

	// if needs to expire, start checker in a goroutine
	if dbPersistentSecond > 0 {
		if info.EXPCheckSecond == 0 {
			info.EXPCheckSecond = 300
		}
		go startDBEXPCheck(info.EXPCheckSecond, tableName)
	}

	// create SQL statements
	insertTokenStm = fmt.Sprintf("INSERT INTO %s(token,user_id,info,create_at,last_use) VALUES($1,$2,$3,$4,$5)", tableName)
	updateLastUseStm = fmt.Sprintf("UPDATE %s SET last_use=$1 WHERE token=$2", tableName)
	deleteTokenStm = fmt.Sprintf("DELETE FROM %s WHERE token=$1", tableName)
	getUserIDStm = fmt.Sprintf("SELECT user_id FROM %s WHERE token=$1", tableName)
	getUserIDWithEXPStm = fmt.Sprintf("SELECT user_id FROM %s WHERE token=$1 AND last_use>$2", tableName)
	queryTokenStm = fmt.Sprintf("SELECT token,info,create_at,last_use FROM %s WHERE user_id=$1", tableName)

	return nil
}

// startDBEXPCheck to delete all records that expired running every given seconds.
func startDBEXPCheck(seconds uint32, tableName string) {
	delExpStm := fmt.Sprintf("DELETE FROM %s WHERE last_use < $1", tableName)
	c := time.Tick(time.Duration(seconds) * time.Second)
	for now := range c {
		if _, err := dbPool.Exec(delExpStm, now.Unix()-int64(dbPersistentSecond)); err != nil {
			// if there is an error, go to chan
			errChan <- err
		}
	}
}

// setToken to set token.
func setToken(info *TokenInfo) error {
	_, err := dbPool.Exec(insertTokenStm, info.Token, info.UserID, info.Info, info.CreateAt, info.LastUse)
	return err
}

func updateToken(token string, lastUse int32) error {
	_, err := dbPool.Exec(updateLastUseStm, lastUse, token)
	// no rows found in DB, maybe requested from cache, so this shouldn't be an error
	if err == pgx.ErrNoRows {
		return nil
	}
	return err
}

// getUserID to get userid from token.
// if userid == 0, meaning not found
func getUserID(token string) (int32, error) {
	var userid int32
	var err error
	now := time.Now().Unix()

	if dbPersistentSecond == 0 {
		// get userid without checking the expiration
		err = dbPool.QueryRow(getUserIDStm, token).Scan(&userid)
	} else {
		// only get the non-expired token
		err = dbPool.QueryRow(getUserIDWithEXPStm, token, now-int64(dbPersistentSecond)).Scan(&userid)
	}

	// nothing found
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	// not a valid UUID
	if err, ok := err.(pgx.PgError); ok && err.Code == "22P02" {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return userid, nil
}

func getAllTokens(userid int32) ([]TokenInfo, error) {
	var tokens []TokenInfo
	rows, _ := dbPool.Query(queryTokenStm, userid)
	if err := rows.Err(); err != nil {
		return tokens, err
	}

	// get all token information of a user
	for rows.Next() {
		var one TokenInfo
		if err := rows.Scan(&one.Token, &one.Info, &one.CreateAt, &one.LastUse); err != nil {
			return tokens, err
		}
		// remove "-" and lower case
		one.Token = strings.ToLower(strings.Replace(one.Token, "-", "", -1))
		one.UserID = userid
		tokens = append(tokens, one)
	}
	return tokens, nil
}

// delToken to delete a certain token.
func delToken(token string) error {
	_, err := dbPool.Exec(deleteTokenStm, token)
	return err
}
