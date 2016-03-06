package kktoken

import (
	"github.com/garyburd/redigo/redis"
	"github.com/jackc/pgx"
)

// Use this to set the pools.
func Use(poolDB *pgx.ConnPool, poolRDS *redis.Pool) error {
	dbPool = poolDB
	rdsPool = poolRDS
	return nil
}
