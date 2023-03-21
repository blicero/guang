// /Users/krylon/go/src/guang/backend/metadata.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 08. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2023-03-21 01:17:14 krylon>
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

	"github.com/oschwald/geoip2-golang"
)

const (
	geoIPCityPath    = "GeoLite2-City.mmdb"
	geoIPCountryPath = "GeoLite2-Country.mmdb"
)

var osList = []string{
	"Windows",
	"Ubuntu",
	"Debian",
	"CentOS",
	"Red Hat",
	"Fedora",
	"Yocto",
	"FreeBSD",
	"NetBSD",
	"OpenBSD",
	"DragonflyBSD",
	"RouterOS",
	"Linux",
}

var osPatterns = map[string][]*regexp.Regexp{
	"Windows": {
		regexp.MustCompile("Microsoft"),
		regexp.MustCompile("Windows"),
	},
	"Debian": {
		regexp.MustCompile("(?i)Debian"),
		regexp.MustCompile("(?i)[.]deb"),
	},
	"Ubuntu": {
		regexp.MustCompile("(?i)ubuntu"),
	},
	"CentOS": {
		regexp.MustCompile("(?i)CentOS"),
	},
	"Red Hat": {
		regexp.MustCompile(`(?i)rhel\d+`),
		regexp.MustCompile("(?i)Red ?Hat"),
		regexp.MustCompile(`(?i)[.]el\d+[.]`),
	},
	"Fedora": {
		regexp.MustCompile("(?i)fedora"),
	},
	"Yocto Linux": {
		regexp.MustCompile("(?i)yocto"),
	},
	"FreeBSD": {
		regexp.MustCompile("(?i)FreeBSD"),
	},
	"OpenBSD": {
		regexp.MustCompile("(?i)OpenBSD"),
	},
	"DragonflyBSD": {
		regexp.MustCompile("Dragonfly"),
	},
	"NetBSD": {
		regexp.MustCompile("(?i)NetBSD"),
	},
	"RouterOS": {
		regexp.MustCompile("(?i)RouterOS"),
	},
	"Linux": {
		regexp.MustCompile("(?i)Linux"),
	},
}

// MetaEngine processes metadata on Hosts.
type MetaEngine struct {
	citydb    *geoip2.Reader
	countrydb *geoip2.Reader
	log       *log.Logger
} // type MetaEngine struct

// OpenMetaEngine creates a new MetaEngine.
func OpenMetaEngine(prefix string) (*MetaEngine, error) {
	var eng *MetaEngine = new(MetaEngine)
	var err error
	var msg, countrydbPath, citydbPath string

	// if ex, _ := krylib.Fexists(prefix); ex {
	// 	countrydbPath = prefix
	// } else {
	// 	countrydbPath = filepath.Join(common.BaseDir, geoIPCityPath)
	// }

	countrydbPath = filepath.Join(common.BaseDir, geoIPCountryPath)
	citydbPath = filepath.Join(common.BaseDir, geoIPCityPath)

	if eng.log, err = common.GetLogger("MetaEngine"); err != nil {
		return nil, err
	} else if eng.countrydb, err = geoip2.Open(countrydbPath); err != nil {
		msg = fmt.Sprintf("Error opening GeoIP database %s: %s",
			countrydbPath,
			err.Error())
		eng.log.Println(msg)
		return nil, errors.New(msg)
	} else if eng.countrydb == nil {
		msg = "Opening GeoIP database did not return an error, but the geoip2.Reader was nil!"
		eng.log.Println(msg)
		return nil, errors.New(msg)
	} else if eng.citydb, err = geoip2.Open(citydbPath); err != nil {
		msg = fmt.Sprintf("Cannot open GeoIP database %s: %s",
			citydbPath,
			err.Error())
		eng.log.Printf("[ERROR] %s\n", msg)
		return nil, errors.New(msg)
	} else {
		return eng, nil
	}
} // func OpenMetaEngine() (*MetaEngine, error)

// Close closes the MetaEngine.
func (m *MetaEngine) Close() {
	m.countrydb.Close()
} // func (m *MetaEngine) Close()

// LookupCountry attempts to determine what county a Host is located in.
func (m *MetaEngine) LookupCountry(h *data.Host) (string, error) {
	var err error
	var country *geoip2.Country

	if country, err = m.countrydb.Country(h.Address); err != nil {
		return "", err
	}

	return country.Country.Names["de"], nil
} // func (m *MetaEngine) LookupCountry(h *Host) (string, error)

// LookupCity attempts to determine what city a Host is located in.
func (m *MetaEngine) LookupCity(h *data.Host) (string, error) {
	var err error
	var city *geoip2.City

	if city, err = m.citydb.City(h.Address); err != nil {
		return "", err
	}

	return city.City.Names["de"], nil
} // func (m *MetaEngine) LookupCity(h *Host) (string, error)

// LookupOperatingSystem attempts to determine what OS a Host is running.
func (m *MetaEngine) LookupOperatingSystem(h *data.HostWithPorts) string {
	var results map[string]int = make(map[string]int)

PORT:
	for _, port := range h.Ports {
		//for os, patterns := range osPatterns {
		for _, osname := range osList {
			patterns := osPatterns[osname]
			for _, pattern := range patterns {
				if port.Reply != nil && pattern.MatchString(*port.Reply) {
					results[osname]++
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
