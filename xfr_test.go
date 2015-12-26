// /Users/krylon/go/src/guang/xfr_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 26. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-26 00:59:49 krylon>

package guang

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

var xfr_client *XFRClient
var req_queue chan string

const REQ_ZONE = "krylon.net."

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

	if err = xfr_client.perform_xfr(REQ_ZONE); err != nil {
		t.Fatalf("Error performing XFR of %s: %s",
			REQ_ZONE, err.Error())
	}
} // func TestXFR(t *testing.T)
