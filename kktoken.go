package kktoken

import (
	"errors"

	"github.com/garyburd/redigo/redis"
	"github.com/jackc/pgx"
)

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
