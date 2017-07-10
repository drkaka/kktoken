package kktoken

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/jackc/pgx"
	"github.com/stretchr/testify/assert"
)

const (
	testTableName = "token_test"
)

func TestMain(t *testing.T) {
	testInvalidUseParameters(t)

	_, err := Use(getDBInfo(t), getRDSInfo(t), nil)
	assert.NoError(t, err, "should not have when testing USE")

	testTableGeneration(testTableName, t)

	testGetAndSetMap(t)
	testPublicMethods(t)
	testGetFromCache(t)
	testGetFromDB(t)
	testMapEXPCheck(t)

	testCacheMethods(t)
	testDBMethods(t)

	if dbPool != nil {
		_, err := dbPool.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", testTableName))
		assert.NoError(t, err, "Should not have error when drop table.")
	}
}

func getDBInfo(t *testing.T) *DBInfo {
	DBName := os.Getenv("dbname")
	DBHost := os.Getenv("dbhost")
	DBUser := os.Getenv("dbuser")
	DBPassword := os.Getenv("dbpassword")

	connPoolConfig := pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     DBHost,
			User:     DBUser,
			Password: DBPassword,
			Database: DBName,
			Dial:     (&net.Dialer{KeepAlive: 1 * time.Minute, Timeout: 10 * time.Second}).Dial,
		},
		MaxConnections: 10,
	}

	var poolDB *pgx.ConnPool
	var err error
	if poolDB, err = pgx.NewConnPool(connPoolConfig); err != nil {
		t.Fatal(err)
	}
	return &DBInfo{
		Pool:      poolDB,
		TableName: testTableName,
	}
}

func getRDSInfo(t *testing.T) *RDSInfo {
	RDSHost := os.Getenv("rdshost")

	opt1 := redis.DialConnectTimeout(5 * time.Second)
	opt2 := redis.DialReadTimeout(5 * time.Second)
	opt3 := redis.DialWriteTimeout(5 * time.Second)
	poolRDS := &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 60 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", RDSHost, opt1, opt2, opt3)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return &RDSInfo{
		Pool: poolRDS,
	}
}

func testInvalidUseParameters(t *testing.T) {
	errChan, err := Use(nil, nil, nil)
	assert.Error(t, err, "Error should happen when pasing nil")
	assert.Nil(t, errChan, "err channel should be nil")

	info := getDBInfo(t)
	errChan, err = Use(info, nil, nil)
	assert.Error(t, err, "Error should happen when pasing nil")
	assert.Nil(t, errChan, "err channel should be nil")
}

func testGetAndSetMap(t *testing.T) {
	tk := "abc"
	userid := int32(2)
	now := int32(time.Now().Unix())
	setToMap(tk, userid)

	time.Sleep(1 * time.Second)

	// get userid from Map
	gotUserID := getAndSetMap(tk)
	assert.Equal(t, userid, gotUserID, "got user id wrong")

	// get a non-existed userid
	gotUserID = getAndSetMap("aaa")
	assert.Equal(t, int32(0), gotUserID, "got user id wrong")

	// check last_use
	allTokens.lock.Lock()
	info, ok := allTokens.all[tk]
	assert.True(t, ok, "should be true to find in Map")
	assert.Equal(t, now+1, info.lastUse, "last_use wrong")
	delete(allTokens.all, "tk")
	allTokens.lock.Unlock()
}

func testPublicMethods(t *testing.T) {
	userid := int32(4)
	info := map[string]interface{}{
		"device": "ios",
	}
	tk, err := MakeToken(userid, info)
	assert.NoError(t, err, "should not have error to make token")

	// should be able to find in Map
	gotUserID := getAndSetMap(tk)
	assert.Equal(t, userid, gotUserID, "should be able to find in Map")

	// should be able to find in Cache
	gotUserID, err = getRedisCache(tk)
	assert.NoError(t, err, "should not have error to get from cache")
	assert.Equal(t, userid, gotUserID, "should be able to find in Cache")

	// should be able to find in DB
	gotUserID, err = getUserID(tk)
	assert.NoError(t, err, "should not have error to get from DB")
	assert.Equal(t, userid, gotUserID, "should be able to find in DB")

	// Public get method
	gotUserID, err = GetUserID(tk)
	assert.NoError(t, err, "should not have error to get with public method")
	assert.Equal(t, userid, gotUserID, "should be able to find with public method")

	// Public get all tokens
	tokens, err := GetUserTokens(userid)
	assert.NoError(t, err, "should not have error to get all tokens")
	assert.Len(t, tokens, 1, "should find 1 token")
	assert.Equal(t, "ios", tokens[0].Info["device"], "info is wrong")

	// public delete method
	err = DelToken(tk)
	assert.NoError(t, err, "should not have error to delete with public method")

	// should be able to find in Map
	gotUserID = getAndSetMap(tk)
	assert.Equal(t, int32(0), gotUserID, "should not be able to find in Map")

	// should be able to find in Cache
	gotUserID, err = getRedisCache(tk)
	assert.NoError(t, err, "should not have error to get from cache")
	assert.Equal(t, int32(0), gotUserID, "should not be able to find in Cache")

	// should be able to find in DB
	gotUserID, err = getUserID(tk)
	assert.NoError(t, err, "should not have error to get from DB")
	assert.Equal(t, int32(0), gotUserID, "should not be able to find in DB")
}

func testGetFromCache(t *testing.T) {
	userid := int32(7)
	info := map[string]interface{}{
		"device": "ios",
	}
	tk, err := MakeToken(userid, info)
	assert.NoError(t, err, "should not have error to make token")

	// delete from Map
	allTokens.lock.Lock()
	delete(allTokens.all, tk)
	allTokens.lock.Unlock()

	// delete from DB
	err = delToken(tk)
	assert.NoError(t, err, "should not have error to delete from DB")

	// get
	gotUserID, err := GetUserID(tk)
	assert.NoError(t, err, "should not have error to get with public method")
	assert.Equal(t, userid, gotUserID, "should be able to find with public method")

	// Map should be set
	gotUserID = getAndSetMap(tk)
	assert.Equal(t, userid, gotUserID, "should be able to find with Map")

	// delete
	err = DelToken(tk)
	assert.NoError(t, err, "should not have error to delete with public method")
}

func testGetFromDB(t *testing.T) {
	userid := int32(8)
	info := map[string]interface{}{
		"device": "ios",
	}
	tk, err := MakeToken(userid, info)
	assert.NoError(t, err, "should not have error to make token")

	// delete from Map
	allTokens.lock.Lock()
	delete(allTokens.all, tk)
	allTokens.lock.Unlock()

	// delete from Redis
	err = delRedisCache(tk)
	assert.NoError(t, err, "should not have error to delete from Redis")

	// get
	gotUserID, err := GetUserID(tk)
	assert.NoError(t, err, "should not have error to get with public method")
	assert.Equal(t, userid, gotUserID, "should be able to find with public method")

	// Map should be set
	gotUserID = getAndSetMap(tk)
	assert.Equal(t, userid, gotUserID, "should be able to find with Map")

	// Redis should be set
	gotUserID, err = getRedisCache(tk)
	assert.NoError(t, err, "should not have error to get from Redis")
	assert.Equal(t, userid, gotUserID, "should be able to find in Redis")

	// delete
	err = DelToken(tk)
	assert.NoError(t, err, "should not have error to delete with public method")
}

func testMapEXPCheck(t *testing.T) {
	// generate a token
	mapLiveSecond = 1
	userid := int32(10)
	info := map[string]interface{}{
		"device": "ios",
	}

	tk, err := MakeToken(userid, info)
	assert.NoError(t, err, "should not have error to make token")

	// start exp check
	go startMapEXPCheck(2)
	time.Sleep(1520 * time.Millisecond)

	tk2, err := MakeToken(int32(11), info)
	assert.NoError(t, err, "should not have error to make token")

	time.Sleep(500 * time.Millisecond)

	// check remains in Map
	allTokens.lock.RLock()
	_, ok := allTokens.all[tk]
	assert.False(t, ok, "should not exist in Map")
	_, ok = allTokens.all[tk2]
	assert.True(t, ok, "should exist in Map")
	allTokens.lock.RUnlock()

	// check redis TTL, should not be updated
	conn := rdsPool.Get()
	defer conn.Close()

	ttl, err := redis.Int(conn.Do("TTL", tk))
	assert.NoError(t, err, "should not have error to get cache TTL")
	assert.Equal(t, rdsLiveSecond-2, uint32(ttl), "TTL wrong")
}
