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

// MakeToken to make token.
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
