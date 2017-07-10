package kktoken

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/satori/go.uuid"
)

// MapInfo the map cache information
type MapInfo struct {
	// How many seconds a token will live in map, default: 60
	LiveSecond uint32
	// How many seconds a check map will happen, default: 31
	// EXPCheck will delete expired tokens in map and update the last_use both in redis and DB.
	// Beware that this should smaller than RDSLiveSecond
	EXPCheckSecond uint32
}

// the latest usage information of token
type tokenLatest struct {
	userid  int32
	lastUse int32
}

// the token store
type tokenStore struct {
	all  map[string]*tokenLatest
	lock *sync.RWMutex
}

var (
	// ErrCache means db is set, while error pop when setting to cache.
	ErrCache = errors.New("cache not set")
	// used to get errors from background goroutine
	errChan = make(chan error)

	// this is not the exact seconds because only EXPCheck will check expiration
	mapLiveSecond = uint32(60)

	allTokens = tokenStore{
		all:  make(map[string]*tokenLatest),
		lock: new(sync.RWMutex),
	}
)

// Use this to set the pools.
// dbLive and cacheLive are the available seconds for in db and in redis.
func Use(dbInfo *DBInfo, rdsInfo *RDSInfo, mapInfo *MapInfo) (chan error, error) {
	if dbInfo == nil {
		return nil, errors.New("dbInfo can't be nil")
	}

	if rdsInfo == nil {
		return nil, errors.New("rdsInfo can't be nil")
	}

	if err := prepareDB(dbInfo); err != nil {
		return nil, err
	}

	if err := prepareRedis(rdsInfo); err != nil {
		return nil, err
	}

	mapEXPCheckSecond := uint32(31)
	if mapInfo != nil {
		if mapInfo.LiveSecond != 0 {
			mapLiveSecond = mapInfo.LiveSecond
		}
		if mapInfo.EXPCheckSecond != 0 {
			mapEXPCheckSecond = mapInfo.EXPCheckSecond
		}
	}

	// start the checker for tokens in map
	go startMapEXPCheck(mapEXPCheckSecond)
	return errChan, nil
}

func startMapEXPCheck(seconds uint32) {
	c := time.Tick(time.Duration(seconds) * time.Second)
	for now := range c {
		var delTokens []string
		var all []string
		var allLatest []int32
		var allIDs []int32

		// get exp threshost
		exp := now.Unix() - int64(mapLiveSecond)

		// get the expired tokens
		allTokens.lock.Lock()
		for k, v := range allTokens.all {
			if int64(v.lastUse) < exp {
				delTokens = append(delTokens, k)
			}
			all = append(all, k)
			allIDs = append(allIDs, v.userid)
			allLatest = append(allLatest, v.lastUse)
		}
		// delete expired tokens from map
		if len(delTokens) > 0 {
			for i := 0; i < len(delTokens); i++ {
				delete(allTokens.all, delTokens[i])
			}
		}
		allTokens.lock.Unlock()

		// update all tokens in map to redis
		if err := setRedisCache(all, allIDs); err != nil {
			errChan <- err
		}
		// update all tokens in map to DB
		for i := 0; i < len(all); i++ {
			err := updateToken(all[i], allLatest[i])
			errChan <- err
		}
	}
}

func getAndSetMap(tk string) int32 {
	allTokens.lock.Lock()
	defer allTokens.lock.Unlock()

	userid := int32(0)
	if info, ok := allTokens.all[tk]; ok {
		userid = info.userid
		info.lastUse = int32(time.Now().Unix())
	}
	return userid
}

func setToMap(tk string, userid int32) {
	allTokens.lock.Lock()
	allTokens.all[tk] = &tokenLatest{
		userid:  userid,
		lastUse: int32(time.Now().Unix()),
	}
	allTokens.lock.Unlock()
}

// MakeToken to make and set token to db, cache and map.
// If error == kktoken.CacheErr, it means db is set, but cache not.
func MakeToken(userid int32, info map[string]interface{}) (string, error) {
	if userid <= 0 {
		return "", errors.New("userid should no less than 0")
	}
	// Generate a UUID v4 token and remove "-"
	tk := strings.Replace(uuid.NewV4().String(), "-", "", -1)
	now := time.Now().Unix()

	one := TokenInfo{
		Token:    tk,
		Info:     info,
		UserID:   userid,
		CreateAt: int32(now),
		LastUse:  int32(now),
	}

	// insert token to DB
	if err := setToken(&one); err != nil {
		return "", err
	}

	// add token to Redis
	if err := setRedisCache([]string{tk}, []int32{userid}); err != nil {
		return tk, ErrCache
	}

	// add token to Map
	setToMap(tk, userid)

	return tk, nil
}

// GetUserID to get userid from token.
// return userid, got, error
// the error can be kktoken.CacheErr.
func GetUserID(token string) (int32, error) {
	var userid int32
	var err error

	// get userid from Map
	if userid = getAndSetMap(token); userid > 0 {
		return userid, nil
	}

	// get user id from cache
	if userid, err = getRedisCache(token); err != nil {
		return userid, err
	} else if userid > 0 {
		setToMap(token, userid)
		return userid, nil
	}

	// then, get from DB
	if userid, err = getUserID(token); err != nil {
		return userid, err
	} else if userid <= 0 {
		// not found from DB
		return userid, nil
	}

	// if in db, set to cache
	if err := setRedisCache([]string{token}, []int32{userid}); err != nil {
		return userid, err
	}

	// add token to Map
	setToMap(token, userid)

	return userid, nil
}

// DelToken to delete the token.
func DelToken(token string) error {
	allTokens.lock.Lock()
	delete(allTokens.all, token)
	allTokens.lock.Unlock()

	err1 := delRedisCache(token)
	err2 := delToken(token)
	if err1 != nil {
		return err1
	}
	return err2
}

// GetUserTokens to get all tokens of a user only from database
func GetUserTokens(userid int32) ([]TokenInfo, error) {
	return getAllTokens(userid)
}
