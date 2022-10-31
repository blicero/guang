// /Users/krylon/go/src/guang/backend/metadata.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 08. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2022-10-31 19:02:38 krylon>
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
	"log"
	"path/filepath"
	"regexp"

	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/blicero/krylib"

	"github.com/oschwald/geoip2-golang"
)

const (
	geoIPCityPath    = "GeoLite2-City.mmdb"
	geoIPCountryPath = "GeoLite2-Country.mmdb"
)

var osPatterns map[string][]*regexp.Regexp = map[string][]*regexp.Regexp{
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
		regexp.MustCompile(`(?i)rhel\d`),
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

// MetaEngine processes metadata on Hosts.
type MetaEngine struct {
	geodb *geoip2.Reader
	log   *log.Logger
} // type MetaEngine struct

// OpenMetaEngine creates a new MetaEngine.
func OpenMetaEngine(path string) (*MetaEngine, error) {
	var eng *MetaEngine = new(MetaEngine)
	var err error
	var msg, dbPath string

	if ex, _ := krylib.Fexists(path); ex {
		dbPath = path
	} else {
		dbPath = filepath.Join(common.BaseDir, geoIPCityPath)
	}

	if eng.log, err = common.GetLogger("MetaEngine"); err != nil {
		return nil, err
	} else if eng.geodb, err = geoip2.Open(dbPath); err != nil {
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

// Close closes the MetaEngine.
func (m *MetaEngine) Close() {
	m.geodb.Close()
} // func (m *MetaEngine) Close()

// LookupCountry attempts to determine what county a Host is located in.
func (m *MetaEngine) LookupCountry(h *data.Host) (string, error) {
	var err error
	var country *geoip2.Country

	if country, err = m.geodb.Country(h.Address); err != nil {
		return "", err
	}

	return country.Country.Names["de"], nil
} // func (m *MetaEngine) LookupCountry(h *Host) (string, error)

// LookupCity attempts to determine what city a Host is located in.
func (m *MetaEngine) LookupCity(h *data.Host) (string, error) {
	var err error
	var city *geoip2.City

	if city, err = m.geodb.City(h.Address); err != nil {
		return "", err
	}

	return city.City.Names["de"], nil
} // func (m *MetaEngine) LookupCity(h *Host) (string, error)

// LookupOperatingSystem attempts to determine what OS a Host is running.
func (m *MetaEngine) LookupOperatingSystem(h *data.HostWithPorts) string {
	var results map[string]int = make(map[string]int)

PORT:
	for _, port := range h.Ports {
		for os, patterns := range osPatterns {
			for _, pattern := range patterns {
				if port.Reply != nil && pattern.MatchString(*port.Reply) {
					results[os]++
					continue PORT
				}
			}
		}
	}

	var (
		os     = "Unknown"
		hitCnt int
	)

	for system, cnt := range results {
		if cnt > hitCnt {
			os = system
		}
	}

	return os
} // func (m *MetaEngine) LookupOperatingSystem(h *HostWithPorts) string
