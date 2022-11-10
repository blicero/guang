// /Users/krylon/go/src/guang/types.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-11-09 22:08:20 krylon>

// Package data provides data types used throughout the application.
package data

import (
	"net"
	"time"

	"github.com/blicero/guang/xfr/xfrstatus"
	"github.com/blicero/krylib"
)

//go:generate stringer -type=HostSource

// HostSource indicates how a Host ended up in the database.
type HostSource int

// HostSourceUser indicates a host was manually added by the user.
// HostSourceGen indicates it was added by the HostGenerator.
// HostSourceA indicates it was gathered from an address record in a
// zone transfer.
// HostSourceMx and HostSourceNs indicate a Host was gathered from the
// respective records in a zone transfer.
const (
	HostSourceUser HostSource = iota
	HostSourceGen
	HostSourceA
	HostSourceMx
	HostSourceNs
)

// Host is a host somewhere on the Internet.
type Host struct {
	ID      krylib.ID
	Source  HostSource
	Address net.IP
	Name    string
	Added   time.Time
}

// Port is a TCP/UDP port that was scanned on a given host.
type Port struct {
	ID        krylib.ID
	HostID    krylib.ID
	Port      uint16
	Timestamp time.Time
	Reply     *string
}

// ReplyString returns the Reply gathered from the Port or an empty string.
func (p *Port) ReplyString() string {
	if p.Reply == nil {
		return ""
	}

	return *p.Reply
} // func (p *Port) ReplyString() string

// HostWithPorts is a Host along with all the Ports that have been scanned
// on that Host.
type HostWithPorts struct {
	Host  Host
	Ports []Port
}

// XFR represents a DNS zone transfer.
type XFR struct {
	ID     krylib.ID
	Zone   string
	Start  time.Time
	End    time.Time
	Status xfrstatus.XfrStatus
}

// XfrNew creates a new XFR.
func XfrNew(zone string) *XFR {
	return &XFR{
		ID:     krylib.INVALID_ID,
		Zone:   zone,
		Start:  time.Now(),
		Status: xfrstatus.Unfinished,
	}
} // func XfrNew(zone string) *XFR

// IsFinished returns true if the XFR has been finished (successfully or not).
func (x *XFR) IsFinished() bool {
	return x.Status != xfrstatus.Unfinished
} // func (self *XFR) IsFinished() bool

// ScanRequest is a request to scan a specific port on a given host
type ScanRequest struct {
	Host Host
	Port uint16
}

// ScanResult represents the result of scanning a single port.
type ScanResult struct {
	Host  Host
	Port  uint16
	Reply *string
	Stamp time.Time
	Err   error
}

// HostName returns the hostname of the scanned Host.
func (res *ScanResult) HostName() string {
	return res.Host.Name
} // func (self *ScanResult) HostName() string

// Address returns the IP address of the scanned host as a string.
func (res *ScanResult) Address() string {
	return res.Host.Address.String()
} // func (self *ScanResult) Address() string

// ReplyString returns the reply gathered from the scanned port.
func (res *ScanResult) ReplyString() string {
	if res.Reply == nil {
		return ""
	}

	return *res.Reply
} // func (self *ScanResult) ReplyString() string

//go:generate stringer -type=ControlMessage

// ControlMessage is a symbolic constant signifying a message send to
// the Nexus.
type ControlMessage int

// CtlMsgStop tells the Nexus to shut down.
// CtlMsgStatus asks the Nexus for information on its current status.
const (
	CtlMsgStop ControlMessage = iota
	CtlMsgShutdown
	CtlMsgStatus
	CtlMsgSpawn
	CtlMsgBye
)
