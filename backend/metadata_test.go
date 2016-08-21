// /Users/krylon/go/src/guang/backend/metadata_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 08. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2016-08-21 17:13:56 krylon>

package backend

import (
	"krylib"
	"net"
	"testing"
)

// const (
// 	TEST_ADDR = "62.153.71.106"
// 	TEST_NAME = "vpn.wellmann-engineering.eu"
// )

var test_hosts []Host = []Host{
	Host{
		ID:      1,
		Address: net.ParseIP("62.153.71.106"),
		Name:    "vpn.wellmann-engineering.eu",
	},
	Host{
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

func TestOpenMeta(t *testing.T) {
	var err error

	if meta_engine, err = OpenMetaEngine("/Users/krylon/guang.d/GeoLite2-City.mmdb"); err != nil {
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
