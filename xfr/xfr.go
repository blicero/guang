// /Users/krylon/go/src/guang/xfr.go
// -*- coding: utf-8; mode: go; -*-
// Created on 25. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-11-09 23:08:14 krylon>

package xfr

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

	"github.com/blicero/guang/blacklist"
	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/blicero/guang/database"
	"github.com/blicero/guang/xfr/xfrstatus"
	"github.com/blicero/krylib"

	//"github.com/miekg/dns"
	dns "github.com/tonnerre/golang-dns"
)

const (
	hostRePat = "^[^.]+[.](.*)$"
)

// var v6_addr_pat = regexp.MustCompile("[0-9a-f:]+")

// Client performs DNS zone transfers.
type Client struct {
	res          *dns.Client
	requestQueue chan string
	RC           chan data.ControlMessage
	log          *log.Logger
	hostRe       *regexp.Regexp
	nameBL       *blacklist.NameBlacklist
	addrBL       *blacklist.IPBlacklist
	workerCnt    int
	lock         sync.RWMutex
	isRunning    bool
}

// MakeXFRClient creates a new XFRClient
func MakeXFRClient(queue chan string) (*Client, error) {
	var err error
	var client *Client = &Client{
		requestQueue: queue,
		RC:           make(chan data.ControlMessage, 4),
		hostRe:       regexp.MustCompile(hostRePat),
		nameBL:       blacklist.DefaultNameBlacklist(),
		addrBL:       blacklist.DefaultIPBlacklist(),
		res:          new(dns.Client),
	}

	client.res.Net = "tcp"

	if client.log, err = common.GetLogger("XFRClient"); err != nil {
		fmt.Printf("Error getting Logger instance for XFRClient: %s\n", err.Error())
		return nil, err
	}

	return client, nil
} // func MakeXFRClient(queue chan string) (*XFRClient, error)

// Start starts the XFRCient
func (xfrc *Client) Start(cnt int) {
	xfrc.lock.Lock()
	defer xfrc.lock.Unlock()

	xfrc.isRunning = true

	for i := 1; i <= cnt; i++ {
		if common.Debug {
			xfrc.log.Printf("Starting XFR Worker #%d\n", i)
		}
		go xfrc.worker(i)
		//xfrc.workerCnt++
	}
} // func (xfrc *XFRClient) Start(cnt int)

func (xfrc *Client) Stop() {
	xfrc.lock.Lock()
	xfrc.isRunning = false
	xfrc.lock.Unlock()
} // func (xfrc *Client) Stop()

func (xfrc *Client) IsRunning() bool {
	xfrc.lock.RLock()
	var r = xfrc.isRunning
	xfrc.lock.RUnlock()
	return r
} // func (xfrc *Client) IsRunning() bool

// Count returns the number of workers.
func (xfrc *Client) Count() int {
	xfrc.lock.RLock()
	var c = xfrc.workerCnt
	xfrc.lock.RUnlock()
	return c
} // func (xfrc *XFRClient) WorkerCount() int

func (xfrc *Client) cntInc() {
	xfrc.lock.Lock()
	xfrc.workerCnt++
	xfrc.lock.Unlock()
} // func (xfrc *HostXfrcerator) cntInc()

func (xfrc *Client) cntDec() {
	xfrc.lock.Lock()
	xfrc.workerCnt--
	xfrc.lock.Unlock()
} // func (xfrc *Client) cntDec()

func (xfrc *Client) worker(workerID int) {
	var (
		hostname, zone, msg string
		err                 error
		xfr                 *data.XFR
		submatch            []string
		db                  *database.HostDB
		pulse               *time.Ticker
	)

	xfrc.cntInc()
	defer xfrc.cntDec()

	if db, err = database.OpenDB(common.DbPath); err != nil {
		msg = fmt.Sprintf("Error opening database at %s: %s",
			common.DbPath, err.Error())
		xfrc.log.Println(msg)
		return
	}

	defer db.Close()

	pulse = time.NewTicker(common.RCTimeout)
	defer pulse.Stop()

LOOP:
	for xfrc.IsRunning() {
		select {
		case hostname = <-xfrc.requestQueue:
			// Alrighty, then
		case ctl := <-xfrc.RC:
			switch ctl {
			case data.CtlMsgStop:
				return
			case data.CtlMsgShutdown:
				xfrc.Stop()
				return
			case data.CtlMsgSpawn:
				var cnt = xfrc.Count()
				go xfrc.worker(cnt + 1)
			}
			continue
		case <-pulse.C:
			continue
		}

		if common.Debug {
			xfrc.log.Printf("XFR Worker #%d got request for host %s\n",
				workerID, hostname)
		}

		if submatch = xfrc.hostRe.FindStringSubmatch(hostname); submatch == nil {
			msg = fmt.Sprintf("Error extracting zone from hostname %s", hostname)
			xfrc.log.Println(msg)
			continue LOOP
		} else if len(submatch) == 0 {
			msg = fmt.Sprintf("CANTHAPPEN: Did not find zone in hostname: %s", hostname)
			xfrc.log.Println(msg)
			continue LOOP
		} else if common.Debug {
			xfrc.log.Printf("XFR#%d - extracted zone %s from hostname %s\n",
				workerID, submatch[1], hostname)
		}

		zone = submatch[1]
		if xfr, err = db.XfrGetByZone(zone); err != nil {
			msg = fmt.Sprintf("Error looking up XFR of %s: %s",
				zone, err.Error())
			xfrc.log.Println(msg)
			continue LOOP
		} else if xfr != nil {
			// Looks like we've been down that road before...
			continue LOOP
		}

		xfr = &data.XFR{
			ID:     krylib.INVALID_ID,
			Zone:   zone,
			Start:  time.Now(),
			Status: xfrstatus.Unfinished,
		}

		if err = db.XfrAdd(xfr); err != nil {
			msg = fmt.Sprintf("Error adding XFR of %s to database: %s",
				zone, err.Error())
			xfrc.log.Println(msg)
			continue LOOP
		} else if xfr.ID == krylib.INVALID_ID {
			xfrc.log.Printf("I added the XFR of %s to the database, but the ID was not set!\n",
				zone)
			continue LOOP
		}

		var status xfrstatus.XfrStatus

		if err = xfrc.performXfr(zone, db); err != nil {
			status = xfrstatus.Refused
		} else {
			status = xfrstatus.Success
		}

		if err = db.XfrFinish(xfr, status); err != nil {
			msg = fmt.Sprintf("Error finishing XFR of %s with status %s: %s",
				zone, status.String(), err.Error())
			xfrc.log.Println(msg)
		}
	}
} // func (xfrc *XFRClient) worker()

func (xfrc *Client) performXfr(zone string, db *database.HostDB) error {
	var err error
	var msg string
	var nsRecords []*net.NS
	var res bool

	// First, need to get the nameservers for the zone:
	if nsRecords, err = net.LookupNS(zone); err != nil {
		msg = fmt.Sprintf("Error looking up nameservers for %s: %s",
			zone, err.Error())
		xfrc.log.Println(msg)
		return errors.New(msg)
	}

	servers := make([]net.IP, 0)

	for _, srv := range nsRecords {
		var addr []net.IP

		if addr, err = net.LookupIP(srv.Host); err != nil {
			msg = fmt.Sprintf("Error looking up %s: %s",
				srv.Host, err.Error())
			xfrc.log.Println(msg)
		} else {
			servers = append(servers, addr...)
		}
	}

	if len(servers) == 0 {
		msg = fmt.Sprintf("Did not find any nameservers for %s", zone)
		xfrc.log.Println(msg)
		return errors.New(msg)
	}

	for _, srv := range servers {
		if res, err = xfrc.attemptXfr(zone, srv, db); err != nil {
			msg = fmt.Sprintf("Error asking %s for XFR of %s: %s",
				srv.String(), zone, err.Error())
			xfrc.log.Println(msg)
		} else if res {
			return nil
		}
	}

	msg = fmt.Sprintf("None of the %d servers I asked wanted to give me an XFR of %s: %s",
		len(servers), zone, err.Error())
	xfrc.log.Println(msg)
	return errors.New(msg)
} // func (xfrc *XFRClient) performXfr(zone string) error

// Samstag, 26. 12. 2015, 00:44
// Maybe I should factor this method into yet more sub-methods. It's rather long...
func (xfrc *Client) attemptXfr(zone string, srv net.IP, db *database.HostDB) (bool, error) {
	var msg string
	var err error
	var rrCnt int64
	var xfrMsg dns.Msg
	var envChan chan *dns.Envelope
	var addrList []string
	var xfrError bool

	xfrMsg.SetAxfr(zone)

	xfrc.log.Printf("Attempting AXFR of %s\n", zone)

	xfrPath := filepath.Join(common.XfrDbgPath, zone)
	fh, err := os.Create(xfrPath)
	if err != nil {
		msg = fmt.Sprintf("Error opening dbg file for XFR (%s): %s",
			xfrPath, err.Error())
		xfrc.log.Println(msg)
		return false, errors.New(msg)
	}

	defer func() {
		fh.Close()
		if rrCnt == 0 {
			os.Remove(xfrPath)
		}
	}()

	ns := fmt.Sprintf("[%s]:53", srv.String())

	if envChan, err = xfrc.res.TransferIn(&xfrMsg, ns); err != nil {
		msg = fmt.Sprintf("Error requesting Transfer of zone %s: %s",
			zone, err.Error())
		xfrc.log.Println(msg)
		return false, errors.New(msg)
	}

	for envelope := range envChan {
		xfrError = false
		if envelope.Error != nil {
			err = envelope.Error
			xfrError = true
			msg = fmt.Sprintf("Error during AXFR of %s: %s",
				zone, envelope.Error.Error())
			xfrc.log.Println(msg)
			continue
		}

		var hostExists bool

	RR_LOOP:
		for _, rr := range envelope.RR {
			var host data.Host
			dbgString := rr.String() + "\n"
			fh.WriteString(dbgString) // nolint: errcheck
			rrCnt++

			switch t := rr.(type) {
			case *dns.A:
				host.Address = t.A
				host.Name = rr.Header().Name
				host.Source = data.HostSourceA

				if xfrc.nameBL.Matches(host.Name) || xfrc.addrBL.MatchesIP(host.Address) {
					continue RR_LOOP
				}

				if hostExists, err = db.HostExists(host.Address.String()); err != nil {
					msg = fmt.Sprintf("Error checking if %s is already in database: %s",
						host.Address.String(), err.Error())
					xfrc.log.Println(msg)
				} else if hostExists {
					continue RR_LOOP
				} else if err = db.HostAdd(&host); err != nil {
					msg = fmt.Sprintf("Error adding host %s/%s to database: %s",
						host.Address.String(), host.Name, err.Error())
					xfrc.log.Println(msg)
				}

			case *dns.NS:
				host.Name = rr.Header().Name
				if xfrc.nameBL.Matches(host.Name) {
					continue RR_LOOP
				}

				if addrList, err = net.LookupHost(host.Name); err != nil {
					msg = fmt.Sprintf("Error looking up name for Nameserver %s: %s",
						host.Name, err.Error())
					xfrc.log.Println(msg)
					continue RR_LOOP
				} else {
				ADDR_LOOP:
					for _, addr := range addrList {
						var nsHost data.Host = data.Host{Name: host.Name}

						nsHost.Address = net.ParseIP(addr)
						nsHost.Source = data.HostSourceNs

						if xfrc.addrBL.MatchesIP(nsHost.Address) {
							continue ADDR_LOOP
						} else if hostExists, err = db.HostExists(nsHost.Address.String()); err != nil {
							msg = fmt.Sprintf("Error checking if %s is already in database: %s",
								nsHost.Address.String(), err.Error())
							xfrc.log.Println(msg)
						} else if hostExists {
							continue ADDR_LOOP
						} else if err = db.HostAdd(&nsHost); err != nil {
							msg = fmt.Sprintf("Error adding Nameserver %s to database: %s",
								nsHost.Name, err.Error())
							xfrc.log.Println(msg)
						}
					}
				}

			case *dns.MX:
				host.Name = rr.Header().Name
				if xfrc.nameBL.Matches(host.Name) {
					continue RR_LOOP
				} else if addrList, err = net.LookupHost(host.Name); err != nil {
					msg = fmt.Sprintf("Error looking up IP Address for %s: %s",
						host.Name, err.Error())
					xfrc.log.Println(msg)
					continue RR_LOOP
				}

			MX_HOST:
				for _, addr := range addrList {
					var mxHost data.Host = data.Host{
						Name:    host.Name,
						Address: net.ParseIP(addr),
						Source:  data.HostSourceMx,
					}

					if hostExists, err = db.HostExists(mxHost.Address.String()); err != nil {
						msg = fmt.Sprintf("Error checking if %s is already in database: %s",
							mxHost.Address.String(), err.Error())
						xfrc.log.Println(msg)
					} else if hostExists {
						continue MX_HOST
					} else if err = db.HostAdd(&mxHost); err != nil {
						msg = fmt.Sprintf("Error adding MX %s/%s to database: %s",
							mxHost.Name,
							mxHost.Address.String(),
							err.Error())
						xfrc.log.Println(msg)
					}
				}

			case *dns.AAAA:
				host.Name = rr.Header().Name
				host.Address = t.AAAA
				host.Source = data.HostSourceA

				if xfrc.nameBL.Matches(host.Name) || xfrc.addrBL.MatchesIP(host.Address) {
					continue RR_LOOP
				} else if hostExists, err = db.HostExists(host.Address.String()); err != nil {
					msg = fmt.Sprintf("Error checking if %s exists in database: %s",
						host.Address.String(), err.Error())
					xfrc.log.Println(msg)
				} else if hostExists {
					continue RR_LOOP
				} else if err = db.HostAdd(&host); err != nil {
					msg = fmt.Sprintf("Error adding host %s/%s to database: %s",
						host.Name, host.Address.String(), err.Error())
					xfrc.log.Println(msg)
				}
			}

		}
	}

	if xfrError {
		return false, err
	}

	return true, nil
} // func (xfrc *XFRClient) attemptXfr(zone string, srv net.IP, db *HostDB) (bool, error)
