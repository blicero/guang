// /home/krylon/go/src/github.com/blicero/guang/logdomain/id.go
// -*- mode: go; coding: utf-8; -*-
// Created on 29. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-10-31 22:25:56 krylon>

// Package logdomain provides symbolic constants to identify the various
// pieces of the application that need to do logging.
package logdomain

//go:generate stringer -type=ID

// ID is an id...
type ID uint8

// These constants represent the pieces of the application that need to log stuff.
const (
	Common ID = iota
	DBPool
	Database
	Backend
	Generator
	XFR
)

// AllDomains returns a slice of all the valid values for ID.
func AllDomains() []ID {
	return []ID{
		Common,
		DBPool,
		Database,
		Backend,
		Generator,
		XFR,
	}
} // func AllDomains() []ID
