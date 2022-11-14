// /home/krylon/go/src/github.com/blicero/guang/backend/facility/facility.go
// -*- mode: go; coding: utf-8; -*-
// Created on 10. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-13 17:01:15 krylon>

// Package facility provides symbolic constants to enumerate the compoentns
// of the application.
package facility

//go:generate stringer -type=Facility

// Facility is a symbolic constant identifying the moving parts of
// the application.
type Facility uint8

// These constants represent the parts that comprise Guang.
const (
	Generator Facility = iota
	Scanner
	XFR
)

// All returns all Facilities.
func All() []Facility {
	return []Facility{
		Generator,
		Scanner,
		XFR,
	}
} // func All() []Facility
