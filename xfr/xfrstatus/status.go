// /home/krylon/go/src/github.com/blicero/guang/xfr/xfrstatus/status.go
// -*- mode: go; coding: utf-8; -*-
// Created on 30. 10. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-10-30 20:39:02 krylon>

package xfrstatus

//go:generate stringer -type=XfrStatus

type XfrStatus int

const (
	Unfinished XfrStatus = iota
	Success
	Refused
	Abort
)

// func (self XfrStatus) String() string {
// 	switch self {
// 	case Unfinished:
// 		return "Unfinished"
// 	case Success:
// 		return "Finished/Success"
// 	case Refused:
// 		return "Finished/Refused"
// 	case Abort:
// 		return "Finished/Aborted"
// 	default:
// 		return fmt.Sprintf("INVALID STATUS (%d)!!!", self)
// 	}
// } // func (XfrStatus self) String() string
