// /Users/krylon/go/src/guang/xfr_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 26. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-30 21:45:31 krylon>

package xfr

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/blicero/guang/common"
	"github.com/blicero/guang/database"
)

var xfrClient *Client
var requestQueue chan string

const reqZone = "krylon.net."
const reqZoneFail = "example.com."

func TestCreateClient(t *testing.T) {
	var rng *rand.Rand = rand.New(rand.NewSource(time.Now().Unix()))
	var testPath string = fmt.Sprintf("/tmp/guang_test_%08x",
		rng.Int31())
	var err error

	fmt.Printf("Setting BASE_DIR to %s\n", testPath)
	common.SetBaseDir(testPath)

	requestQueue = make(chan string)

	if xfrClient, err = MakeXFRClient(requestQueue); err != nil {
		t.Fatalf("Error creating XFRClient: %s", err.Error())
	}
} // func TestCreateClient(t *testing.T)

func TestXFR(t *testing.T) {
	var err error
	var db *database.HostDB

	if db, err = database.OpenDB(common.DbPath); err != nil {
		t.Fatalf("Error opening database: %s", err.Error())
	} else {
		defer db.Close()
	}

	if err = xfrClient.performXfr(reqZone, db); err != nil {
		t.Fatalf("Error performing XFR of %s: %s",
			reqZone, err.Error())
	}
} // func TestXFR(t *testing.T)

func TestXFRFail(t *testing.T) {
	var err error
	var db *database.HostDB

	if db, err = database.OpenDB(common.DbPath); err != nil {
		t.Fatalf("Error opening HostDB at %s: %s",
			common.DbPath, err.Error())
	} else if err = xfrClient.performXfr(reqZoneFail, db); err == nil {
		t.Fatalf("Well THAT was unexpected: XFR of %s should have failed, but apparently it did not.",
			reqZoneFail)
	}
} // func TestXFRFail(t *testing.T)
