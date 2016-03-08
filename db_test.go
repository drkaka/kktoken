package kktoken

import (
	"testing"
	"time"

	"github.com/jackc/pgx"
	"github.com/satori/go.uuid"
)

func testTableGeneration(t *testing.T) {
	var dbname pgx.NullString
	if err := dbPool.QueryRow("SELECT 'public.token'::regclass;").Scan(&dbname); err != nil {
		t.Fatal(err)
	}

	if dbname.String != "token" {
		t.Fatal("dbname is not correct.")
	}
}

func testDBMethods(t *testing.T) {
	getEmpty(t)
	deleteEmpty(t)
	testSetGetDelete(t)
}

func deleteEmpty(t *testing.T) {
	tk := uuid.NewV1().String()
	if err := delToken(tk); err != nil {
		t.Error(err)
	}
}

func getEmpty(t *testing.T) {
	if _, _, err := getUserID("aa"); err == nil {
		t.Error("Should have db error.")
	}

	if _, ok, err := getUserID(uuid.NewV1().String()); err != nil {
		t.Error(err)
	} else if ok {
		t.Error("Should not have result.")
	}
}

func testSetGetDelete(t *testing.T) {
	tk := uuid.NewV1().String()
	userid := int32(3)
	exp := uint32(time.Now().Unix()) + dbLiveSeconds

	if err := setToken(tk, userid, exp); err != nil {
		t.Error(err)
	}

	if uid, ok, err := getUserID(tk); err != nil {
		t.Error(err)
	} else if !ok {
		t.Error("Should have get user id.")
	} else if uid != userid {
		t.Error("Result is wrong.")
	}

	if err := delToken(tk); err != nil {
		t.Error(err)
	}

	if _, ok, err := getUserID(tk); err != nil {
		t.Error(err)
	} else if ok {
		t.Error("Should not have get user id.")
	}
}
