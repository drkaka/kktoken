package kktoken

import "github.com/garyburd/redigo/redis"

var rdsPool *redis.Pool
var rdsLiveSeconds uint32

func setCache(token string, userid int32) {
    
}
