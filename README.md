# kktoken [![Build Status][ci-img]][ci] [![Coverage Status][cov-img]][cov]

There are three levels for token usage. First go to Map, then go to Redis and finally go to PostgreSQL. Every given EXPCheckSecond for MapInfo, it will check expirations in Map, the expired records will update last_use information to DB and active records will refresh cache in Redis. Every given EXPCheckSecond for DBInfo, it will check expirations in PostgreSQL and delete the expired tokens.

## Database

It is using PostgreSQL as the database and will create a table with the given table name (default: token):

```sql
CREATE TABLE IF NOT EXISTS token (
  token UUID PRIMARY KEY,
  user_id INTEGER NOT NULL,
  info JSONB,
  create_at INTEGER NOT NULL,
  last_use INTEGER NOT NULL
);
```

And index on user_id to serach all tokens for a user.

```sql
CREATE INDEX IF NOT EXISTS token_user_id_index ON token USING btree (user_id);
```

And index on last_use to serach expired token.

```sql
CREATE INDEX IF NOT EXISTS token_last_use_index ON token USING btree (last_use);
```

## Dependence

```Go
go get github.com/jackc/pgx
go get github.com/satori/go.uuid
go get github.com/garyburd/redigo/redis
```

## Usage

First need to use the module with configurations:

```Go
dbInfo := &DBInfo{
  Pool:      poolDB,
  TableName: testTableName,
  PersistentSecond: 0, // 0 for never expire
  EXPCheckSecond: 300, // default: 300
}

rdsInfo := &RDSInfo{
  Pool: poolRDS,
  LiveSecond: 300, // default: 300
}

mapInfo := &MapInfo{
  LiveSecond: 60, // default: 60
  EXPCheckSecond: 31, // default: 31
}

// errChan to receive errors generated from background goroutines
errChan, err := kktoken.Use(dbInfo, rdsInfo, mapInfo)
```

Make and store token for userid and related info:

```Go
info := map[string]interface{}{
	"device": "ios",
}
token, err := MakeToken(userid, info)
```

Get userid from token:

```Go
userid, err := GetUserID(token)
```

If token is not in cache, it will set to Map and Cache. Every call will update last_use of a certain and flush to DB when MapEXPCheck happen.

Delete token

```Go
err := DelToken(token)
```

Get all token information:

```Go
tokens, err = GetUserTokens(userid)
```

[ci-img]: https://travis-ci.org/drkaka/kktoken.svg?branch=master
[ci]: https://travis-ci.org/drkaka/kktoken
[cov-img]: https://coveralls.io/repos/github/drkaka/kktoken/badge.svg?branch=master
[cov]: https://coveralls.io/github/drkaka/kktoken?branch=master