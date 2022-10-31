// /home/krylon/go/src/github.com/blicero/guang/logdomain/id.go
// -*- mode: go; coding: utf-8; -*-
// Created on 29. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-10-31 19:39:52 krylon>

// Package logdomain provides symbolic constants to identify the various
// pieces of the application that need to do logging.
package logdomain

//go:generate stringer -type=ID

// ID is an id...
type ID uint8

const (
	Common ID = iota
	DBPool
	Database
	Backend
	Generator
	XFR
)
