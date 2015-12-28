// /Users/krylon/go/src/guang/xfr_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 26. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-28 15:05:32 krylon>

package backend

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

var xfr_client *XFRClient
var req_queue chan string

const REQ_ZONE = "krylon.net."

const REQ_ZONE_FAIL = "example.com."

func TestCreateClient(t *testing.T) {
	var rng *rand.Rand = rand.New(rand.NewSource(time.Now().Unix()))
	var test_path string = fmt.Sprintf("/tmp/guang_test_%08x",
		rng.Int31())
	var err error

	fmt.Printf("Setting BASE_DIR to %s\n", test_path)
	SetBaseDir(test_path)

	req_queue = make(chan string)

	if xfr_client, err = MakeXFRClient(req_queue); err != nil {
		t.Fatalf("Error creating XFRClient: %s", err.Error())
	}
} // func TestCreateClient(t *testing.T)

func TestXFR(t *testing.T) {
	var err error
	var db *HostDB

	if db, err = OpenDB(DB_PATH); err != nil {
		t.Fatalf("Error opening database: %s", err.Error())
	} else {
		defer db.Close()
	}

	if err = xfr_client.perform_xfr(REQ_ZONE, db); err != nil {
		t.Fatalf("Error performing XFR of %s: %s",
			REQ_ZONE, err.Error())
	}
} // func TestXFR(t *testing.T)

func TestXFRFail(t *testing.T) {
	var err error
	var db *HostDB

	if db, err = OpenDB(DB_PATH); err != nil {
		t.Fatalf("Error opening HostDB at %s: %s",
			DB_PATH, err.Error())
	} else if err = xfr_client.perform_xfr(REQ_ZONE_FAIL, db); err == nil {
		t.Fatalf("Well THAT was unexpected: XFR of %s should have failed, but apparently it did not.",
			REQ_ZONE_FAIL)
	}
} // func TestXFRFail(t *testing.T)
