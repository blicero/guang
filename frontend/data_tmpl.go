// /home/krylon/go/src/github.com/blicero/guang/frontend/data_tmpl.go
// -*- mode: go; coding: utf-8; -*-
// Created on 31. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-13 17:06:47 krylon>

package frontend

import (
	"github.com/blicero/guang/backend/facility"
	"github.com/blicero/guang/data"
)

type tmplDataIndex struct {
	Debug        bool
	Title        string
	Error        []string
	Facilities   []facility.Facility
	HostGenCnt   int
	XFRCnt       int
	ScanCnt      int
	HostCnt      int64
	PortReplyCnt int64
}

type reportInfoPort struct {
	Port    uint16
	Results []data.ScanResult
}

// type host_scan_result struct {
// 	Host  *data.Host
// 	Ports []data.ScanResult
// }

type tmplDataByPort struct {
	Debug      bool
	Title      string
	Error      []string
	Facilities []facility.Facility
	Count      int
	Ports      map[uint16]reportInfoPort
}

// Donnerstag, 18. 08. 2016, 21:10
// Damit ich das in der HTML-Template gescheit verarbeiten kann, müsste ich
// eigentlich eine Liste von Strukturen haben, wo der Host und die Ports drin
// liegen.
// Oder? Ich könnte eine Methode schreiben, die den Host anhand der ID zurück
// gibt? In den Rohdaten aus der Datenbank steht der ja drin.
type tmplDataByHost struct {
	Debug      bool
	Title      string
	Error      []string
	Facilities []facility.Facility
	Count      int
	Hosts      []data.HostWithPorts
}
