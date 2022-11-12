// /home/krylon/go/src/github.com/blicero/guang/frontend/ajax.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-11 20:56:48 krylon>

package frontend

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/blicero/guang/backend/facility"
	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/blicero/guang/database"
	"github.com/gorilla/mux"
	"github.com/pquerna/ffjson/ffjson"
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
		err     error
		tstr    string
		tstamp  int64
		db      *database.HostDB
		dbRes   []data.ScanResult
		refTime time.Time
		outbuf  []byte
		res     = ajaxDataPorts{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	if common.Debug {
		srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)
	}

	args := mux.Vars(r)
	tstr = args["stamp"]

	if tstamp, err = strconv.ParseInt(tstr, 10, 64); err != nil {
		res.Message = fmt.Sprintf("Could not parse timestamp %q: %s",
			tstr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n",
			res.Message)
		goto RESPOND
	}

	db = srv.dbPool.Get()
	defer srv.dbPool.Put(db)

	// refTime = srv.getCkPortstamp()
	// srv.updateCkPortstamp(time.Now())
	refTime = time.Unix(tstamp, 0)

	if dbRes, err = db.PortGetRecent(refTime); err != nil {
		srv.log.Printf("[ERROR] Failed to load recently scanned ports from database: %s\n",
			err.Error())
		res.Message = err.Error()
		goto RESPOND
	}

	res.Results = make(map[uint16][]data.ScanResult, len(dbRes))

	for _, r := range dbRes {
		var (
			ok bool
		)

		if _, ok = res.Results[r.Port]; !ok {
			res.Results[r.Port] = make([]data.ScanResult, 0, 64)
		}

		res.Results[r.Port] = append(res.Results[r.Port], r)
		res.Count++
	}

	res.Status = true

RESPOND:
	if outbuf, err = ffjson.Marshal(&res); err != nil {
		res.Message = fmt.Sprintf("Error serializing Response to %s: %s",
			r.RemoteAddr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
	} else {
		defer ffjson.Pool(outbuf)
	}

	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(outbuf)), 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(200)
	w.Write(outbuf) // nolint: errcheck
} // func (srv *WebFrontend) handlePortsRecent(w http.ResponseWriter, r *http.Request)

func (srv *WebFrontend) handleWorkerSpawn(w http.ResponseWriter, r *http.Request) {
	var (
		err                error
		facStr, cntStr     string
		oldCnt, cnt, facID int64
		fac                facility.Facility
		res                = ajaxCtlResponse{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)

	vars := mux.Vars(r)

	facStr = vars["facility"]
	cntStr = vars["cnt"]

	if cnt, err = strconv.ParseInt(cntStr, 10, 64); err != nil {
		res.Message = fmt.Sprintf("Cannot parse number of workers to spawn (%q): %s",
			cntStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	} else if facID, err = strconv.ParseInt(facStr, 10, 8); err != nil {
		res.Message = fmt.Sprintf("Cannot parse facility ID %q: %s",
			facStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	}

	fac = facility.Facility(facID)

	if cnt < 1 {
		res.Message = fmt.Sprintf("Invalid number of %s workers to spawn: %d",
			fac,
			cnt)
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	}

	oldCnt = int64(srv.nexus.WorkerCount(fac))

	srv.nexus.SpawnWorker(fac, int(cnt))

	res.NewCnt = int(oldCnt + cnt)
	res.Status = true

RESPOND:
	var outbuf []byte

	if outbuf, err = ffjson.Marshal(&res); err != nil {
		res.Message = fmt.Sprintf("Error serializing Response to %s: %s",
			r.RemoteAddr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
	} else {
		defer ffjson.Pool(outbuf)
	}

	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(outbuf)), 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(200)
	w.Write(outbuf) // nolint: errcheck
} // func (srv *WebFrontend) handleWorkerSpawn(w http.ResponseWriter, r *http.Request)

func (srv *WebFrontend) handleWorkerStop(w http.ResponseWriter, r *http.Request) {
	var (
		err                error
		facStr, cntStr     string
		oldCnt, cnt, facID int64
		fac                facility.Facility
		res                = ajaxCtlResponse{
			ajaxData: ajaxData{
				Timestamp: time.Now(),
			},
		}
	)

	srv.log.Printf("[TRACE] Handling request for %s\n", r.RequestURI)

	vars := mux.Vars(r)

	facStr = vars["facility"]
	cntStr = vars["cnt"]

	if cnt, err = strconv.ParseInt(cntStr, 10, 64); err != nil {
		res.Message = fmt.Sprintf("Cannot parse number of workers to spawn (%q): %s",
			cntStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	} else if facID, err = strconv.ParseInt(facStr, 10, 8); err != nil {
		res.Message = fmt.Sprintf("Cannot parse facility ID %q: %s",
			facStr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	}

	fac = facility.Facility(facID)
	oldCnt = int64(srv.nexus.WorkerCount(fac))

	if cnt < 1 {
		res.Message = fmt.Sprintf("Invalid number of %s workers to stop: %d",
			fac,
			cnt)
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	} else if cnt > oldCnt {
		res.Message = fmt.Sprintf("Too many %s workers to stop: %d but only %d are running",
			fac,
			cnt,
			oldCnt)
		srv.log.Printf("[ERROR] %s\n", res.Message)
		goto RESPOND
	}

	srv.nexus.StopWorker(fac, int(cnt))

	res.NewCnt = int(oldCnt - cnt)
	res.Status = true

RESPOND:
	var outbuf []byte

	if outbuf, err = ffjson.Marshal(&res); err != nil {
		res.Message = fmt.Sprintf("Error serializing Response to %s: %s",
			r.RemoteAddr,
			err.Error())
		srv.log.Printf("[ERROR] %s\n", res.Message)
	} else {
		defer ffjson.Pool(outbuf)
	}

	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(outbuf)), 10))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(200)
	w.Write(outbuf) // nolint: errcheck
} // func (srv *WebFrontend) handleWorkerStop(w http.ResponseWriter, r *http.Request)
