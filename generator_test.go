// /Users/krylon/go/src/guang/generator_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 24. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-25 00:19:23 krylon>

package guang

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

var gen *HostGenerator

func TestCreateGenerator(t *testing.T) {
	var rng *rand.Rand = rand.New(rand.NewSource(time.Now().Unix()))

	var test_path string = fmt.Sprintf("/tmp/guang_test_%08x",
		rng.Int31())
	var err error

	fmt.Printf("Setting BASE_DIR to %s\n", test_path)
	SetBaseDir(test_path)

	if gen, err = CreateGenerator(8); err != nil {
		t.Fatalf("Error creating HostGenerator: %s", err.Error())
	} else if gen == nil {
		t.Fatal("CreateGenerator(8) did not return an error, but no HostGenerator, either!")
	} else {
		gen.Start()
	}
} // func TestCreateGenerator(t *testing.T)

func TestReceiveHosts(t *testing.T) {
	var host Host

	host = <-gen.HostQueue

	if DEBUG {
		fmt.Printf("Got one host: %s (%s)\n",
			host.Name, host.Address)
	}

	gen.Stop()

	time.Sleep(time.Second * 2)
} // func TestReceiveHosts(t *testing.T)
