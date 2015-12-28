// /Users/krylon/go/src/guang/guang.go
// -*- coding: utf-8; mode: go; -*-
// Created on 27. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-27 18:00:17 krylon>

package main

import (
	"flag"
	"fmt"
	"guang/backend"
	"log"
	"os"

	"net/http"
	_ "net/http/pprof"
)

func main() {
	var gen_cnt int = 16
	var xfr_worker_cnt int = 2
	var do_profile bool = false
	var err error
	// var log *log.Logger
	var gen *backend.HostGenerator
	//var xfr *backend.XFRClient
	var xfr_queue chan string
	var db *backend.HostDB

	flag.IntVar(&gen_cnt, "generators", gen_cnt, "Number of Host Generators to run")
	flag.IntVar(&xfr_worker_cnt, "xfr", xfr_worker_cnt, "Number of XFR workers to run")
	flag.BoolVar(&do_profile, "profile", do_profile, "Run the builtin profiling server")

	flag.Parse()

	if do_profile {
		go func() {
			log.Println(http.ListenAndServe(":7998", nil))
		}()
	}

	if db, err = backend.OpenDB(backend.DB_PATH); err != nil {
		log.Printf("Error opening database at %s: %s\n",
			backend.DB_PATH, err.Error())
		os.Exit(1)
	}

	if gen_cnt > 0 {
		if gen, err = backend.CreateGenerator(gen_cnt); err != nil {
			log.Printf("Error creating HostGenerator: %s\n", err.Error())
			os.Exit(1)
		} else {
			gen.Start()
			if backend.DEBUG {
				log.Printf("Started generator with %d workers.\n", gen_cnt)
			}
		}
	}

	if xfr_worker_cnt > 0 {
		xfr_queue = make(chan string, xfr_worker_cnt)
		for i := 0; i < xfr_worker_cnt; i++ {
			if _, err = backend.MakeXFRClient(xfr_queue); err != nil {
				log.Printf("Error creating XFRClient: %s\n", err.Error())
				os.Exit(1)
			}
		}

		if backend.DEBUG {
			log.Printf("Started %d XFR workers.\n", xfr_worker_cnt)
		}
	}

	for {
		host := <-gen.HostQueue

		if backend.DEBUG {
			log.Printf("Got host %s/%s from generator queue.\n",
				host.Name, host.Address)
		}

		if err = db.HostAdd(&host); err != nil {
			fmt.Printf("Error adding host %s/%s to database: %s",
				host.Name, host.Address, err.Error())
		} else {
			xfr_queue <- host.Name
		}
	}
} // func main()
