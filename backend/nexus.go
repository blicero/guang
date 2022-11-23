// /Users/krylon/go/src/guang/backend/nexus.go
// -*- coding: utf-8; mode: go; -*-
// Created on 12. 02. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2022-11-24 00:44:40 krylon>

package backend

import (
	"log"

	"github.com/blicero/guang/backend/facility"
	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
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
	return nx.xfr.Count()
} // func (nx *Nexus) GetXFRCount() int

func (nx *Nexus) SpawnWorker(f facility.Facility, n int) {
	var c chan data.ControlMessage

	switch f {
	case facility.Generator:
		c = nx.generator.RC
	case facility.Scanner:
		c = nx.scanner.RC
	case facility.XFR:
		c = nx.xfr.RC
	default:
		nx.log.Printf("[ERROR] Don't know how to spawn more workers for %s\n",
			f)
		return
	}

	for i := 0; i < n; i++ {
		c <- data.CtlMsgSpawn
	}
} // func (nx *Nexus) SpawnWorker(f facility.Facility, n int)

func (nx *Nexus) StopWorker(f facility.Facility, n int) {
	var c chan data.ControlMessage

	switch f {
	case facility.Generator:
		c = nx.generator.RC
	case facility.Scanner:
		c = nx.scanner.RC
	case facility.XFR:
		c = nx.xfr.RC
	default:
		nx.log.Printf("[ERROR] Don't know how to stop workers for %s\n",
			f)
		return
	}

	nx.log.Printf("[INFO] Stopping %d %s workers\n",
		n,
		f)

	for i := 0; i < n; i++ {
		c <- data.CtlMsgStop
	}

	nx.log.Printf("[INFO] Sent %d stop messages to %s\n",
		n,
		f)
} // func (nx *Nexus) StopWorker(f facility.Facility, n int)

func (nx *Nexus) WorkerCount(f facility.Facility) int {
	switch f {
	case facility.Generator:
		return nx.generator.Count()
	case facility.Scanner:
		return nx.scanner.Count()
	case facility.XFR:
		return nx.xfr.Count()
	default:
		return 0
	}
} // func (nx *Nexus) WorkerCount(f facility.Facility) int64
