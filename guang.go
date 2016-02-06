// /Users/krylon/go/src/guang/guang.go
// -*- coding: utf-8; mode: go; -*-
// Created on 27. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2016-01-09 14:37:34 krylon>

package main

import (
	"flag"
	"fmt"
	"guang/backend"
	"log"
	"os"
	"path/filepath"
	"time"

	"net/http"
	_ "net/http/pprof"
)

func main() {
	var gen_cnt int = 16
	var xfr_worker_cnt int = 2
	var scanner_cnt int = 4
	var do_profile bool = false
	var err error
	var mlog *log.Logger
	var gen *backend.HostGenerator
	var xfr *backend.XFRClient
	var xfr_queue chan string
	var db *backend.HostDB
	var do_xfr bool
	var scanner *backend.Scanner

	flag.IntVar(&gen_cnt, "generators", gen_cnt, "Number of Host Generators to run")
	flag.IntVar(&xfr_worker_cnt, "xfr", xfr_worker_cnt, "Number of XFR workers to run")
	flag.IntVar(&scanner_cnt, "scanner", scanner_cnt, "Number of scanner workers to run")
	flag.BoolVar(&do_profile, "profile", do_profile, "Run the builtin profiling server")

	flag.Parse()

	if gen_cnt == 0 && xfr_worker_cnt == 0 && scanner_cnt == 0 {
		fmt.Println("Alrighty then!")
		os.Exit(0)
	}

	// Freitag, 08. 01. 2016, 22:39
	// At some point in the future, we are going to have a web interface,
	// in that case, we can ditch this.
	if do_profile {
		go func() {
			log.Println(http.ListenAndServe(":7998", nil))
		}()
	}

	base_dir := filepath.Join(os.Getenv("HOME"), ".guang")
	backend.SetBaseDir(base_dir)

	if mlog, err = backend.GetLogger("MAIN"); err != nil {
		fmt.Printf("Error creating Logger instance: %s\n",
			err.Error())
		os.Exit(1)
	}

	if db, err = backend.OpenDB(backend.DB_PATH); err != nil {
		mlog.Printf("Error opening database at %s: %s\n",
			backend.DB_PATH, err.Error())
		os.Exit(1)
	}

	if gen_cnt > 0 {
		if gen, err = backend.CreateGenerator(gen_cnt); err != nil {
			mlog.Printf("Error creating HostGenerator: %s\n", err.Error())
			os.Exit(1)
		} else {
			gen.Start()
			if backend.DEBUG {
				mlog.Printf("Started generator with %d workers.\n", gen_cnt)
			}
		}

		go func() {
			for {
				var host_present bool

				host := <-gen.HostQueue

				if backend.DEBUG {
					mlog.Printf("Got host %s/%s from generator queue.\n",
						host.Name, host.Address)
				}

				if host_present, err = db.HostExists(host.Address.String()); err != nil {
					fmt.Printf("Error checking if host %s exists: %s\n",
						host.Address.String(), err.Error())
				} else if host_present {
					continue
				} else if err = db.HostAdd(&host); err != nil {
					fmt.Printf("Error adding host %s/%s to database: %s",
						host.Name, host.Address, err.Error())
				} else if do_xfr {
					xfr_queue <- host.Name
				}
			}
		}()
	}

	if xfr_worker_cnt > 0 {
		do_xfr = true
		xfr_queue = make(chan string, xfr_worker_cnt)

		if xfr, err = backend.MakeXFRClient(xfr_queue); err != nil {
			mlog.Printf("Error creating XFR client: %s\n", err.Error())
			os.Exit(1)
		} else {
			xfr.Start(xfr_worker_cnt)
		}

		if backend.DEBUG {
			mlog.Printf("Started %d XFR workers.\n", xfr_worker_cnt)
		}
	}

	if scanner_cnt > 0 {
		if scanner, err = backend.CreateScanner(scanner_cnt); err != nil {
			mlog.Printf("Error creating scanner with %d workers: %s\n",
				scanner_cnt, err.Error())
			os.Exit(1)
		} else {
			scanner.Start()
			go scanner.Loop()
		}
	}

	// Once I got the web frontend more or less working, I am going to run
	// the web server here.
	for {
		time.Sleep(time.Second * 10)
	}

} // func main()
