// /Users/krylon/go/src/guang/backend/metadata.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 08. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2016-08-21 18:58:25 krylon>
//
// Sonntag, 21. 08. 2016, 18:25
// Looking up locations seems to work reasonably well. Whether or not the
// results are reliable is out of my control anyway.
// So, next I want to be able to guesstimate the operating system
// a host is running.

package backend

import (
	"errors"
	"fmt"
	"krylib"
	"log"
	"path/filepath"
	"regexp"

	"github.com/oschwald/geoip2-golang"
)

const (
	GEOIP_CITY_PATH    = "GeoLite2-City.mmdb"
	GEOIP_COUNTRY_PATH = "GeoLite2-Country.mmdb"
)

var os_pattern_map map[string][]*regexp.Regexp = map[string][]*regexp.Regexp{
	"Windows": []*regexp.Regexp{
		regexp.MustCompile("Microsoft"),
		regexp.MustCompile("Windows"),
	},
	"Debian": []*regexp.Regexp{
		regexp.MustCompile("(?i)Debian"),
		regexp.MustCompile("(?i)[.]deb"),
	},
	"Ubuntu": []*regexp.Regexp{
		regexp.MustCompile("(?i)ubuntu"),
	},
	"CentOS": []*regexp.Regexp{
		regexp.MustCompile("CentOS"),
	},
	"Red Hat Enterprise Linux": []*regexp.Regexp{
		regexp.MustCompile("(?i)rhel\\d"),
		regexp.MustCompile("(?i)Red ?Hat"),
	},
	"FreeBSD": []*regexp.Regexp{
		regexp.MustCompile("(?i)FreeBSD"),
	},
	"OpenBSD": []*regexp.Regexp{
		regexp.MustCompile("(?i)OpenBSD"),
	},
	"DragonflyBSD": []*regexp.Regexp{
		regexp.MustCompile("Dragonfly"),
	},
	"NetBSD": []*regexp.Regexp{
		regexp.MustCompile("(?i)netbsd"),
	},
	"Linux": []*regexp.Regexp{
		regexp.MustCompile("(?i)Linux"),
	},
}

type MetaEngine struct {
	geodb *geoip2.Reader
	log   *log.Logger
} // type MetaEngine struct

func OpenMetaEngine(path string) (*MetaEngine, error) {
	var eng *MetaEngine = new(MetaEngine)
	var err error
	var msg, db_path string

	if ex, _ := krylib.Fexists(path); ex {
		db_path = path
	} else {
		db_path = filepath.Join(BASE_DIR, GEOIP_CITY_PATH)
	}

	if eng.log, err = GetLogger("MetaEngine"); err != nil {
		return nil, err
	} else if eng.geodb, err = geoip2.Open(db_path); err != nil {
		msg = fmt.Sprintf("Error opening GeoIP database: %s", err.Error())
		eng.log.Println(msg)
		return nil, errors.New(msg)
	} else if eng.geodb == nil {
		msg = "Opening GeoIP database did not return an error, but the geoip2.Reader was nil!"
		eng.log.Println(msg)
		return nil, errors.New(msg)
	} else {
		return eng, nil
	}
} // func OpenMetaEngine() (*MetaEngine, error)

func (self *MetaEngine) Close() {
	self.geodb.Close()
} // func (self *MetaEngine) Close()

func (self *MetaEngine) LookupCountry(h *Host) (string, error) {
	var err error
	var country *geoip2.Country

	if country, err = self.geodb.Country(h.Address); err != nil {
		return "", err
	} else {
		return country.Country.Names["de"], nil
	}
} // func (self *MetaEngine) LookupCountry(h *Host) (string, error)

func (self *MetaEngine) LookupCity(h *Host) (string, error) {
	var err error
	var city *geoip2.City

	if city, err = self.geodb.City(h.Address); err != nil {
		return "", err
	} else {
		return city.City.Names["de"], nil
	}
} // func (self *MetaEngine) LookupCity(h *Host) (string, error)

func (self *MetaEngine) LookupOperatingSystem(h *HostWithPorts) string {
	var result_map map[string]int = make(map[string]int)

PORT:
	for _, port := range h.Ports {
		for os, patterns := range os_pattern_map {
			for _, pattern := range patterns {
				if port.Reply != nil && pattern.MatchString(*port.Reply) {
					result_map[os]++
					continue PORT
				}
			}
		}
	}

	var os string = "Unknown"
	var hit_cnt int

	for system, cnt := range result_map {
		if cnt > hit_cnt {
			os = system
		}
	}

	return os
} // func (self *MetaEngine) LookupOperatingSystem(h *HostWithPorts) string
