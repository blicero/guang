// /Users/krylon/go/src/guang/types.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-27 16:32:03 krylon>

package backend

import (
	"krylib"
	"net"
	"time"
)

const (
	HOST_SOURCE_USER = iota
	HOST_SOURCE_GEN
	HOST_SOURCE_A
	HOST_SOURCE_MX
	HOST_SOURCE_NS
)

type HostSource int

type Host struct {
	ID      krylib.ID
	Source  HostSource
	Address net.IP
	Name    string
	Added   time.Time
}

type Port struct {
	ID        krylib.ID
	HostID    krylib.ID
	Port      uint16
	Timestamp time.Time
	Reply     string
}

const (
	XFR_STATUS_UNFINISHED = 0
	XFR_STATUS_SUCCESS    = 1
	XFR_STATUS_REFUSED    = 2
	XFR_STATUS_ABORT      = 3
)

type XfrStatus int

func (self XfrStatus) String() string {
	switch self {
	case XFR_STATUS_UNFINISHED:
		return "Unfinished"
	case XFR_STATUS_SUCCESS:
		return "Finished/Success"
	case XFR_STATUS_REFUSED:
		return "Finished/Refused"
	case XFR_STATUS_ABORT:
		return "Finished/Aborted"
	default:
		return "INVALID STATUS!!!"
	}
} // func (XfrStatus self) String() string

type XFR struct {
	ID     krylib.ID
	Zone   string
	Start  time.Time
	End    time.Time
	Status XfrStatus
}

func XfrNew(zone string) *XFR {
	return &XFR{
		ID:     krylib.INVALID_ID,
		Zone:   zone,
		Start:  time.Now(),
		Status: XFR_STATUS_UNFINISHED,
	}
} // func XfrNew(zone string) *XFR

func (self *XFR) IsFinished() bool {
	return self.Status != XFR_STATUS_UNFINISHED
} // func (self *XFR) IsFinished() bool
