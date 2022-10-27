// /Users/krylon/go/src/guang/backend/nexus.go
// -*- coding: utf-8; mode: go; -*-
// Created on 12. 02. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2022-10-27 20:47:19 krylon>

package backend

import (
	"log"

	"github.com/blicero/guang/common"
)

type Nexus struct {
	generator *HostGenerator
	scanner   *Scanner
	xfr       *XFRClient
	log       *log.Logger
}

func CreateNexus(gen *HostGenerator, scanner *Scanner, xfr *XFRClient) (*Nexus, error) {
	var nexus *Nexus = new(Nexus)
	var err error

	if nexus.log, err = common.GetLogger("Nexus"); err != nil {
		return nil, err
	} else {
		nexus.generator = gen
		nexus.scanner = scanner
		nexus.xfr = xfr
		return nexus, nil
	}
} // func CreateNexus(gen *HostGenerator, scanner *Scanner, xfr *XFRClient) (*Nexus, error)

func (self *Nexus) GetGeneratorCount() int {
	var cnt int

	self.generator.lock.Lock()
	cnt = self.generator.worker_cnt
	self.generator.lock.Unlock()

	return cnt
} // func (self *Nexus) GetGeneratorCount() int

func (self *Nexus) GetScannerCount() int {
	var cnt int

	self.scanner.lock.Lock()
	cnt = self.scanner.worker_cnt
	self.scanner.lock.Unlock()

	return cnt
} // func (self *Nexus) GetScannerCount() int

func (self *Nexus) GetXFRCount() int {
	var cnt int

	self.xfr.lock.Lock()
	cnt = self.xfr.worker_cnt
	self.xfr.lock.Unlock()

	return cnt
} // func (self *Nexus) GetXFRCount() int

func (self *Nexus) StartScanner() {
	self.scanner.Start()
} // func (self *Nexus) StartScanner()

func (self *Nexus) StartGenerator() {
	self.generator.Start()
} // func (Self *Nexus) StartGenerator()

func (self *Nexus) StartXFR(cnt int) {
	self.xfr.Start(cnt)
} // func (self *Nexus) StartXFR()
