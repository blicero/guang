// /Users/krylon/go/src/guang/generator.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-27 16:31:32 krylon>
//
// IIRC, throughput never was much of an issue with this part of the program.
// But if it were, there are a few tricks on could pull here.
// Especially, right now, there is one of each blacklist type shared across
// all workers. If each had its own blacklist, it would probably improve
// parallelism. OTOH, the main problem here is probably DNS lookups.

package backend

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/fsouza/gokabinet/kc"
)

type HostGenerator struct {
	HostQueue  chan Host
	name_bl    *NameBlacklist
	addr_bl    *IPBlacklist
	cache      *kc.DB
	lock       sync.Mutex
	running    bool
	worker_cnt int
	log        *log.Logger
}

func CreateGenerator(worker_cnt int) (*HostGenerator, error) {
	var err error
	var msg string

	gen := &HostGenerator{
		HostQueue:  make(chan Host, worker_cnt*2),
		running:    true,
		worker_cnt: worker_cnt,
		//cache:      cabinet.New(),
		name_bl: DefaultNameBlacklist(),
		addr_bl: DefaultIPBlacklist(),
	}

	if gen.log, err = GetLogger("Generator"); err != nil {
		fmt.Printf("Error getting Logger instance for host generator: %s\n",
			err.Error())
		return nil, err
		//} else if err = gen.cache.Open(HOST_CACHE_PATH, cabinet.KCOWRITER|cabinet.KCOCREATE|cabinet.KCOAUTOTRAN|cabinet.KCOAUTOSYNC); err != nil {
	} else if gen.cache, err = kc.Open(HOST_CACHE_PATH, kc.WRITE); err != nil {
		msg = fmt.Sprintf("Error opening Host cache at %s: %s",
			HOST_CACHE_PATH, err.Error())
		gen.log.Println(msg)
		return nil, errors.New(msg)
	}

	return gen, nil
} // func CreateGenerator(worker_cnt int) (*HostGenerator, error)

func (self *HostGenerator) Start() {
	for i := 0; i < self.worker_cnt; i++ {
		go self.worker(i)
	}
} // func (self *HostGenerator) Start()

func (self *HostGenerator) IsRunning() bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.running
} // func (self *HostGenerator) IsRunning() bool

func (self *HostGenerator) Stop() {
	self.lock.Lock()
	self.running = false
	self.lock.Unlock()
} // func (self *HostGenerator) Stop()

func (self *HostGenerator) worker(id int) {
	var msg, astr string
	var err error
	var rng *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	var addr net.IP
	var namelist []string

MAIN_LOOP:
	for self.IsRunning() {
		var host Host
		for addr = self.get_rand_ip(rng); self.addr_bl.MatchesIP(addr); addr = self.get_rand_ip(rng) {
			// This loop has no body.
			// It's all in the head.
		}

		astr = addr.String()

		if res, err := self.cache.GetInt(astr); err != nil {
			msg = fmt.Sprintf("Error looking for %s in cache: %s",
				astr, err.Error())
		} else if res != 0 {
			continue MAIN_LOOP
		} else {
			self.cache.SetInt(astr, 1)
			self.cache.Commit()
		}

		if namelist, err = net.LookupAddr(astr); err != nil {
			continue MAIN_LOOP
		} else if len(namelist) == 0 {
			msg = fmt.Sprintf("net.LookupAddr(%s) returned neither an error nor any names",
				astr)
			self.log.Println(msg)
			continue MAIN_LOOP
		} else if self.name_bl.Matches(namelist[0]) {
			continue MAIN_LOOP
		} else {
			host.Address = addr
			host.Name = namelist[0]
			host.Source = HOST_SOURCE_GEN
			host.Added = time.Now()
			self.HostQueue <- host
		}
	}

	if DEBUG {
		self.log.Printf("Generator worker #%d is quitting.\n", id)
	}
} // func (self *HostGenerator) worker(id int)

// Create and return a random IPv4 address.
func (self *HostGenerator) get_rand_ip(rng *rand.Rand) net.IP {
	var octets [4]byte

	octets[0] = byte(rng.Intn(256))
	octets[1] = byte(rng.Intn(256))
	octets[2] = byte(rng.Intn(256))
	octets[3] = byte(rng.Intn(256))

	return net.IPv4(octets[0], octets[1], octets[2], octets[3])
} // func (gen *IPGenerator) get_rand_ip() net.IP
