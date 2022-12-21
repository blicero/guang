// /Users/krylon/go/src/guang/generator_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 24. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-12-20 19:50:04 krylon>

package generator

import (
	"fmt"
	"testing"
	"time"

	"github.com/blicero/guang/common"
)

var gen *HostGenerator

func TestCreateGenerator(t *testing.T) {
	var (
		testPath = time.Now().Format("/tmp/guang_test_generator_2006_01_02__15_04_05")
		err      error
	)

	fmt.Printf("Setting BASE_DIR to %s\n", testPath)
	common.SetBaseDir(testPath)

	if gen, err = CreateGenerator(8); err != nil {
		t.Fatalf("Error creating HostGenerator: %s", err.Error())
	} else if gen == nil {
		t.Fatal("CreateGenerator(8) did not return an error, but no HostGenerator, either!")
	} else {
		gen.Start()
	}
} // func TestCreateGenerator(t *testing.T)

func TestReceiveHosts(t *testing.T) {
	if gen == nil {
		t.SkipNow()
	}

	var host = <-gen.HostQueue

	if common.Debug {
		fmt.Printf("Got one host: %s (%s)\n",
			host.Name, host.Address)
	}

	gen.Stop()

	time.Sleep(time.Second * 2)
} // func TestReceiveHosts(t *testing.T)
