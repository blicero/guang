// /home/krylon/go/src/github.com/blicero/guang/frontend/ajax_types.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-03 20:56:02 krylon>

package frontend

import (
	"time"

	"github.com/blicero/guang/data"
)

//go:generate ffjson ajax_types.go

type ajaxData struct {
	Status    bool
	Message   string
	Timestamp time.Time
}

type ajaxDataPorts struct {
	ajaxData
	Results map[uint16][]data.ScanResult
}
