// /Users/krylon/go/src/guang/types.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-23 16:32:41 krylon>

package guang

import (
	"krylib"
	"net"
	"time"
)

const (
	HOST_SOURCE_USER = iota
	HOST_SOURCE_GEN
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
