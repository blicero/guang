// /home/krylon/go/src/github.com/blicero/guang/frontend/helpers_web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-07 19:01:02 krylon>

package frontend

import "time"

// Methods that are not directly handling HTTP requests live here.

// func (srv *WebFrontend) getCkPortstamp() time.Time {
// 	srv.lock.RLock()
// 	var s = srv.ckPortStamp
// 	srv.lock.RUnlock()
// 	return s
// } // func (srv *WebFrontend) getCkPortstamp() time.Time

func (srv *WebFrontend) updateCkPortstamp(t time.Time) {
	srv.lock.Lock()
	srv.ckPortStamp = t
	srv.lock.Unlock()
} // func (srv *WebFrontend) updateCkPortstamp(t time.Time)
