// /Users/krylon/go/src/guang/database_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 25. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2023-03-27 11:14:51 krylon>

package database

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/blicero/guang/xfr/xfrstatus"
	"github.com/blicero/krylib"
)

var db *HostDB

var hosts []data.Host = []data.Host{
	data.Host{
		ID:      krylib.INVALID_ID,
		Address: net.ParseIP("192.168.0.1"),
		Name:    "finn.krylon.net",
		Source:  data.HostSourceUser,
	},
	data.Host{
		ID:      krylib.INVALID_ID,
		Address: net.ParseIP("192.168.0.13"),
		Name:    "neuromancer.krylon.net",
		Source:  data.HostSourceUser,
	},
	data.Host{
		ID:      krylib.INVALID_ID,
		Address: net.ParseIP("192.168.0.4"),
		Name:    "wintermute.krylon.net",
		Source:  data.HostSourceUser,
	},
}

func TestCreateDatabase(t *testing.T) {
	var rng *rand.Rand = rand.New(rand.NewSource(time.Now().Unix()))

	var testPath string = fmt.Sprintf("/tmp/guang_test_%08x",
		rng.Int31())
	var err error

	fmt.Printf("Setting BASE_DIR to %s\n", testPath)
	common.SetBaseDir(testPath)

	if db, err = OpenDB(common.DbPath); err != nil {
		t.Fatalf("Error opening database at %s: %s",
			common.DbPath, err.Error())
	}
} // func TestCreateDatabase(t *testing.T)

func TestQueries(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}

	var err error

	for id := range dbQueries {
		if _, err = db.getStatement(id); err != nil {
			t.Errorf("Failed to prepare query %s", id)
		}
	}
} // func TestQueries(t *testing.T)

func TestAddHosts(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}
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
	if db == nil {
		t.SkipNow()
	}
	var err error
	var host *data.Host

	for _, refHost := range hosts {
		if host, err = db.HostGetByID(refHost.ID); err != nil {
			t.Fatalf("Error getting Host by ID #%d: %s",
				refHost.ID, err.Error())
		} else if refHost.ID != host.ID {
			t.Errorf("Host came back with the wrong ID: %d (expected) <-> %d (actual)",
				refHost.ID, host.ID)
		} else if refHost.Name != host.Name {
			t.Errorf("Host came back with the wrong name: %s (expected) <-> %s (actual)",
				refHost.Name, host.Name)
		}
	}
} // func TestGetHosts(t *testing.T)

var xfrClient *data.XFR

func TestAddXFR(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}
	var err error
	xfrClient = &data.XFR{
		ID:     krylib.INVALID_ID,
		Zone:   "krylon.net",
		Start:  time.Now(),
		Status: xfrstatus.Unfinished,
	}

	if err = db.XfrAdd(xfrClient); err != nil {
		t.Fatalf("Error adding xfr: %s", err.Error())
	} else if xfrClient.ID == krylib.INVALID_ID {
		t.Fatalf("Error: XFR was added without error, but it didn't get an ID!")
	}
} // func TestAddXFR(t *testing.T)

func TestFinishXFR(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}
	var err error
	if err = db.XfrFinish(xfrClient, xfrstatus.Success); err != nil {
		t.Fatalf("Error finishing XFR: %s", err.Error())
	}
} // func TestFinishXFR(t *testing.T)

func TestPortAdd(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}
	var err error
	var testReply string = "Wer das liest, ist doof."
	var res data.ScanResult = data.ScanResult{
		Host:  hosts[0],
		Port:  22,
		Reply: &testReply,
		Stamp: time.Now(),
	}

	if err = db.PortAdd(&res); err != nil {
		t.Fatalf("Error adding ScanResult: %s", err.Error())
	}
} // func TestPortAdd(t *testing.T)

func TestPortCnt(t *testing.T) {
	if db == nil {
		t.SkipNow()
	}
	var err error
	var cnt int64

	if cnt, err = db.PortGetReplyCount(); err != nil {
		t.Errorf("Error getting reply count from database: %s",
			err.Error())
	} else if cnt < 1 {
		t.Fatalf("Invalid/Unexpected reply count: %d", cnt)
	}
}
