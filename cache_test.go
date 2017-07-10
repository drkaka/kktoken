package kktoken

import (
	"testing"

	"github.com/garyburd/redigo/redis"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func testCacheMethods(t *testing.T) {
	// test empty get.
	userid, err := getRedisCache("abcdefg")
	assert.NoError(t, err, "should not have error to get non-existed cache")
	assert.Equal(t, int32(0), userid, "userid wrong")

	tk1 := uuid.NewV4().String()
	tk2 := uuid.NewV4().String()
	userid = int32(3)

	// set cache
	err = setRedisCache([]string{tk1, tk2}, []int32{userid})
	assert.Error(t, err, "should have error when array length not same")

	err = setRedisCache([]string{tk1, tk2}, []int32{userid, userid})
	assert.NoError(t, err, "should have no error to set cache")

	// get cache
	gotUserID, err := getRedisCache(tk1)
	assert.NoError(t, err, "should have no error to get from cache")
	assert.Equal(t, userid, gotUserID, "userid wrong")

	gotUserID, err = getRedisCache(tk2)
	assert.NoError(t, err, "should have no error to get from cache")
	assert.Equal(t, userid, gotUserID, "userid wrong")

	checkTTL(tk1, t)

	// delete cache
	err = delRedisCache(tk1)
	assert.NoError(t, err, "should not have error to delete cache")

	err = delRedisCache(tk2)
	assert.NoError(t, err, "should not have error to delete cache")

	// get cache should return 0
	gotUserID, err = getRedisCache(tk1)
	assert.NoError(t, err, "should have no error to get from cache")
	assert.Equal(t, int32(0), gotUserID, "userid wrong")
}

func checkTTL(tk string, t *testing.T) {
	conn := rdsPool.Get()
	defer conn.Close()

	ttl, err := redis.Int(conn.Do("TTL", tk))
	assert.NoError(t, err, "should not have error to get cache TTL")
	assert.Equal(t, rdsLiveSecond, uint32(ttl), "TTL wrong")
}
