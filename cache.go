package kktoken

import (
	"errors"

	"github.com/garyburd/redigo/redis"
)

// RDSInfo containing Redis information
type RDSInfo struct {
	Pool *redis.Pool
	// The seconds to live in redis, default: 300
	LiveSecond uint32
}

var (
	rdsPool       *redis.Pool
	rdsLiveSecond = uint32(300)
)

func prepareRedis(rdsInfo *RDSInfo) error {
	if rdsInfo.Pool == nil {
		return errors.New("rdsInfo Pool Can't be nil")
	}

	if rdsInfo.LiveSecond > 0 {
		// set the value if not 0
		rdsLiveSecond = rdsInfo.LiveSecond
	}

	// PING to check redis server
	conn := rdsInfo.Pool.Get()
	defer conn.Close()
	if pong, err := redis.String(conn.Do("PING")); err != nil {
		return err
	} else if pong != "PONG" {
		return errors.New("redis ping wrong")
	}
	rdsPool = rdsInfo.Pool

	return nil
}

// setCache to set cache tokens for users.
func setRedisCache(tokens []string, userids []int32) error {
	conn := rdsPool.Get()
	defer conn.Close()

	l := len(tokens)
	if l != len(userids) || l == 0 {
		return errors.New("parameters wrong for redis batch set")
	}

	conn.Send("MULTI")
	for i := 0; i < l; i++ {
		conn.Send("SETEX", tokens[i], rdsLiveSecond, userids[i])
	}
	_, err := conn.Do("EXEC")
	return err
}

// getRedisCache to get a cache from redis.
// Return userid (0 means not found), error
func getRedisCache(token string) (int32, error) {
	conn := rdsPool.Get()
	defer conn.Close()

	userid, err := redis.Int(conn.Do("GET", token))
	if err == redis.ErrNil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return int32(userid), nil
}

// delCache to delete a cache.
func delRedisCache(token string) error {
	conn := rdsPool.Get()
	defer conn.Close()

	if _, err := conn.Do("DEL", token); err != nil && err != redis.ErrNil {
		return err
	}
	return nil
}
