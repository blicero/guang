// /Users/krylon/go/src/guang/xfr.go
// -*- coding: utf-8; mode: go; -*-
// Created on 25. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-26 00:53:57 krylon>

package guang

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"

	//"github.com/miekg/dns"
	"github.com/tonnerre/golang-dns"
)

const (
	HOST_RE_PAT = "^[^.]+[.](.*)$"
)

var v6_addr_pat = regexp.MustCompile("[0-9a-f:]+")

type XFRClient struct {
	res           *dns.Client
	db            *HostDB
	request_queue chan string
	log           *log.Logger
	host_re       *regexp.Regexp
	name_bl       *NameBlacklist
	addr_bl       *IPBlacklist
}

func MakeXFRClient(queue chan string) (*XFRClient, error) {
	var err error
	var msg string
	var client *XFRClient = &XFRClient{
		request_queue: queue,
		host_re:       regexp.MustCompile(HOST_RE_PAT),
		name_bl:       DefaultNameBlacklist(),
		addr_bl:       DefaultIPBlacklist(),
		res:           new(dns.Client),
	}

	client.res.Net = "tcp"

	if client.log, err = GetLogger("XFRClient"); err != nil {
		fmt.Printf("Error getting Logger instance for XFRClient: %s\n", err.Error())
		return nil, err
	} else if client.db, err = OpenDB(DB_PATH); err != nil {
		msg = fmt.Sprintf("Error opening database at %s: %s",
			DB_PATH, err.Error())
		client.log.Println(msg)
		return nil, errors.New(msg)
	} else {
		return client, nil
	}
} // func MakeXFRClient(queue chan string) (*XFRClient, error)

func (self *XFRClient) perform_xfr(zone string) error {
	var err error
	var msg string
	var ns_records []*net.NS

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
		if res, err := self.attempt_xfr(zone, srv); err != nil {
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
func (self *XFRClient) attempt_xfr(zone string, srv net.IP) (bool, error) {
	var msg string
	var err error
	var rr_cnt int64
	var xfr_msg dns.Msg
	var env_chan chan *dns.Envelope
	var addr_list []string

	xfr_msg.SetAxfr(zone)

	self.log.Printf("Attempting AXFR of %s\n", zone)

	xfr_path := filepath.Join(XFR_DBG_PATH, zone)
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
		if envelope.Error != nil {
			msg = fmt.Sprintf("Error during AXFR of %s: %s",
				zone, envelope.Error.Error())
			self.log.Println(msg)
			continue
		}

	RR_LOOP:
		for _, rr := range envelope.RR {
			var host Host
			dbg_string := rr.String() + "\n"
			fh.WriteString(dbg_string)
			rr_cnt++

			switch t := rr.(type) {
			case *dns.A:
				host.Address = t.A
				host.Name = rr.Header().Name
				host.Source = HOST_SOURCE_A

				if self.name_bl.Matches(host.Name) || self.addr_bl.MatchesIP(host.Address) {
					continue RR_LOOP
				}

				if err = self.db.HostAdd(&host); err != nil {
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
						var ns_host Host = Host{Name: host.Name}

						ns_host.Address = net.ParseIP(addr)
						ns_host.Source = HOST_SOURCE_NS

						if self.addr_bl.MatchesIP(ns_host.Address) {
							continue ADDR_LOOP
						} else if err = self.db.HostAdd(&ns_host); err != nil {
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

				for _, addr := range addr_list {
					var mx_host Host = Host{
						Name:    host.Name,
						Address: net.ParseIP(addr),
						Source:  HOST_SOURCE_MX,
					}

					if err = self.db.HostAdd(&mx_host); err != nil {
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
				host.Source = HOST_SOURCE_A

				if self.name_bl.Matches(host.Name) || self.addr_bl.MatchesIP(host.Address) {
					continue RR_LOOP
				} else if err = self.db.HostAdd(&host); err != nil {
					msg = fmt.Sprintf("Error adding host %s/%s to database: %s",
						host.Name, host.Address.String(), err.Error())
					self.log.Println(msg)
				}
			}

		}
	}

	return true, nil
} // func (self *XFRClient) attempt_xfr(zone string, srv net.IP) (bool, error)
