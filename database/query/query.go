// /home/krylon/go/src/github.com/blicero/guang/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 27. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-03 00:47:08 krylon>

// Package query provides symbolic constants for the various
// database queries/operations.
package query

//go:generate stringer -type=ID

// ID identifies a database query.
type ID int

// These constants identify the various database queries used in the application.
const (
	HostAdd ID = iota
	HostGetByID
	HostGetRandom
	HostGetCnt
	HostExists
	HostPortByPort
	PortAdd
	PortGetByHost
	PortGetReplyCnt
	PortGetOpen
	PortGetRecent
	XfrAdd
	XfrGetByZone
	XfrFinish
	XfrGetUnfinished
)
