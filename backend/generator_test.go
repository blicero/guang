// /Users/krylon/go/src/guang/generator_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 24. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-27 21:33:19 krylon>

package backend

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/blicero/guang/common"
)

var gen *HostGenerator

func TestCreateGenerator(t *testing.T) {
	var rng *rand.Rand = rand.New(rand.NewSource(time.Now().Unix()))

	var test_path string = fmt.Sprintf("/tmp/guang_test_%08x",
		rng.Int31())
	var err error

	fmt.Printf("Setting BASE_DIR to %s\n", test_path)
	common.SetBaseDir(test_path)

	if gen, err = CreateGenerator(8); err != nil {
		t.Fatalf("Error creating HostGenerator: %s", err.Error())
	} else if gen == nil {
		t.Fatal("CreateGenerator(8) did not return an error, but no HostGenerator, either!")
	} else {
		gen.Start()
	}
} // func TestCreateGenerator(t *testing.T)

func TestReceiveHosts(t *testing.T) {
	var host = <-gen.HostQueue

	if common.DEBUG {
		fmt.Printf("Got one host: %s (%s)\n",
			host.Name, host.Address)
	}

	gen.Stop()

	time.Sleep(time.Second * 2)
} // func TestReceiveHosts(t *testing.T)
