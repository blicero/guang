// /Users/krylon/go/src/guang/backend/metadata_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 08. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2022-10-31 19:04:41 krylon>

package backend

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/blicero/guang/data"
	"github.com/blicero/krylib"
)

var testHosts []data.Host = []data.Host{
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

type hostLocation struct {
	City    string
	Country string
}

var testLocations map[krylib.ID]hostLocation = map[krylib.ID]hostLocation{
	1: hostLocation{Country: "Deutschland", City: ""},
	2: hostLocation{Country: "Deutschland", City: "Frankfurt am Main"},
}

var metaEngine *MetaEngine

func strPtr(s string) *string {
	return &s
} // func strPtr(s string) *string

func TestOpenMeta(t *testing.T) {
	var err error

	if metaEngine, err = OpenMetaEngine(filepath.Join(os.Getenv("HOME"), "guang.d/GeoLite2-City.mmdb")); err != nil {
		t.Fatalf("Error opening meta engine: %s", err.Error())
	} else if metaEngine == nil {
		t.Fatal("OpenMetaEngine() returned a nil value!")
	}
} // func TestOpenMeta(t *testing.T)

func TestLookupCountry(t *testing.T) {
	var err error
	var country string

	for _, host := range testHosts {
		loc := testLocations[host.ID]

		if country, err = metaEngine.LookupCountry(&host); err != nil {
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

	for _, host := range testHosts {
		loc := testLocations[host.ID]

		if city, err = metaEngine.LookupCity(&host); err != nil {
			t.Errorf("Error looking up city for %s: %s",
				host.Address, err.Error())
		} else if city != loc.City {
			t.Errorf("Error looking up City for %s: Expected %s, Result %s",
				host.Address, loc.City, city)
		}
	}
} // func TestLookupCity(t *testing.T)

func TestGuessOperatingSystem(t *testing.T) {
	var testHostsWithPorts []data.HostWithPorts = []data.HostWithPorts{
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
					Reply:  strPtr("Apache 2.0.47 (Ubuntu)"),
				},
			},
		},
	}
	var osMap map[krylib.ID]string = map[krylib.ID]string{
		1: "Ubuntu",
	}

	for _, h := range testHostsWithPorts {
		system := metaEngine.LookupOperatingSystem(&h)
		if system != osMap[h.Host.ID] {
			t.Errorf("Unexpected Operating System for host %s: Expected %s, Result %s",
				h.Host.Name,
				osMap[h.Host.ID],
				system)
		}
	}
} // func TestGuessOperatingSystem(t *testing.T)
