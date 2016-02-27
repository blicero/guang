// /Users/krylon/go/src/guang/database_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 25. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2016-02-12 21:34:00 krylon>

package backend

import (
	"fmt"
	"krylib"
	"math/rand"
	"net"
	"testing"
	"time"
)

var db *HostDB

var hosts []Host = []Host{
	Host{
		ID:      krylib.INVALID_ID,
		Address: net.ParseIP("192.168.0.1"),
		Name:    "finn.krylon.net",
		Source:  HOST_SOURCE_USER,
	},
	Host{
		ID:      krylib.INVALID_ID,
		Address: net.ParseIP("192.168.0.13"),
		Name:    "neuromancer.krylon.net",
		Source:  HOST_SOURCE_USER,
	},
	Host{
		ID:      krylib.INVALID_ID,
		Address: net.ParseIP("192.168.0.4"),
		Name:    "wintermute.krylon.net",
		Source:  HOST_SOURCE_USER,
	},
}

func TestCreateDatabase(t *testing.T) {
	var rng *rand.Rand = rand.New(rand.NewSource(time.Now().Unix()))

	var test_path string = fmt.Sprintf("/tmp/guang_test_%08x",
		rng.Int31())
	var err error

	fmt.Printf("Setting BASE_DIR to %s\n", test_path)
	SetBaseDir(test_path)

	if db, err = OpenDB(DB_PATH); err != nil {
		t.Fatalf("Error opening database at %s: %s",
			DB_PATH, err.Error())
	}
} // func TestCreateDatabase(t *testing.T)

func TestAddHosts(t *testing.T) {
	var err error

	if err = db.Begin(); err != nil {
		t.Fatalf("Error starting transaction: %s", err.Error())
	}

	for idx, host := range hosts {
		if err = db.HostAdd(&host); err != nil {
			t.Fatalf("Error adding host %s to database: %s",
				host.Name, err.Error())
		} else if host.ID == krylib.INVALID_ID {
			t.Errorf("After adding host %s, no ID was set!",
				host.Name)
		} else {
			hosts[idx].ID = host.ID
		}
	}

	if err = db.Commit(); err != nil {
		t.Fatalf("Error committing transaction: %s", err.Error())
	}
} // func TestAddHosts(t *testing.T)

func TestGetHosts(t *testing.T) {
	var err error
	var host *Host

	for _, ref_host := range hosts {
		if host, err = db.HostGetByID(ref_host.ID); err != nil {
			t.Fatalf("Error getting Host by ID #%d: %s",
				ref_host.ID, err.Error())
		} else if ref_host.ID != host.ID {
			t.Errorf("Host came back with the wrong ID: %d (expected) <-> %d (actual)",
				ref_host.ID, host.ID)
		} else if ref_host.Name != host.Name {
			t.Errorf("Host came back with the wrong name: %s (expected) <-> %s (actual)",
				ref_host.Name, host.Name)
		}
	}
} // func TestGetHosts(t *testing.T)

var xfr *XFR

func TestAddXFR(t *testing.T) {
	var err error
	xfr = &XFR{
		ID:     krylib.INVALID_ID,
		Zone:   "krylon.net",
		Start:  time.Now(),
		Status: XFR_STATUS_UNFINISHED,
	}

	if err = db.XfrAdd(xfr); err != nil {
		t.Fatalf("Error adding xfr: %s", err.Error())
	} else if xfr.ID == krylib.INVALID_ID {
		t.Fatalf("Error: XFR was added without error, but it didn't get an ID!")
	}
} // func TestAddXFR(t *testing.T)

func TestFinishXFR(t *testing.T) {
	var err error
	if err = db.XfrFinish(xfr, XFR_STATUS_SUCCESS); err != nil {
		t.Fatalf("Error finishing XFR: %s", err.Error())
	}
} // func TestFinishXFR(t *testing.T)

func TestPortAdd(t *testing.T) {
	var err error
	var test_reply string = "Wer das liest, ist doof."
	var res ScanResult = ScanResult{
		Host:  hosts[0],
		Port:  22,
		Reply: &test_reply,
		Stamp: time.Now(),
	}

	if err = db.PortAdd(&res); err != nil {
		t.Fatalf("Error adding ScanResult: %s", err.Error())
	}
} // func TestPortAdd(t *testing.T)

func TestPortCnt(t *testing.T) {
	var err error
	var cnt int64

	if cnt, err = db.PortGetReplyCount(); err != nil {
		t.Errorf("Error getting reply count from database: %s",
			err.Error())
	} else if cnt < 1 {
		t.Fatalf("Invalid/Unexpected reply count: %d", cnt)
	}
}
