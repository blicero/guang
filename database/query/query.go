// /home/krylon/go/src/github.com/blicero/guang/database/query/query.go
// -*- mode: go; coding: utf-8; -*-
// Created on 27. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-10-27 20:07:11 krylon>

// Package query provides symbolic constants for the various
// database queries/operations.
package query

//go:generate stringer -type=QueryID

// QueryID identifies a database query.
type QueryID int

const (
	STMT_HOST_ADD QueryID = iota
	STMT_HOST_GET_BY_ID
	STMT_HOST_GET_RANDOM
	STMT_HOST_GET_CNT
	STMT_HOST_EXISTS
	STMT_HOST_PORT_BY_HOST
	STMT_PORT_ADD
	STMT_PORT_GET_BY_HOST
	STMT_PORT_GET_REPLY_CNT
	STMT_PORT_GET_OPEN
	STMT_XFR_ADD
	STMT_XFR_GET_BY_ZONE
	STMT_XFR_FINISH
	STMT_XFR_GET_UNFINISHED
)
