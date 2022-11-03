// /home/krylon/go/src/github.com/blicero/guang/frontend/ajax.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-03 01:59:23 krylon>

package frontend

import (
	"fmt"
	"net/http"
	"time"

	"github.com/blicero/guang/common"
	"github.com/blicero/guang/database"
)

// This file is the home of the AJAX handler methods.

func (srv *WebFrontend) handleBeacon(w http.ResponseWriter, r *http.Request) {
	// It doesn't bother me enough to do anything about it other
	// than writing this comment, but this method is probably
	// grossly inefficient re memory.
	var timestamp = time.Now().Format(common.TimestampFormat)
	const appName = common.AppName + " " + common.Version
	var jstr = fmt.Sprintf(`{ "Status": true, "Message": "%s", "Timestamp": "%s", "Hostname": "%s" }`,
		appName,
		timestamp,
		hostname())
	var response = []byte(jstr)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.WriteHeader(200)
	w.Write(response) // nolint: errcheck,gosec
} // func (srv *WebFrontend) handleBeacon(w http.ResponseWriter, r *http.Request)

func (srv *WebFrontend) handlePortsRecent(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		db  *database.HostDB
	)
} // func (srv *WebFrontend) handlePortsRecent(w http.ResponseWriter, r *http.Request)
