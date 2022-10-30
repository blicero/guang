// /Users/krylon/go/src/guang/scanner.go
// -*- coding: utf-8; mode: go; -*-
// Created on 28. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-30 21:02:51 krylon>
//
// Freitag, 08. 01. 2016, 22:10
// I kinda feel like I'm not going to write a comprehensive test suite for this
// one.  Instead, I'm going make it really verbose.

package backend

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/alouca/gosnmp"
	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/blicero/guang/database"
	"github.com/miekg/dns"
)

var www_pat *regexp.Regexp = regexp.MustCompile("(?i)^www")
var ftp_pat *regexp.Regexp = regexp.MustCompile("(?i)^ftp")
var mx_pat *regexp.Regexp = regexp.MustCompile("(?i)^(?:mx|mail|smtp|pop|imap)")
var newline = regexp.MustCompile("[\r\n]+$")

// Samstag, 05. 07. 2014, 16:40
// Da muss ich später noch mal schauen, wie weit ich die Liste erweitern kann,
// erst mal will ich das Gerüst stehen haben.
//
// Montag, 29. 08. 2016, 18:07
// Ich habe guang eine Weile auf meinem Server bei Digitalocean laufen lassen
// und irgendwann eine Mail bekommen, dass denen wiederum jemand geschrieben hat,
// dass auf deren Firewall Alarm geschlagen wurde.
// Ich nehme die Ports 1024 und 4444 mal vorsichtshalber raus. Nicht, dass
// wir noch Ärger bekommen.
var PORTS []uint16 = []uint16{21, 22, 23, 25, 53, 79, 80, 110, 143, 161, 631 /* 1024, 4444, */, 2525, 5353, 5800, 5900, 8000, 8080, 8081}

func get_scan_port(host *data.Host, ports map[uint16]bool) uint16 {
	if host.Source == data.HostSourceMx {
		if !ports[25] {
			return 25
		} else if !ports[110] {
			return 110
		} else if !ports[143] {
			return 143
		}
	} else if (host.Source == data.HostSourceNs) && !ports[53] {
		return 53
	} else if www_pat.MatchString(host.Name) && !ports[80] {
		// Samstag, 05. 07. 2014, 16:37
		// Ich weiß noch nicht, wie einfach es ist, SSL zu reden, aber
		// wenn das kein großer Krampf ist, kann ich hier natürlich
		// auch auf Port 443 prüfen. Dito für die Mail-Protokolle!
		return 80
	} else if ftp_pat.MatchString(host.Name) && !ports[21] {
		return 21
	} else if mx_pat.MatchString(host.Name) {
		if !ports[25] {
			return 25
		} else if !ports[110] {
			return 110
		} else if !ports[143] {
			return 143
		}
	}

	indexlist := rand.Perm(len(PORTS))
	for _, idx := range indexlist {
		if !ports[PORTS[idx]] {
			return PORTS[idx]
		}
	}

	return 0
} // func get_scan_port(host *Host, ports map[uint16]bool) uint16

type Scanner struct {
	db            *database.HostDB
	scan_queue    chan data.ScanRequest
	result_queue  chan data.ScanResult
	control_queue chan data.ControlMessage
	host_queue    chan data.HostWithPorts
	log           *log.Logger
	worker_cnt    int
	started       int
	lock          sync.Mutex
	running       bool
}

func CreateScanner(worker_cnt int) (*Scanner, error) {
	var err error
	var scanner *Scanner
	var msg string

	scanner = &Scanner{
		scan_queue:   make(chan data.ScanRequest, worker_cnt),
		result_queue: make(chan data.ScanResult, worker_cnt*2),
		host_queue:   make(chan data.HostWithPorts, worker_cnt*2),
		worker_cnt:   worker_cnt,
		started:      0,
	}

	if scanner.log, err = common.GetLogger("Scanner"); err != nil {
		msg = fmt.Sprintf("Error getting Logger instance for scanner: %s", err.Error())
		return nil, errors.New(msg)
	} else if scanner.db, err = database.OpenDB(common.DbPath); err != nil {
		msg = fmt.Sprintf("Error opening database at %s: %s",
			common.DbPath, err.Error())
		scanner.log.Println(msg)
		return nil, errors.New(msg)
	} else if common.Debug {
		scanner.log.Printf("Created new Scanner, will use %d workers, ready to go.\n", worker_cnt)
	}

	return scanner, nil
} // func CreateScanner(worker_cnt int) (*Scanner, error)

func (self *Scanner) Start() {
	self.lock.Lock()
	self.running = true
	self.lock.Unlock()

	if common.Debug {
		self.log.Printf("Scanner starting Host feeder and %d workers.\n", self.worker_cnt)
	}

	go self.hostFeeder()

	for i := 1; i <= self.worker_cnt; i++ {
		go self.worker(i)
		self.started++
	}
} // func (self *Scanner) Start()

func (self *Scanner) Stop() {
	self.lock.Lock()
	self.running = false
	self.lock.Unlock()
} // func (self *Scanner) Stop()

func (self *Scanner) IsRunning() bool {
	var is_running bool

	self.lock.Lock()
	is_running = self.running
	self.lock.Unlock()

	return is_running
} // func (self *Scanner) IsRunning() bool

func (self *Scanner) PrintStatus() {

} // func (self *Scanner) PrintStatus()

func (self *Scanner) Loop() {
	var err error
	var req data.ScanRequest
	var res data.ScanResult
	var msg string
	var control data.ControlMessage

	req = self.getRandomScanRequest()

	if common.Debug {
		self.log.Println("Scanner Loop() starting up...")
	}

	for self.IsRunning() {
		select {
		case control = <-self.control_queue:
			if common.Debug {
				self.log.Println("Got one control message!")
			}

			switch control {
			case data.CtlMsgStop:
				self.Stop()
				return
			case data.CtlMsgStatus:
				self.PrintStatus()
			}

		case self.scan_queue <- req:
			if common.Debug {
				self.log.Println("Scanner Loop dispatched one ScanRequest, getting another one.")
			}
			req = self.getRandomScanRequest()

		case res = <-self.result_queue:
			// Add Port to database!
			if common.Debug {
				var reply string
				if res.Reply == nil {
					reply = "NULL"
				} else {
					reply = *res.Reply
				}
				msg = fmt.Sprintf("Got ScanResult: %s:%d - %s",
					res.Host.Name, res.Port, reply)
				self.log.Println(msg)
			}

			if err = self.db.PortAdd(&res); err != nil {
				msg = fmt.Sprintf("Error adding Port to DB: %s", err.Error())
				self.log.Println(msg)
			}
		}
	}

} // func (self *Scanner) Loop()

func (self *Scanner) hostFeeder() {
	var hosts []data.Host
	var db *database.HostDB
	var err error
	var msg string

	if db, err = database.OpenDB(common.DbPath); err != nil {
		msg = fmt.Sprintf("Error opening DB at %s for hostFeeder: %s",
			common.DbPath, err.Error())
		self.log.Println(msg)
		return
	} else {
		defer db.Close()
	}

	if common.Debug {
		self.log.Println("hostFeeder() starting up...")
	}

	for self.IsRunning() {
		if hosts, err = db.HostGetRandom(self.worker_cnt * 10); err != nil {
			msg = fmt.Sprintf("Error getting (up to) %d random hosts: %s",
				self.worker_cnt, err.Error())
			self.log.Println(msg)
		} else {
			if common.Debug {
				self.log.Printf("hostFeeder retrieved %d hosts from the database.\n",
					len(hosts))
			}

			for _, host := range hosts {
				var ports []data.Port
				var phost *data.Host = new(data.Host)

				if ports, err = db.PortGetByHost(host.ID); err != nil {
					msg = fmt.Sprintf("Error getting ports for host %s/%s: %s",
						host.Name, host.Address, err.Error())
					self.log.Println(msg)
				} else {
					*phost = host
					host_with_ports := data.HostWithPorts{
						Host:  *phost,
						Ports: ports,
					}

					if common.Debug {
						self.log.Printf("Enqueueing host %s/%s as a scan target.\n",
							host.Address.String(), host.Name)
					}

					self.host_queue <- host_with_ports
				}
			}
		}
	}
} // func (self *Scanner) hostFeeder()

func (self *Scanner) getRandomScanRequest() data.ScanRequest {
	var req data.ScanRequest
	var portmap map[uint16]bool = make(map[uint16]bool)
	var hwp data.HostWithPorts

	if common.Debug {
		self.log.Println("Getting one random scan request from the host queue...")
	}

GET_HOST:
	hwp = <-self.host_queue

	if common.Debug {
		self.log.Printf("\t...got one random scan request from the host queue: %s\n",
			hwp.Host.Name)
	}

	for _, port := range hwp.Ports {
		portmap[port.Port] = true
	}

	req.Host = hwp.Host

	req.Port = get_scan_port(&req.Host, portmap)

	if req.Port == 0 {
		goto GET_HOST
	} else if common.Debug {
		self.log.Printf("Returning Request to scan %s:%d\n",
			req.Host.Name, req.Port)
	}

	return req
} // func (self *Scanner) getRandomScanRequest() ScanRequest

func (self *Scanner) worker(id int) {
	var request data.ScanRequest
	var result *data.ScanResult
	var err error

	defer func() {
		self.lock.Lock()
		self.worker_cnt--
		self.lock.Unlock()
	}()

	if common.Debug {
		self.log.Printf("Scanner worker %d starting up...\n",
			id)
	}

	for self.IsRunning() {
		request = <-self.scan_queue

		//result, err = scan_host(&request.Host, request.Port)
		if result, err = scan_host(&request.Host, request.Port); err != nil {
			msg := fmt.Sprintf("Error scanning %s:%d -- %s",
				request.Host.Name,
				request.Port,
				err.Error())
			self.log.Println(msg)
			result = new(data.ScanResult)
			result.Host = request.Host
			result.Port = request.Port
			result.Reply = nil
			result.Err = errors.New(msg)

			self.result_queue <- *result
		} else {
			if common.Debug {
				var reply string
				if result.Reply == nil {
					reply = "(NULL)"
				} else {
					reply = *result.Reply
				}
				self.log.Printf("Successfully scanned %s:%d - %s\n",
					request.Host.Name,
					request.Port,
					reply)
			}

			self.result_queue <- *result
		}
	}
} // func (self *Scanner) worker(id int)

func scan_host(host *data.Host, port uint16) (*data.ScanResult, error) {
	switch port {
	case 23:
		return scan_telnet(host, port)
	case 21, 22, 25, 110, 2525:
		return scan_plain(host, port)
	case 53, 5353:
		return scan_dns(host, port)
	case 79:
		return scan_finger(host, port)
	case 80, 8000, 8080, 8081, 3128, 3689, 631, 1024, 4444, 5800:
		return scan_http(host, port)
	case 161:
		return scan_snmp(host, port)
	default:
		return scan_plain(host, port)
	}
} // func scan_host(host *Host, port uint16) (*ScanResult, error)

func scan_plain(host *data.Host, port uint16) (*data.ScanResult, error) {
	if common.Debug {
		fmt.Printf("Scanning %s:%d using plain scanner.\n", host.Address.String(), port)
	}
	srv := fmt.Sprintf("%s:%d", host.Address, port)
	conn, err := net.Dial("tcp", srv)
	if err != nil {
		msg := fmt.Sprintf("Error connecting to %s: %s", srv, err.Error())
		return nil, errors.New(msg)
	} else {
		defer conn.Close()
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		msg := fmt.Sprintf("Error receiving data from %s: %s", srv, err.Error())
		return nil, errors.New(msg)
	} else {
		line = newline.ReplaceAllString(line, "")
		res := new(data.ScanResult)
		res.Host = *host
		res.Port = port
		res.Reply = &line
		if common.Debug {
			fmt.Printf("Got Reply: %s\n", line)
		}
		res.Stamp = time.Now()
		return res, nil
	}
} // func scan_plain(host *Host, port uint16) (*ScanResult, error)

func scan_finger(host *data.Host, port uint16) (*data.ScanResult, error) {
	var err error
	var recvbuffer []byte = make([]byte, 4096)
	var n int
	const TIMEOUT = 5 * time.Second

	if common.Debug {
		fmt.Printf("Fingering root@%s (port %d)...\n",
			host.Name, port)
	}

	srv := fmt.Sprintf("%s:%d", host.Address, port)
	conn, err := net.Dial("tcp", srv)
	if err != nil {
		msg := fmt.Sprintf("Error connecting to %s: %s", srv, err.Error())
		return nil, errors.New(msg)
	} else {
		defer conn.Close()
	}

	conn.Write([]byte("root\r\n")) // nolint: errcheck

	conn.SetDeadline(time.Now().Add(TIMEOUT)) // nolint: errcheck

	if n, err = conn.Read(recvbuffer); err != nil {
		msg := fmt.Sprintf("Error receiving from [%s]:%d - %s",
			host.Address, port, err.Error())
		return nil, errors.New(msg)
	} else {
		var reply_str *string = new(string)
		*reply_str = string((recvbuffer[:n]))
		result := &data.ScanResult{
			Host:  *host,
			Port:  port,
			Reply: reply_str,
			Stamp: time.Now(),
			Err:   nil,
		}
		return result, nil
	}
} // func scan_finger(host *Host, port uint16) (*ScanResult, error)

var dns_reply_pat *regexp.Regexp = regexp.MustCompile("\"([^\"]+)\"")

// Samstag, 05. 07. 2014, 20:26
// Den Code habe ich mehr oder weniger aus dem Beispiel im golang-dns Repository
// geklaut, Copyright 2011 Miek Gieben
//
// Samstag, 26. 07. 2014, 13:22
// Kann es sein, dass das nicht ganz so funktioniert, wie ich mir das vorstelle?
// Ich bekomme irgendwie nicht einen einzigen Port 53 erfolgreich gescannt...
//
// Ich habe den Quellcode kritisch angestarrt und keinen offensichtlichen Fehler
// entdeckt. Ich sollte mal testen, ob das Ding überhaupt funktioniert.
//
// Freitag, 01. 08. 2014, 17:59
// Mmmh, es gibt da ein kleines Problem: Die Replies, die in der Datenbank landen, sehen ungefähr so aus:
// version.bind.   1476526080      IN      TXT     "Microsoft DNS 6.1.7601 (1DB14556)"

func scan_dns(host *data.Host, port uint16) (*data.ScanResult, error) {
	if common.Debug {
		fmt.Printf("Scanning %s:%d using DNS scanner.\n", host.Address.String(), port)
	}
	m := new(dns.Msg)
	m.Question = make([]dns.Question, 1)
	c := new(dns.Client)
	m.Question[0] = dns.Question{Name: "version.bind.", Qtype: dns.TypeTXT, Qclass: dns.ClassCHAOS}
	addr := fmt.Sprintf("[%s]:%d", host.Address.String(), port)
	in, _, err := c.Exchange(m, addr)
	if err != nil {
		msg := fmt.Sprintf("Error asking %s for version.bind: %s", host.Name, err.Error())
		return nil, errors.New(msg)
	} else if in != nil && len(in.Answer) > 0 {
		reply := in.Answer[0]
		switch t := reply.(type) {
		case *dns.TXT:
			version_string := new(string)
			*version_string = t.String()
			match := dns_reply_pat.FindStringSubmatch(*version_string)
			if nil != match {
				*version_string = match[1]
			}

			result := new(data.ScanResult)
			result.Host = *host
			result.Port = port
			result.Reply = version_string
			result.Stamp = time.Now()
			if common.Debug {
				fmt.Printf("Got reply: %s:%d is %s\n",
					host.Address.String(),
					port,
					*version_string)
			}
			return result, nil
		default:
			// CANTHAPPEN
			println("Potzblitz! Damit konnte ja wirklich NIEMAND rechnen!")
			return nil, errors.New("No reply was received")
		}
	}

	return nil, errors.New("No valid reply was received, but error status was nil")
} // func scan_dns(host *Host, port uint16) (*ScanResult, error)

func scan_http(host *data.Host, port uint16) (*data.ScanResult, error) {
	if common.Debug {
		fmt.Printf("Scanning %s:%d using HTTP scanner.\n", host.Address.String(), port)
	}
	transport := &http.Transport{
		Proxy: nil,
	}
	client := new(http.Client)
	client.Transport = transport
	client.Timeout = 15 * time.Second

	url := fmt.Sprintf("http://%s:%d/", host.Address.String(), port)
	response, err := client.Head(url)
	if err != nil {
		msg := fmt.Sprintf("Error fetching headers for URL %s: %s", url, err.Error())
		return nil, errors.New(msg)
	}

	result := new(data.ScanResult)
	result.Host = *host
	result.Port = port
	var res_string = newline.ReplaceAllString(response.Header.Get("Server"), "")
	if common.Debug {
		fmt.Printf("http://%s:%d/ -> %s\n",
			host.Address.String(),
			port,
			res_string)
	}
	result.Reply = &res_string
	result.Stamp = time.Now()
	return result, nil
} // func scan_http(host *Host, port uint16) (*ScanResult, error)

func scan_snmp(host *data.Host, port uint16) (*data.ScanResult, error) {
	if common.Debug {
		fmt.Printf("Scanning %s:%d using SNMP scanner.\n", host.Address.String(), port)
	}
	snmp, err := gosnmp.NewGoSNMP(host.Address.String(), "public", gosnmp.Version2c, 5)
	if err != nil {
		return nil, err
	}

	result := new(data.ScanResult)
	result.Host = *host
	result.Port = port
	var res_string string
	success := false

	// 3.6.1.2.1.1.1.0
	resp, err := snmp.Get(".1.3.6.1.2.1.1.1.0")
	if err == nil {
	VARLOOP:
		for _, v := range resp.Variables {
			switch v.Type {
			case gosnmp.OctetString:
				res_string = v.Value.(string)
				success = true
				break VARLOOP
			}
		}
	}

	if success {
		result.Reply = new(string)
		*result.Reply = res_string
		//return result, nil
	}

	return result, nil
} // func scan_snmp(host *Host, port uint16) (*ScanResult, error)

func scan_telnet(host *data.Host, port uint16) (*data.ScanResult, error) {
	if common.Debug {
		fmt.Printf("Scanning %s:%d using Telnet scanner.\n", host.Address.String(), port)
	}
	var txtbuf []byte
	var recvbuffer []byte = make([]byte, 4096)
	var n int
	var probe []byte = []byte{
		0xff, 0xfc, 0x25, // Won't Authentication
		0xff, 0xfd, 0x03, // Do Suppress Go Ahead
		0xff, 0xfc, 0x18, // Won't Terminal Type
		0xff, 0xfc, 0x1f, // Won't Window Size
		0xff, 0xfc, 0x20, // Won't Terminal Speed
		0xff, 0xfb, 0x22, // Will Linemode
	}

	target := fmt.Sprintf("%s:%d", host.Address, port)

	conn, err := net.Dial("tcp", target)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to %s: %s", host.Name, err.Error())
	} else {
		defer conn.Close()
	}

	n, err = conn.Read(recvbuffer)
	if err != nil {
		return nil, fmt.Errorf("Error receiving from %s: %s", host.Name, err.Error())
	}

	conn.Write(probe) // nolint: errcheck
	var snd_fill int

	for {
		var i int
		snd_buf := make([]byte, 256)
		snd_fill = 0

		for i = 0; i < n; i++ {
			if recvbuffer[i] == 0xff {
				snd_buf[snd_fill] = 0xff
				snd_fill++
				i++
				switch recvbuffer[i] {
				case 0xfb: // WILL
					snd_buf[snd_fill] = 0xfe
					snd_fill++
				case 0xfd: // DO
					snd_buf[snd_fill] = 0xfc
					snd_fill++
				}
				i++
				snd_buf[snd_fill] = recvbuffer[i]
				snd_fill++
			} else if recvbuffer[i] < 0x80 {
				fmt.Printf("Received data from %s: %d/%d\n", host.Name, i, n)
				//return string(recvbuffer[i:n]), nil
				txtbuf = recvbuffer[i:n]
				goto TEXT_FOUND
			}
		}

		if snd_fill > 0 {
			_, err = conn.Write(snd_buf[:snd_fill])
			if err != nil {
				msg := fmt.Sprintf("Error sending snd_buf to server: %s\n", err.Error())
				fmt.Println(msg)
				return nil, errors.New(msg)
			}
		}

		n, err = conn.Read(recvbuffer)
		if err != nil {
			return nil, fmt.Errorf("Error receiving from %s: %s", host.Name, err.Error())
		} else {
			fmt.Printf("Received %d bytes of data from server.\n", n)
		}
	}

	//return nil, errors.New("No text found in output from server.")

TEXT_FOUND:
	begin := 0
	for ; begin < len(txtbuf); begin++ {
		r, _ := utf8.DecodeRune(txtbuf[begin:begin])
		if txtbuf[begin] >= 0x41 && unicode.IsPrint(r) {
			fmt.Printf("Found Printable character: 0x%02x\n", txtbuf[begin])
			break
		}
	}
	txtbuf = txtbuf[begin:]
	end := 1
	for ; end < len(txtbuf); end++ {
		if txtbuf[end] == 0x00 {
			end--
			txtbuf = txtbuf[:end]
			break
		}
	}
	fmt.Printf("%d bytes of data remaining.\n", len(txtbuf))
	for i := 0; i < len(txtbuf); i++ {
		fmt.Printf("%02d: 0x%02x\n", i, txtbuf[i])
	}

	result := new(data.ScanResult)
	result.Host = *host
	result.Port = port
	result.Reply = new(string)
	*result.Reply = string(txtbuf)
	result.Stamp = time.Now()
	return result, nil
} // func scan_telnet(host *Host, port uint16) (*ScanResult, error)
