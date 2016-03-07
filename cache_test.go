package kktoken

import (
	"testing"

	"github.com/garyburd/redigo/redis"
	"github.com/satori/go.uuid"
)

func testCacheMethods(t *testing.T) {
	// test empty get.
	if _, ok, err := getCache("abc"); err != nil {
		t.Error(t)
	} else if ok {
		t.Error("Should have not got.")
	}

	tk := uuid.NewV4().String()
	userid := int32(3)

	// set cache
	if err := setCache(tk, userid); err != nil {
		t.Error(err)
	}

	// get cache
	if uid, ok, err := getCache(tk); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("Failed to get cache.")
	} else if uid != userid {
		t.Error("userid is wrong.")
	}

	// check TTL.
	conn := rdsPool.Get()
	defer conn.Close()

	if ttl, err := redis.Int(conn.Do("TTL", tk)); err != nil {
		t.Error(err)
	} else if uint32(ttl) != rdsLiveSeconds {
		t.Error("ttl is wrong.")
	}

	// delete cache
	if err := delCache(tk); err != nil {
		t.Error(err)
	}
}
