package kktoken

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/jackc/pgx"
)

func TestMain(t *testing.T) {
	testInvalidUseParameters(t)

	setPools(t)
	defer dbPool.Close()
	defer rdsPool.Close()

	testTableGeneration(t)

	testCacheMethods(t)
	testDBMethods(t)

	testPublicMethods(t)
}

func testInvalidUseParameters(t *testing.T) {
	if err := Use(nil, nil, 0, 1); err == nil {
		t.Error("Should have error.")
	}

	if err := Use(nil, nil, 1, 0); err == nil {
		t.Error("Should have error.")
	}
}

func testPublicMethods(t *testing.T) {
	userid := int32(3)

	tk, err := MakeToken(userid)
	if err != nil {
		t.Error(err)
	}

	if uid, ok, err := getUserID(tk); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("Should get userid from db.")
	} else if uid != userid {
		t.Error("Userid is wrong.")
	}

	if uid, ok, err := getCache(tk); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("Should get userid from cache.")
	} else if uid != userid {
		t.Error("Userid is wrong.")
	}

	// delete the cache
	if err := delCache(tk); err != nil {
		t.Error(err)
	}

	// should get result and set cache
	if uid, ok, err := GetUserID(tk); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("Should get userid.")
	} else if uid != userid {
		t.Error("Userid is wrong.")
	}

	// should have cache now
	if uid, ok, err := getCache(tk); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("Should get userid from cache.")
	} else if uid != userid {
		t.Error("Userid is wrong.")
	}

	if err := DelToken(tk); err != nil {
		t.Error(err)
	}
}

func setPools(t *testing.T) {
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

	var err error
	var poolDB *pgx.ConnPool
	if poolDB, err = pgx.NewConnPool(connPoolConfig); err != nil {
		t.Fatal(err)
	}

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

	if err := Use(poolDB, poolRDS, uint32(300), uint32(30000000)); err != nil {
		t.Error(err)
	}
}
