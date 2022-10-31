// /Users/krylon/go/src/guang/backend/nexus.go
// -*- coding: utf-8; mode: go; -*-
// Created on 12. 02. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2022-10-31 19:13:10 krylon>

package backend

import (
	"log"

	"github.com/blicero/guang/common"
	"github.com/blicero/guang/generator"
	"github.com/blicero/guang/xfr"
)

// Nexus aggregates the various pieces that comprise the backend.
type Nexus struct {
	generator *generator.HostGenerator
	scanner   *Scanner
	xfr       *xfr.Client
	log       *log.Logger
}

// CreateNexus creates a new Nexus instance with the given components.
func CreateNexus(gen *generator.HostGenerator, scanner *Scanner, xfr *xfr.Client) (*Nexus, error) {
	var nexus *Nexus = new(Nexus)
	var err error

	if nexus.log, err = common.GetLogger("Nexus"); err != nil {
		return nil, err
	}

	nexus.generator = gen
	nexus.scanner = scanner
	nexus.xfr = xfr
	return nexus, nil
} // func CreateNexus(gen *HostGenerator, scanner *Scanner, xfr *XFRClient) (*Nexus, error)

// GetGeneratorCount returns the number of workers in the Generator.
func (nx *Nexus) GetGeneratorCount() int {
	return nx.generator.Count()
} // func (nx *Nexus) GetGeneratorCount() int

// GetScannerCount returns the number of workers in the Scanner
func (nx *Nexus) GetScannerCount() int {
	var cnt int

	nx.scanner.lock.Lock()
	cnt = nx.scanner.workerCnt
	nx.scanner.lock.Unlock()

	return cnt
} // func (nx *Nexus) GetScannerCount() int

// GetXFRCount returns the number of XFR workers.
func (nx *Nexus) GetXFRCount() int {
	return nx.xfr.WorkerCount()
} // func (nx *Nexus) GetXFRCount() int

// StartScanner starts the Scanner.
func (nx *Nexus) StartScanner() {
	nx.scanner.Start()
} // func (nx *Nexus) StartScanner()

// StartGenerator starts the HostGenerator.
func (nx *Nexus) StartGenerator() {
	nx.generator.Start()
} // func (Nx *Nexus) StartGenerator()

// StartXFR starts the XFR workers.
func (nx *Nexus) StartXFR(cnt int) {
	nx.xfr.Start(cnt)
} // func (nx *Nexus) StartXFR()
