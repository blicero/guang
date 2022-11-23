// /Users/krylon/go/src/guang/scanner.go
// -*- coding: utf-8; mode: go; -*-
// Created on 28. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-11-24 00:42:49 krylon>
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

var wwwPat *regexp.Regexp = regexp.MustCompile("(?i)^www")
var ftpPat *regexp.Regexp = regexp.MustCompile("(?i)^ftp")
var mxPat *regexp.Regexp = regexp.MustCompile("(?i)^(?:mx|mail|smtp|pop|imap)")
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

// Ports is the list of ports (TCP and UDP) we consider interesting.
var Ports []uint16 = []uint16{
	21,
	22,
	23,
	25,
	53,
	79,
	80,
	110,
	143,
	161,
	443,
	631,
	1024,
	4444,
	2525,
	5353,
	5800,
	5900,
	8000,
	8080,
	8081,
}

func getScanPort(host *data.Host, ports map[uint16]bool) uint16 {
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
	} else if wwwPat.MatchString(host.Name) && !ports[80] {
		// Samstag, 05. 07. 2014, 16:37
		// Ich weiß noch nicht, wie einfach es ist, SSL zu reden, aber
		// wenn das kein großer Krampf ist, kann ich hier natürlich
		// auch auf Port 443 prüfen. Dito für die Mail-Protokolle!
		return 80
	} else if ftpPat.MatchString(host.Name) && !ports[21] {
		return 21
	} else if mxPat.MatchString(host.Name) {
		if !ports[25] {
			return 25
		} else if !ports[110] {
			return 110
		} else if !ports[143] {
			return 143
		}
	}

	indexlist := rand.Perm(len(Ports))
	for _, idx := range indexlist {
		if !ports[Ports[idx]] {
			return Ports[idx]
		}
	}

	return 0
} // func get_scan_port(host *Host, ports map[uint16]bool) uint16

// Scanner is a port scanner. Kind of.
type Scanner struct {
	db        *database.HostDB
	scanQ     chan data.ScanRequest
	resultQ   chan data.ScanResult
	RC        chan data.ControlMessage
	hostQ     chan data.HostWithPorts
	mmQ       chan data.ControlMessage
	log       *log.Logger
	workerCnt int
	started   int
	lock      sync.RWMutex
	running   bool
}

// CreateScanner creates a new Scanner.
func CreateScanner(workerCnt int) (*Scanner, error) {
	var err error
	var scanner *Scanner
	var msg string

	scanner = &Scanner{
		scanQ:     make(chan data.ScanRequest, workerCnt),
		resultQ:   make(chan data.ScanResult, workerCnt*2),
		hostQ:     make(chan data.HostWithPorts, workerCnt),
		mmQ:       make(chan data.ControlMessage, workerCnt),
		RC:        make(chan data.ControlMessage, 2),
		workerCnt: workerCnt,
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
		scanner.log.Printf("[DEBUG] Created new Scanner, will use %d workers, ready to go.\n", workerCnt)
	}

	return scanner, nil
} // func CreateScanner(worker_cnt int) (*Scanner, error)

// Start starts the Scanner. If it is already running, this method does nothing.
func (sc *Scanner) Start() {
	sc.lock.Lock()
	if sc.running {
		sc.lock.Unlock()
		return
	}

	sc.running = true
	defer sc.lock.Unlock()

	if common.Debug {
		sc.log.Printf("Scanner starting Host feeder and %d workers.\n", sc.workerCnt)
	}

	go sc.hostFeeder()

	for i := 1; i <= sc.workerCnt; i++ {
		go sc.worker(i)
	}
} // func (sc *Scanner) Start()

// Stop tells the Scanner to stop.
func (sc *Scanner) Stop() {
	sc.lock.Lock()
	sc.running = false
	sc.lock.Unlock()
} // func (sc *Scanner) Stop()

func (sc *Scanner) Count() int {
	sc.lock.RLock()
	var cnt = sc.started
	sc.lock.RUnlock()
	return cnt
} // func (sc *Scanner) Count() int

func (sc *Scanner) cntInc() {
	sc.lock.Lock()
	sc.started++
	sc.lock.Unlock()
} // func (sc *Scanner) cntInc()

func (sc *Scanner) cntDec() {
	sc.lock.Lock()
	sc.started--
	sc.lock.Unlock()
} // func (sc *Scanner) cntDec()

// IsRunning returns true if the Scanner is running.
func (sc *Scanner) IsRunning() bool {
	var isRunning bool

	sc.lock.RLock()
	isRunning = sc.running
	sc.lock.RUnlock()

	return isRunning
} // func (sc *Scanner) IsRunning() bool

// PrintStatus emits the Scanner's status.
func (sc *Scanner) PrintStatus() {

} // func (sc *Scanner) PrintStatus()

// Loop is the Scanner's main loop.
func (sc *Scanner) Loop() {
	var (
		err error
		req data.ScanRequest
		res data.ScanResult
		msg string
		ctl data.ControlMessage
	)

	req = sc.getRandomScanRequest()

	if common.Debug {
		sc.log.Println("Scanner Loop() starting up...")
	}

	for sc.IsRunning() {
		select {
		case ctl = <-sc.RC:
			sc.log.Printf("[DEBUG] Got one control message: %s\n",
				ctl)

			switch ctl {
			case data.CtlMsgShutdown:
				sc.Stop()
				return
			case data.CtlMsgStop:
				sc.log.Printf("[DEBUG] Telling one worker to stop.\n")
				sc.mmQ <- data.CtlMsgStop
			case data.CtlMsgSpawn:
				sc.log.Printf("[DEBUG] Spawning one additional worker\n")
				go sc.worker(sc.Count())
			case data.CtlMsgStatus:
				sc.PrintStatus()
			}

		case sc.scanQ <- req:
			if common.Debug {
				sc.log.Println("Scanner Loop dispatched one ScanRequest, getting another one.")
			}
			req = sc.getRandomScanRequest()

		case res = <-sc.resultQ:
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
				sc.log.Println(msg)
			}

			if err = sc.db.PortAdd(&res); err != nil {
				msg = fmt.Sprintf("Error adding Port to DB: %s", err.Error())
				sc.log.Println(msg)
			}
		}
	}

} // func (sc *Scanner) Loop()

func (sc *Scanner) hostFeeder() {
	var hosts []data.Host
	var db *database.HostDB
	var err error
	var msg string

	if db, err = database.OpenDB(common.DbPath); err != nil {
		msg = fmt.Sprintf("Error opening DB at %s for hostFeeder: %s",
			common.DbPath, err.Error())
		sc.log.Println(msg)
		return
	}

	defer db.Close()

	if common.Debug {
		sc.log.Println("hostFeeder() starting up...")
	}

	for sc.IsRunning() {
		if hosts, err = db.HostGetRandom(sc.workerCnt); err != nil {
			msg = fmt.Sprintf("Error getting (up to) %d random hosts: %s",
				sc.workerCnt, err.Error())
			sc.log.Println(msg)
		} else {
			if common.Debug {
				sc.log.Printf("hostFeeder retrieved %d hosts from the database.\n",
					len(hosts))
			}

			for _, host := range hosts {
				var ports []data.Port
				var phost *data.Host = new(data.Host)

				if ports, err = db.PortGetByHost(host.ID); err != nil {
					msg = fmt.Sprintf("Error getting ports for host %s/%s: %s",
						host.Name, host.Address, err.Error())
					sc.log.Println(msg)
				} else {
					*phost = host
					hostWithPorts := data.HostWithPorts{
						Host:  *phost,
						Ports: ports,
					}

					if common.Debug {
						sc.log.Printf("Enqueueing host %s/%s as a scan target.\n",
							host.Address.String(), host.Name)
					}

					sc.hostQ <- hostWithPorts
				}
			}
		}
	}
} // func (sc *Scanner) hostFeeder()

func (sc *Scanner) getRandomScanRequest() data.ScanRequest {
	var req data.ScanRequest
	var portmap map[uint16]bool = make(map[uint16]bool)
	var hwp data.HostWithPorts

	if common.Debug {
		sc.log.Println("Getting one random scan request from the host queue...")
	}

GET_HOST:
	hwp = <-sc.hostQ

	if common.Debug {
		sc.log.Printf("\t...got one random scan request from the host queue: %s\n",
			hwp.Host.Name)
	}

	for _, port := range hwp.Ports {
		portmap[port.Port] = true
	}

	req.Host = hwp.Host

	req.Port = getScanPort(&req.Host, portmap)

	if req.Port == 0 {
		goto GET_HOST
	} else if common.Debug {
		sc.log.Printf("Returning Request to scan %s:%d\n",
			req.Host.Name, req.Port)
	}

	return req
} // func (sc *Scanner) getRandomScanRequest() ScanRequest

func (sc *Scanner) worker(id int) {
	var (
		err     error
		request data.ScanRequest
		result  *data.ScanResult
		pulse   *time.Ticker
		msg     data.ControlMessage
	)

	pulse = time.NewTicker(common.HeartBeat)
	defer pulse.Stop()

	sc.cntInc()
	defer sc.cntDec()

	if common.Debug {
		sc.log.Printf("[DEBUG] Scanner worker %d starting up...\n",
			id)
	}

	for sc.IsRunning() {
		select {
		case msg = <-sc.mmQ:
			switch msg {
			case data.CtlMsgStop:
				sc.log.Printf("[INFO] Worker #%d is quitting as ordered.\n",
					id)
				return
			default:
				sc.log.Printf("[DEBUG] Scanner Worker #%d ignoring message %s\n",
					id,
					msg)
			}
		case request = <-sc.scanQ:
			if result, err = scanHost(&request.Host, request.Port); err != nil {
				msg := fmt.Sprintf("Error scanning %s:%d -- %s",
					request.Host.Name,
					request.Port,
					err.Error())
				sc.log.Println(msg)
				result = new(data.ScanResult)
				result.Host = request.Host
				result.Port = request.Port
				result.Reply = nil
				result.Err = errors.New(msg)

				sc.resultQ <- *result
			} else {
				if common.Debug {
					var reply string
					if result.Reply == nil {
						reply = "(NULL)"
					} else {
						reply = *result.Reply
					}
					sc.log.Printf("Successfully scanned %s:%d - %s\n",
						request.Host.Name,
						request.Port,
						reply)
				}

				sc.resultQ <- *result
			}
		case <-pulse.C:
			continue
		}
	}
} // func (sc *Scanner) worker(id int)

func scanHost(host *data.Host, port uint16) (*data.ScanResult, error) {
	switch port {
	case 23:
		return scanTelnet(host, port)
	case 21, 22, 25, 110, 2525:
		return scanPlain(host, port)
	case 53, 5353:
		return scanDNS(host, port)
	case 79:
		return scanFinger(host, port)
	case 80, 443, 8000, 8080, 8081, 3128, 3689, 631, 1024, 4444, 5800:
		return scanHTTP(host, port)
	case 161:
		return scanSNMP(host, port)
	default:
		return scanPlain(host, port)
	}
} // func scanHost(host *Host, port uint16) (*ScanResult, error)

func scanPlain(host *data.Host, port uint16) (*data.ScanResult, error) {
	if common.Debug {
		fmt.Printf("Scanning %s:%d using plain scanner.\n", host.Address.String(), port)
	}
	srv := fmt.Sprintf("%s:%d", host.Address, port)
	conn, err := net.Dial("tcp", srv)
	if err != nil {
		msg := fmt.Sprintf("Error connecting to %s: %s", srv, err.Error())
		return nil, errors.New(msg)
	}

	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		msg := fmt.Sprintf("Error receiving data from %s: %s", srv, err.Error())
		return nil, errors.New(msg)
	}

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
} // func scan_plain(host *Host, port uint16) (*ScanResult, error)

func scanFinger(host *data.Host, port uint16) (*data.ScanResult, error) {
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
	}

	defer conn.Close()

	conn.Write([]byte("root\r\n")) // nolint: errcheck

	conn.SetDeadline(time.Now().Add(TIMEOUT)) // nolint: errcheck

	if n, err = conn.Read(recvbuffer); err != nil {
		msg := fmt.Sprintf("Error receiving from [%s]:%d - %s",
			host.Address, port, err.Error())
		return nil, errors.New(msg)
	}

	var replyStr *string = new(string)
	*replyStr = string((recvbuffer[:n]))
	result := &data.ScanResult{
		Host:  *host,
		Port:  port,
		Reply: replyStr,
		Stamp: time.Now(),
		Err:   nil,
	}
	return result, nil
} // func scan_finger(host *Host, port uint16) (*ScanResult, error)

var dnsReplyPat *regexp.Regexp = regexp.MustCompile("\"([^\"]+)\"")

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

func scanDNS(host *data.Host, port uint16) (*data.ScanResult, error) {
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
			versionStr := new(string)
			*versionStr = t.String()
			match := dnsReplyPat.FindStringSubmatch(*versionStr)
			if nil != match {
				*versionStr = match[1]
			}

			result := new(data.ScanResult)
			result.Host = *host
			result.Port = port
			result.Reply = versionStr
			result.Stamp = time.Now()
			if common.Debug {
				fmt.Printf("Got reply: %s:%d is %s\n",
					host.Address.String(),
					port,
					*versionStr)
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

func scanHTTP(host *data.Host, port uint16) (*data.ScanResult, error) {
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
	var resStr = newline.ReplaceAllString(response.Header.Get("Server"), "")
	if common.Debug {
		fmt.Printf("http://%s:%d/ -> %s\n",
			host.Address.String(),
			port,
			resStr)
	}
	result.Reply = &resStr
	result.Stamp = time.Now()
	return result, nil
} // func scan_http(host *Host, port uint16) (*ScanResult, error)

func scanSNMP(host *data.Host, port uint16) (*data.ScanResult, error) {
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
	var resStr string
	success := false

	// 3.6.1.2.1.1.1.0
	resp, err := snmp.Get(".1.3.6.1.2.1.1.1.0")
	if err == nil {
	VARLOOP:
		for _, v := range resp.Variables {
			switch v.Type {
			case gosnmp.OctetString:
				resStr = v.Value.(string)
				success = true
				break VARLOOP
			}
		}
	}

	if success {
		result.Reply = new(string)
		*result.Reply = resStr
		//return result, nil
	}

	return result, nil
} // func scan_snmp(host *Host, port uint16) (*ScanResult, error)

func scanTelnet(host *data.Host, port uint16) (*data.ScanResult, error) {
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
	}

	defer conn.Close()

	n, err = conn.Read(recvbuffer)
	if err != nil {
		return nil, fmt.Errorf("Error receiving from %s: %s", host.Name, err.Error())
	}

	conn.Write(probe) // nolint: errcheck
	var sndFill int

	for {
		var i int
		sndBuf := make([]byte, 256)
		sndFill = 0

		for i = 0; i < n; i++ {
			if recvbuffer[i] == 0xff {
				sndBuf[sndFill] = 0xff
				sndFill++
				i++
				switch recvbuffer[i] {
				case 0xfb: // WILL
					sndBuf[sndFill] = 0xfe
					sndFill++
				case 0xfd: // DO
					sndBuf[sndFill] = 0xfc
					sndFill++
				}
				i++
				sndBuf[sndFill] = recvbuffer[i]
				sndFill++
			} else if recvbuffer[i] < 0x80 {
				fmt.Printf("Received data from %s: %d/%d\n", host.Name, i, n)
				//return string(recvbuffer[i:n]), nil
				txtbuf = recvbuffer[i:n]
				goto TEXT_FOUND
			}
		}

		if sndFill > 0 {
			_, err = conn.Write(sndBuf[:sndFill])
			if err != nil {
				msg := fmt.Sprintf("Error sending snd_buf to server: %s\n", err.Error())
				fmt.Println(msg)
				return nil, errors.New(msg)
			}
		}

		n, err = conn.Read(recvbuffer)
		if err != nil {
			return nil, fmt.Errorf("Error receiving from %s: %s", host.Name, err.Error())
		}

		fmt.Printf("Received %d bytes of data from server.\n", n)
	}

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
