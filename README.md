# kktoken
[![Build Status](https://travis-ci.org/drkaka/kktoken.svg)](https://travis-ci.org/drkaka/kktoken)
[![Coverage Status](https://codecov.io/github/drkaka/kktoken/coverage.svg?branch=master)](https://codecov.io/github/drkaka/kktoken?branch=master) 

The token module for golang project. The token is just auth token to identify user without always carrying userid in the request.

## Structure

![](https://github.com/drkaka/kktoken/blob/master/token.jpg)

## Database
It is using PostgreSQL as the database and will create a table:

```sql  
CREATE TABLE IF NOT EXISTS token (
	token uuid primary key,
	userid integer,
    expire integer
);
```

## Dependence

```Go
go get github.com/jackc/pgx
go get github.com/satori/go.uuid
go get github.com/garyburd/redigo/redis
```

## Usage 

####First need to use the module with the pgx pool, redis pool, db live seconds and cache live seconds passed in:
```Go
err := kktoken.Use(poolDB, poolRDS, uint32(300), uint32(30000000))
```

####Make and store token for user:
```Go
token, err := MakeToken(userid)
```

####Get userid from token:
```Go
userid, ok, err := GetUserID(token)
```
If token is not in cache, it will set to cache.

####Delete token:
```Go
err := DelToken(token)
```