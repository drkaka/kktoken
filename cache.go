package kktoken

import "github.com/garyburd/redigo/redis"

var rdsPool *redis.Pool
var rdsLiveSeconds uint32

// setCache to set a cache token for user.
func setCache(token string, userid int32) error {
	conn := rdsPool.Get()
	defer conn.Close()

	if _, err := conn.Do("SETEX", token, rdsLiveSeconds, userid); err != nil {
		return err
	}
	return nil
}

// getCache to get a cache.
// Return userid, got or not, error
func getCache(token string) (int32, bool, error) {
	conn := rdsPool.Get()
	defer conn.Close()

	userid, err := redis.Int(conn.Do("GET", token))
	if err == redis.ErrNil {
		return 0, false, nil
	}

	if err != nil {
		return 0, false, err
	}

	return int32(userid), true, nil
}

// delCache to delete a cache.
func delCache(token string) error {
	conn := rdsPool.Get()
	defer conn.Close()

	if _, err := conn.Do("DEL", token); err != nil && err != redis.ErrNil {
		return err
	}
	return nil
}
