// /Users/krylon/go/src/guang/backend/metadata.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 08. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2024-05-30 00:03:13 krylon>
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
	"github.com/blicero/guang/database"

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
	"JUNOS",
	"Cisco IOS",
	"SonicOS",
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
	"JUNOS": {
		regexp.MustCompile("(?i:JUNOS|Juniper)"),
	},
	"Cisco IOS": {
		regexp.MustCompile("(?i)Cisco IOS Software"),
		regexp.MustCompile("(?i)Cisco Systems"),
	},
	"SonicOS": {
		regexp.MustCompile("(?i)SonicOS"),
		regexp.MustCompile("(?i)SonicWALL"),
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
			hitCnt = cnt
		}
	}

	return os
} // func (m *MetaEngine) LookupOperatingSystem(h *HostWithPorts) string

// UpdateMetadata refreshes the location and OS metadata for all hosts.
func (m *MetaEngine) UpdateMetadata() error {
	var (
		err   error
		db    *database.HostDB
		hosts []data.Host
	)

	if db, err = database.OpenDB(common.DbPath); err != nil {
		m.log.Printf("[ERROR] Cannot open HostDB at %s: %s\n",
			common.DbPath,
			err.Error())
		return err
	}

	defer db.Close() // nolint: errcheck

	if hosts, err = db.HostGetAll(); err != nil {
		m.log.Printf("[ERROR] Cannot get all hosts: %s\n",
			err.Error())
		return err
	}

	for _, host := range hosts {
		var (
			city, country, location, os string
			hwp                         = data.HostWithPorts{Host: host}
		)

		if city, err = m.LookupCity(&host); err != nil {
			m.log.Printf("[ERROR] Cannot lookup city for %s: %s\n",
				host.Address,
				err.Error())
			city = ""
		} else if country, err = m.LookupCountry(&host); err != nil {
			m.log.Printf("[ERROR] Cannot lookup country for %s: %s\n",
				host.Address, err.Error())
			goto LOOKUP_OS
		}

		if city != "" {
			location = fmt.Sprintf("%s, %s",
				city, country)
		} else {
			location = country
		}

		if location == "" {
			goto LOOKUP_OS
		} else if err = db.HostSetLocation(&host, location); err != nil {
			m.log.Printf("[ERROR] Cannot set Location for %s to %q: %s\n",
				host.Address,
				location,
				err.Error())
		}

	LOOKUP_OS:
		if hwp.Ports, err = db.PortGetByHost(host.ID); err != nil {
			m.log.Printf("[ERROR] Failed to get scanned ports for %s: %s\n",
				host.Address,
				err.Error())
			continue
		} else if len(hwp.Ports) == 0 {
			continue
		}

		os = m.LookupOperatingSystem(&hwp)

		if err = db.HostSetOS(&host, os); err != nil {
			m.log.Printf("[ERROR] Failed to set OS on host %s to %s: %s\n",
				host.Address,
				os,
				err.Error())
		}
	}

	return nil
} // func (m *MetaEngine) UpdateMetadata() error
