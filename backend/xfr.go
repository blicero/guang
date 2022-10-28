// /Users/krylon/go/src/guang/xfr.go
// -*- coding: utf-8; mode: go; -*-
// Created on 25. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-28 23:12:18 krylon>

package backend

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/blicero/guang/database"
	"github.com/blicero/guang/generator"
	"github.com/blicero/krylib"

	//"github.com/miekg/dns"
	dns "github.com/tonnerre/golang-dns"
)

const (
	HOST_RE_PAT = "^[^.]+[.](.*)$"
)

// var v6_addr_pat = regexp.MustCompile("[0-9a-f:]+")

type XFRClient struct {
	res           *dns.Client
	request_queue chan string
	log           *log.Logger
	host_re       *regexp.Regexp
	name_bl       *generator.NameBlacklist
	addr_bl       *generator.IPBlacklist
	worker_cnt    int
	lock          sync.Mutex
}

func MakeXFRClient(queue chan string) (*XFRClient, error) {
	var err error
	var client *XFRClient = &XFRClient{
		request_queue: queue,
		host_re:       regexp.MustCompile(HOST_RE_PAT),
		name_bl:       generator.DefaultNameBlacklist(),
		addr_bl:       generator.DefaultIPBlacklist(),
		res:           new(dns.Client),
	}

	client.res.Net = "tcp"

	if client.log, err = common.GetLogger("XFRClient"); err != nil {
		fmt.Printf("Error getting Logger instance for XFRClient: %s\n", err.Error())
		return nil, err
	} else {
		return client, nil
	}
} // func MakeXFRClient(queue chan string) (*XFRClient, error)

func (self *XFRClient) Start(cnt int) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for i := 1; i <= cnt; i++ {
		if common.DEBUG {
			self.log.Printf("Starting XFR Worker #%d\n", i)
		}
		go self.Worker(i)
		self.worker_cnt++
	}
} // func (self *XFRClient) Start(cnt int)

func (self *XFRClient) Worker(worker_id int) {
	var hostname, zone, msg string
	var err error
	var xfr *data.XFR
	var submatch []string
	var db *database.HostDB

	defer func() {
		self.lock.Lock()
		self.worker_cnt--
		self.lock.Unlock()
	}()

	if db, err = database.OpenDB(common.DB_PATH); err != nil {
		msg = fmt.Sprintf("Error opening database at %s: %s",
			common.DB_PATH, err.Error())
		self.log.Println(msg)
		return
	} else {
		defer db.Close()
	}

LOOP:
	for {
		hostname = <-self.request_queue

		if common.DEBUG {
			self.log.Printf("XFR Worker #%d got request for host %s\n",
				worker_id, hostname)
		}

		if submatch = self.host_re.FindStringSubmatch(hostname); submatch == nil {
			msg = fmt.Sprintf("Error extracting zone from hostname %s", hostname)
			self.log.Println(msg)
			continue LOOP
		} else if len(submatch) == 0 {
			msg = fmt.Sprintf("CANTHAPPEN: Did not find zone in hostname: %s", hostname)
			self.log.Println(msg)
			continue LOOP
		} else if common.DEBUG {
			self.log.Printf("XFR#%d - extracted zone %s from hostname %s\n",
				worker_id, submatch[1], hostname)
		}

		zone = submatch[1]
		if xfr, err = db.XfrGetByZone(zone); err != nil {
			msg = fmt.Sprintf("Error looking up XFR of %s: %s",
				zone, err.Error())
			self.log.Println(msg)
			continue LOOP
		} else if xfr != nil {
			// Looks like we've been down that road before...
			continue LOOP
		}

		xfr = &data.XFR{
			ID:     krylib.INVALID_ID,
			Zone:   zone,
			Start:  time.Now(),
			Status: data.XFR_STATUS_UNFINISHED,
		}

		if err = db.XfrAdd(xfr); err != nil {
			msg = fmt.Sprintf("Error adding XFR of %s to database: %s",
				zone, err.Error())
			self.log.Println(msg)
			continue LOOP
		} else if xfr.ID == krylib.INVALID_ID {
			self.log.Printf("I added the XFR of %s to the database, but the ID was not set!\n",
				zone)
			continue LOOP
		}

		var status data.XfrStatus

		if err = self.perform_xfr(zone, db); err != nil {
			status = data.XFR_STATUS_REFUSED
		} else {
			status = data.XFR_STATUS_SUCCESS
		}

		if err = db.XfrFinish(xfr, status); err != nil {
			msg = fmt.Sprintf("Error finishing XFR of %s with status %s: %s",
				zone, status.String(), err.Error())
			self.log.Println(msg)
		}
	}
} // func (self *XFRClient) Worker()

func (self *XFRClient) perform_xfr(zone string, db *database.HostDB) error {
	var err error
	var msg string
	var ns_records []*net.NS
	var res bool

	// First, need to get the nameservers for the zone:
	if ns_records, err = net.LookupNS(zone); err != nil {
		msg = fmt.Sprintf("Error looking up nameservers for %s: %s",
			zone, err.Error())
		self.log.Println(msg)
		return errors.New(msg)
	}

	servers := make([]net.IP, 0)

	for _, srv := range ns_records {
		var addr []net.IP

		if addr, err = net.LookupIP(srv.Host); err != nil {
			msg = fmt.Sprintf("Error looking up %s: %s",
				srv.Host, err.Error())
			self.log.Println(msg)
		} else {
			servers = append(servers, addr...)
		}
	}

	if len(servers) == 0 {
		msg = fmt.Sprintf("Did not find any nameservers for %s", zone)
		self.log.Println(msg)
		return errors.New(msg)
	}

	for _, srv := range servers {
		if res, err = self.attempt_xfr(zone, srv, db); err != nil {
			msg = fmt.Sprintf("Error asking %s for XFR of %s: %s",
				srv.String(), zone, err.Error())
			self.log.Println(msg)
		} else if res {
			return nil
		}
	}

	msg = fmt.Sprintf("None of the %d servers I asked wanted to give me an XFR of %s: %s",
		len(servers), zone, err.Error())
	self.log.Println(msg)
	return errors.New(msg)
} // func (self *XFRClient) perform_xfr(zone string) error

// Samstag, 26. 12. 2015, 00:44
// Maybe I should factor this method into yet more sub-methods. It's rather long...
func (self *XFRClient) attempt_xfr(zone string, srv net.IP, db *database.HostDB) (bool, error) {
	var msg string
	var err error
	var rr_cnt int64
	var xfr_msg dns.Msg
	var env_chan chan *dns.Envelope
	var addr_list []string
	var xfr_error bool

	xfr_msg.SetAxfr(zone)

	self.log.Printf("Attempting AXFR of %s\n", zone)

	xfr_path := filepath.Join(common.XFR_DBG_PATH, zone)
	fh, err := os.Create(xfr_path)
	if err != nil {
		msg = fmt.Sprintf("Error opening dbg file for XFR (%s): %s",
			xfr_path, err.Error())
		self.log.Println(msg)
		return false, errors.New(msg)
	} else {
		cleanup := func() {
			fh.Close()
			if rr_cnt == 0 {
				os.Remove(xfr_path)
			}
		}

		defer cleanup()
	}

	ns := fmt.Sprintf("[%s]:53", srv.String())

	if env_chan, err = self.res.TransferIn(&xfr_msg, ns); err != nil {
		msg = fmt.Sprintf("Error requesting Transfer of zone %s: %s",
			zone, err.Error())
		self.log.Println(msg)
		return false, errors.New(msg)
	}

	for envelope := range env_chan {
		xfr_error = false
		if envelope.Error != nil {
			err = envelope.Error
			xfr_error = true
			msg = fmt.Sprintf("Error during AXFR of %s: %s",
				zone, envelope.Error.Error())
			self.log.Println(msg)
			continue
		}

		var host_exists bool

	RR_LOOP:
		for _, rr := range envelope.RR {
			var host data.Host
			dbg_string := rr.String() + "\n"
			fh.WriteString(dbg_string) // nolint: errcheck
			rr_cnt++

			switch t := rr.(type) {
			case *dns.A:
				host.Address = t.A
				host.Name = rr.Header().Name
				host.Source = data.HOST_SOURCE_A

				if self.name_bl.Matches(host.Name) || self.addr_bl.MatchesIP(host.Address) {
					continue RR_LOOP
				}

				if host_exists, err = db.HostExists(host.Address.String()); err != nil {
					msg = fmt.Sprintf("Error checking if %s is already in database: %s",
						host.Address.String(), err.Error())
					self.log.Println(msg)
				} else if host_exists {
					continue RR_LOOP
				} else if err = db.HostAdd(&host); err != nil {
					msg = fmt.Sprintf("Error adding host %s/%s to database: %s",
						host.Address.String(), host.Name, err.Error())
					self.log.Println(msg)
				}

			case *dns.NS:
				host.Name = rr.Header().Name
				if self.name_bl.Matches(host.Name) {
					continue RR_LOOP
				}

				if addr_list, err = net.LookupHost(host.Name); err != nil {
					msg = fmt.Sprintf("Error looking up name for Nameserver %s: %s",
						host.Name, err.Error())
					self.log.Println(msg)
					continue RR_LOOP
				} else {
				ADDR_LOOP:
					for _, addr := range addr_list {
						var ns_host data.Host = data.Host{Name: host.Name}

						ns_host.Address = net.ParseIP(addr)
						ns_host.Source = data.HOST_SOURCE_NS

						if self.addr_bl.MatchesIP(ns_host.Address) {
							continue ADDR_LOOP
						} else if host_exists, err = db.HostExists(ns_host.Address.String()); err != nil {
							msg = fmt.Sprintf("Error checking if %s is already in database: %s",
								ns_host.Address.String(), err.Error())
							self.log.Println(msg)
						} else if host_exists {
							continue ADDR_LOOP
						} else if err = db.HostAdd(&ns_host); err != nil {
							msg = fmt.Sprintf("Error adding Nameserver %s to database: %s",
								ns_host.Name, err.Error())
							self.log.Println(msg)
						}
					}
				}

			case *dns.MX:
				host.Name = rr.Header().Name
				if self.name_bl.Matches(host.Name) {
					continue RR_LOOP
				} else if addr_list, err = net.LookupHost(host.Name); err != nil {
					msg = fmt.Sprintf("Error looking up IP Address for %s: %s",
						host.Name, err.Error())
					self.log.Println(msg)
					continue RR_LOOP
				}

			MX_HOST:
				for _, addr := range addr_list {
					var mx_host data.Host = data.Host{
						Name:    host.Name,
						Address: net.ParseIP(addr),
						Source:  data.HOST_SOURCE_MX,
					}

					if host_exists, err = db.HostExists(mx_host.Address.String()); err != nil {
						msg = fmt.Sprintf("Error checking if %s is already in database: %s",
							mx_host.Address.String(), err.Error())
						self.log.Println(msg)
					} else if host_exists {
						continue MX_HOST
					} else if err = db.HostAdd(&mx_host); err != nil {
						msg = fmt.Sprintf("Error adding MX %s/%s to database: %s",
							mx_host.Name,
							mx_host.Address.String(),
							err.Error())
						self.log.Println(msg)
					}
				}

			case *dns.AAAA:
				host.Name = rr.Header().Name
				host.Address = t.AAAA
				host.Source = data.HOST_SOURCE_A

				if self.name_bl.Matches(host.Name) || self.addr_bl.MatchesIP(host.Address) {
					continue RR_LOOP
				} else if host_exists, err = db.HostExists(host.Address.String()); err != nil {
					msg = fmt.Sprintf("Error checking if %s exists in database: %s",
						host.Address.String(), err.Error())
					self.log.Println(msg)
				} else if host_exists {
					continue RR_LOOP
				} else if err = db.HostAdd(&host); err != nil {
					msg = fmt.Sprintf("Error adding host %s/%s to database: %s",
						host.Name, host.Address.String(), err.Error())
					self.log.Println(msg)
				}
			}

		}
	}

	if xfr_error {
		return false, err
	}

	return true, nil
} // func (self *XFRClient) attempt_xfr(zone string, srv net.IP, db *HostDB) (bool, error)
