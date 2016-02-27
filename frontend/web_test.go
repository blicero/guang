// /Users/krylon/go/src/guang/frontend/web_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 13. 02. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2016-02-13 20:39:26 krylon>

package frontend

import "testing"

var web *WebFrontend

func TestCreate(t *testing.T) {
	var err error

	if web, err = CreateFrontend("", 4711, nil); err != nil {
		t.Fatalf("Error creating Web Frontend: %s", err.Error())
	}
}
