// /Users/krylon/go/src/guang/generator.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-29 18:23:56 krylon>
//
// IIRC, throughput never was much of an issue with this part of the program.
// But if it were, there are a few tricks on could pull here.
// Especially, right now, there is one of each blacklist type shared across
// all workers. If each had its own blacklist, it would probably improve
// parallelism. OTOH, the main problem here is probably DNS lookups.

package generator

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/blicero/guang/blacklist"
	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/fsouza/gokabinet/kc"
)

type HostGenerator struct {
	HostQueue  chan data.Host
	name_bl    *blacklist.NameBlacklist
	addr_bl    *blacklist.IPBlacklist
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
		HostQueue:  make(chan data.Host, worker_cnt*2),
		running:    true,
		worker_cnt: worker_cnt,
		name_bl:    blacklist.DefaultNameBlacklist(),
		addr_bl:    blacklist.DefaultIPBlacklist(),
	}

	if gen.log, err = common.GetLogger("Generator"); err != nil {
		fmt.Printf("Error getting Logger instance for host generator: %s\n",
			err.Error())
		return nil, err
		//} else if err = gen.cache.Open(HOST_CACHE_PATH, cabinet.KCOWRITER|cabinet.KCOCREATE|cabinet.KCOAUTOTRAN|cabinet.KCOAUTOSYNC); err != nil {
	} else if gen.cache, err = kc.Open(common.HOST_CACHE_PATH, kc.WRITE); err != nil {
		msg = fmt.Sprintf("Error opening Host cache at %s: %s",
			common.HOST_CACHE_PATH, err.Error())
		gen.log.Println(msg)
		return nil, errors.New(msg)
	}

	return gen, nil
} // func CreateGenerator(worker_cnt int) (*HostGenerator, error)

func (gen *HostGenerator) Start() {
	for i := 0; i < gen.worker_cnt; i++ {
		go gen.worker(i)
	}
} // func (gen *HostGenerator) Start()

func (gen *HostGenerator) IsRunning() bool {
	gen.lock.Lock()
	defer gen.lock.Unlock()
	return gen.running
} // func (gen *HostGenerator) IsRunning() bool

func (gen *HostGenerator) Stop() {
	gen.lock.Lock()
	gen.running = false
	gen.lock.Unlock()
} // func (gen *HostGenerator) Stop()

func (gen *HostGenerator) Count() int {
	gen.lock.Lock()
	var cnt = gen.worker_cnt
	gen.lock.Unlock()
	return cnt
} // func (gen *HostGenerator) Count() int

func (gen *HostGenerator) worker(id int) {
	defer func() {
		gen.lock.Lock()
		gen.worker_cnt--
		gen.lock.Unlock()
	}()

	var msg, astr string
	var err error
	var rng *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	var addr net.IP
	var namelist []string

MAIN_LOOP:
	for gen.IsRunning() {
		var host data.Host
		for addr = gen.get_rand_ip(rng); gen.addr_bl.MatchesIP(addr); addr = gen.get_rand_ip(rng) {
			// This loop has no body.
			// It's all in the head.
		}

		astr = addr.String()

		if res, err := gen.cache.GetInt(astr); err != nil {
			// msg = fmt.Sprintf("Error looking for %s in cache: %s",
			// 	astr, err.Error())
			// gen.log.Println(msg)
		} else if res != 0 {
			continue MAIN_LOOP
		} else {
			gen.cache.SetInt(astr, 1) // nolint: errcheck
			gen.cache.Commit()        // nolint: errcheck
		}

		if namelist, err = net.LookupAddr(astr); err != nil {
			continue MAIN_LOOP
		} else if len(namelist) == 0 {
			msg = fmt.Sprintf("net.LookupAddr(%s) returned neither an error nor any names",
				astr)
			gen.log.Println(msg)
			continue MAIN_LOOP
		} else if gen.name_bl.Matches(namelist[0]) {
			continue MAIN_LOOP
		} else {
			host.Address = addr
			host.Name = namelist[0]

			host.Source = data.HOST_SOURCE_GEN
			host.Added = time.Now()
			gen.HostQueue <- host
		}
	}

	if common.DEBUG {
		gen.log.Printf("Generator worker #%d is quitting.\n", id)
	}
} // func (gen *HostGenerator) worker(id int)

// Create and return a random IPv4 address.
func (gen *HostGenerator) get_rand_ip(rng *rand.Rand) net.IP {
	var octets [4]byte

	octets[0] = byte(rng.Intn(256))
	octets[1] = byte(rng.Intn(256))
	octets[2] = byte(rng.Intn(256))
	octets[3] = byte(rng.Intn(256))

	return net.IPv4(octets[0], octets[1], octets[2], octets[3])
} // func (gen *IPGenerator) get_rand_ip() net.IP
