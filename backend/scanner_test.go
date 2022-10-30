// /Users/krylon/go/src/guang/backend/scanner_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 05. 02. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2022-10-30 20:34:58 krylon>

package backend

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/blicero/guang/data"
	"github.com/blicero/krylib"
)

type scan_target struct {
	Host  data.Host
	Ports []uint16
}

// It is a little sparse for now, but I want it to work first.
// I can easily add more targets later.
var targets []scan_target = []scan_target{
	scan_target{
		Host: data.Host{
			ID:      krylib.INVALID_ID,
			Source:  data.HostSourceUser,
			Address: net.ParseIP("10.10.0.1"),
			Name:    "wintermute.krylon.net",
			Added:   time.Now(),
		},

		Ports: []uint16{21, 22, 79, 80},
	},

	scan_target{
		Host: data.Host{
			ID:      krylib.INVALID_ID,
			Source:  data.HostSourceUser,
			Address: net.ParseIP("10.10.0.7"),
			Name:    "neuromancer.krylon.net",
			Added:   time.Now(),
		},

		Ports: []uint16{22, 23, 79, 5900},
	},

	scan_target{
		Host: data.Host{
			ID:      krylib.INVALID_ID,
			Source:  data.HostSourceUser,
			Address: net.ParseIP("10.10.0.3"),
			Name:    "dixie.krylon.net",
			Added:   time.Now(),
		},

		Ports: []uint16{22, 5900},
	},

	scan_target{
		Host: data.Host{
			ID:      krylib.INVALID_ID,
			Source:  data.HostSourceUser,
			Address: net.ParseIP("10.10.8.10"),
			Name:    "achtfaden.krylon.net",
			Added:   time.Now(),
		},
		Ports: []uint16{22, 79},
	},

	// scan_target{
	// 	Host: data.Host{
	// 		ID:      krylib.INVALID_ID,
	// 		Source:  data.HOST_SOURCE_USER,
	// 		Address: net.ParseIP("192.168.0.4"),
	// 		Name:    "wintermute.krylon.net",
	// 		Added:   time.Now(),
	// 	},

	// 	Ports: []uint16{21, 22, 53, 80},
	// },
}

var scanner *Scanner

func TestCreateScanner(t *testing.T) {
	var err error

	if scanner, err = CreateScanner(1); err != nil {
		t.Fatalf("Error creating scanner: %s\n", err.Error())
	}
}

func TestPerformScan(t *testing.T) {
	var result *data.ScanResult
	var err error
	for _, target := range targets {
		msg := fmt.Sprintf("Scanning host %s (%s)...\n",
			target.Host.Name,
			target.Host.Address.String())
		fmt.Println(msg)
		for _, port_no := range target.Ports {
			result, err = scan_host(&target.Host, port_no)
			if err != nil {
				t.Errorf("Error scanning %s:%d - %s",
					target.Host.Name, port_no, err.Error())
			} else if result == nil {
				t.Errorf("Error scanning %s:%d - no result!",
					target.Host.Name, port_no)
			} else {
				var reply_str string
				if result.Reply != nil {
					reply_str = *result.Reply
				} else {
					reply_str = "(NULL)"
				}
				fmt.Printf("%s:%d -- %s\n",
					target.Host.Name, port_no, reply_str)
			}
		}
	}
}
