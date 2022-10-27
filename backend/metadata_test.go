// /Users/krylon/go/src/guang/backend/metadata_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 08. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2022-10-27 21:23:56 krylon>

package backend

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/blicero/guang/data"
	"github.com/blicero/krylib"
)

var test_hosts []data.Host = []data.Host{
	data.Host{
		ID:      1,
		Address: net.ParseIP("62.153.71.106"),
		Name:    "vpn.wellmann-engineering.eu",
	},
	data.Host{
		ID:      2,
		Address: net.ParseIP("46.101.125.6"),
		Name:    "straylight.krylon.selfhost.eu",
	},
}

type host_location struct {
	City    string
	Country string
}

var test_locations map[krylib.ID]host_location = map[krylib.ID]host_location{
	1: host_location{Country: "Deutschland", City: ""},
	2: host_location{Country: "Deutschland", City: "Frankfurt am Main"},
}

var meta_engine *MetaEngine

func str_ptr(s string) *string {
	return &s
} // func str_ptr(s string) *string

func TestOpenMeta(t *testing.T) {
	var err error

	if meta_engine, err = OpenMetaEngine(filepath.Join(os.Getenv("HOME"), "guang.d/GeoLite2-City.mmdb")); err != nil {
		t.Fatalf("Error opening meta engine: %s", err.Error())
	} else if meta_engine == nil {
		t.Fatal("OpenMetaEngine() returned a nil value!")
	}
} // func TestOpenMeta(t *testing.T)

func TestLookupCountry(t *testing.T) {
	var err error
	var country string

	for _, host := range test_hosts {
		loc := test_locations[host.ID]

		if country, err = meta_engine.LookupCountry(&host); err != nil {
			t.Errorf("Error looking up country for %s: %s",
				host.Address, err.Error())
		} else if country != loc.Country {
			t.Errorf("Error looking up country for %s: Expected %s, Result %s",
				host.Address, loc.Country, country)
		}
	}
} // func TestLookupCountry(t *testing.T)

func TestLookupCity(t *testing.T) {
	var err error
	var city string

	for _, host := range test_hosts {
		loc := test_locations[host.ID]

		if city, err = meta_engine.LookupCity(&host); err != nil {
			t.Errorf("Error looking up city for %s: %s",
				host.Address, err.Error())
		} else if city != loc.City {
			t.Errorf("Error looking up City for %s: Expected %s, Result %s",
				host.Address, loc.City, city)
		}
	}
} // func TestLookupCity(t *testing.T)

func TestGuessOperatingSystem(t *testing.T) {
	var test_hosts_with_ports []data.HostWithPorts = []data.HostWithPorts{
		data.HostWithPorts{
			Host: data.Host{
				ID:      1,
				Name:    "host01.example.com",
				Address: net.ParseIP("127.0.0.1"),
			},
			Ports: []data.Port{
				data.Port{
					ID:     krylib.INVALID_ID,
					HostID: 1,
					Port:   80,
					Reply:  str_ptr("Apache 2.0.47 (Ubuntu)"),
				},
			},
		},
	}
	var os_map map[krylib.ID]string = map[krylib.ID]string{
		1: "Ubuntu",
	}

	for _, h := range test_hosts_with_ports {
		system := meta_engine.LookupOperatingSystem(&h)
		if system != os_map[h.Host.ID] {
			t.Errorf("Unexpected Operating System for host %s: Expected %s, Result %s",
				h.Host.Name,
				os_map[h.Host.ID],
				system)
		}
	}
} // func TestGuessOperatingSystem(t *testing.T)
