package kktoken

import (
	"fmt"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func testTableGeneration(tableName string, t *testing.T) {
	var table string
	err := dbPool.QueryRow(fmt.Sprintf("SELECT 'public.%s'::regclass;", tableName)).Scan(&table)
	assert.NoError(t, err, "should not have error")
	assert.Equal(t, tableName, table, "table name wrong")
}

func testDBMethods(t *testing.T) {
	getEmpty(t)
	deleteEmpty(t)
	testCRUD(t)
	testEXPCheck(t)
}

func deleteEmpty(t *testing.T) {
	err := delToken(uuid.NewV1().String())
	assert.NoError(t, err, "should not have error to delete non-existed token")
}

func getEmpty(t *testing.T) {
	userid, err := getUserID("aa")
	assert.NoError(t, err, "should not have error with an invalid UUID")
	assert.EqualValues(t, 0, userid, "userid should be 0 when not exist")

	userid, err = getUserID(uuid.NewV1().String())
	assert.NoError(t, err, "should not have error to get non-exsted userid")
	assert.EqualValues(t, 0, userid, "userid should be 0 when not exist")
}

func testEXPCheck(t *testing.T) {
	// generate a token
	tk := uuid.NewV1().String()
	userid := int32(3)

	info := map[string]interface{}{
		"device": "ios",
	}

	now := int32(time.Now().Unix())
	tkInfo := &TokenInfo{
		Token:    tk,
		Info:     info,
		UserID:   userid,
		CreateAt: now,
		LastUse:  now,
	}

	// should be ok to set
	err := setToken(tkInfo)
	assert.NoError(t, err, "should not have error to set token")

	dbPersistentSecond = 1
	go startDBEXPCheck(2, testTableName)
	time.Sleep(2100 * time.Millisecond)

	// after 2 second and check
	gotUserid, err := getUserID(tk)
	assert.NoError(t, err, "should not have error to get user id")
	assert.Equal(t, int32(0), gotUserid, "userid result wrong")
}

func testCRUD(t *testing.T) {
	tk := uuid.NewV1().String()
	userid := int32(3)

	info := map[string]interface{}{
		"device": "ios",
	}

	now := int32(time.Now().Unix())
	tkInfo := &TokenInfo{
		Token:    tk,
		Info:     info,
		UserID:   userid,
		CreateAt: now,
		LastUse:  now,
	}

	// should be ok to set
	err := setToken(tkInfo)
	assert.NoError(t, err, "should not have error to set token")

	// can't set with invalid token
	tkInfo.Token = "abc"
	err = setToken(tkInfo)
	assert.Error(t, err, "should have error to set an invalid token")

	// should be ok to get userid
	gotUserid, err := getUserID(tk)
	assert.NoError(t, err, "should not have error to get user id")
	assert.Equal(t, userid, gotUserid, "userid result wrong")

	// should be ok to get even after dbPersistentSecond set
	dbPersistentSecond = 2
	gotUserid, err = getUserID(tk)
	assert.NoError(t, err, "should not have error to get user id")
	assert.Equal(t, userid, gotUserid, "userid result wrong")

	// token should be invalid after 2 second
	time.Sleep(2 * time.Second)
	gotUserid, err = getUserID(tk)
	assert.NoError(t, err, "should not have error to get user id")
	assert.Equal(t, int32(0), gotUserid, "userid result wrong")

	// update last_use
	now = int32(time.Now().Unix())
	err = updateToken(tk, now)
	assert.NoError(t, err, "should not have error to update token")

	// the userid should be valid again
	gotUserid, err = getUserID(tk)
	assert.NoError(t, err, "should not have error to get user id")
	assert.Equal(t, userid, gotUserid, "userid result wrong")

	// update a non-existed token
	err = updateToken(uuid.NewV1().String(), now)
	assert.NoError(t, err, "should not have error to update a non-existed token")

	// get all tokens of a user
	tokens, err := getAllTokens(userid)
	assert.NoError(t, err, "should not have error to get all tokens")
	assert.Len(t, tokens, 1, "all tokens length wrong")
	assert.Equal(t, "ios", tokens[0].Info["device"], "info wrong")

	// user not existed
	tokens, err = getAllTokens(int32(5))
	assert.NoError(t, err, "should not have error to get all tokens")
	assert.Len(t, tokens, 0, "all tokens length wrong")

	// delete
	err = delToken(tk)
	assert.NoError(t, err, "should not have error to delete token")

	// the userid should not exist after delete.
	gotUserid, err = getUserID(tk)
	assert.NoError(t, err, "should not have error to get user id")
	assert.Equal(t, int32(0), gotUserid, "userid result wrong")
}
