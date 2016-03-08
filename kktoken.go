package kktoken

import (
	"errors"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/jackc/pgx"
	"github.com/satori/go.uuid"
)

// CacheErr means db is set, while error pop when setting to cache.
var CacheErr error

func init() {
	CacheErr = errors.New("Cache not set.")
}

// Use this to set the pools.
// dbLive and cacheLive are the available seconds for in db and in redis.
func Use(poolDB *pgx.ConnPool, poolRDS *redis.Pool, dbLive, rdsLive uint32) error {
	if dbLive == 0 || rdsLive == 0 {
		return errors.New("Live seconds must larger than 0.")
	}

	dbPool = poolDB
	rdsPool = poolRDS

	dbLiveSeconds = dbLive
	rdsLiveSeconds = rdsLive

	return prepareDB()
}

// MakeToken to make and set token to db and cache.
// If error == kktoken.CacheErr, it means db is set, but cache not.
func MakeToken(userid int32) (string, error) {
	tk := uuid.NewV4().String()
	now := time.Now().Unix()

	if err := setToken(tk, userid, uint32(now)+dbLiveSeconds); err != nil {
		return "", err
	}

	if err := setCache(tk, userid); err != nil {
		return tk, CacheErr
	}

	return tk, nil
}

// GetUserID to get userid from token.
// return userid, got, error
// the error can be kktoken.CacheErr.
func GetUserID(token string) (int32, bool, error) {
	var userid int32
	var err error
	var ok bool

	// get user id from cache
	if userid, ok, err = getCache(token); err != nil {
		return userid, ok, err
	} else if ok {
		return userid, ok, nil
	}

	// if not get, get from db
	if userid, ok, err = getUserID(token); err != nil {
		return userid, ok, err
	} else if !ok {
		return userid, false, nil
	}

	// if in db, set to cache
	if err = setCache(token, userid); err != nil {
		return userid, true, err
	}
	return userid, true, nil
}

// DelToken to delete the token.
func DelToken(token string) error {
	err1 := delCache(token)
	err2 := delToken(token)
	if err1 != nil {
		return err1
	}
	return err2
}
