// /Users/krylon/go/src/guang/guang.go
// -*- coding: utf-8; mode: go; -*-
// Created on 27. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-30 21:38:31 krylon>

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/blicero/guang/backend"
	"github.com/blicero/guang/common"
	"github.com/blicero/guang/database"
	"github.com/blicero/guang/frontend"
	"github.com/blicero/guang/generator"
	"github.com/blicero/guang/xfr"

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
	var gen *generator.HostGenerator
	var xfrClient *xfr.Client
	var xfr_queue chan string
	var db *database.HostDB
	var do_xfr bool
	var scanner *backend.Scanner
	var webserver *frontend.WebFrontend
	var port int = 4711
	var nexus *backend.Nexus
	var base_dir string = common.BaseDir

	fmt.Printf("%s %s - built on %s\n",
		common.AppName,
		common.Version,
		common.BuildStamp.Format(common.TimeFormat))

	flag.IntVar(&gen_cnt, "generators", gen_cnt, "Number of Host Generators to run")
	flag.IntVar(&xfr_worker_cnt, "xfr", xfr_worker_cnt, "Number of XFR workers to run")
	flag.IntVar(&scanner_cnt, "scanner", scanner_cnt, "Number of scanner workers to run")
	flag.BoolVar(&do_profile, "profile", do_profile, "Run the builtin profiling server")
	flag.IntVar(&port, "port", port, "Port for the web server to listen on")
	flag.StringVar(&base_dir, "basedir", common.BaseDir, "Base directory for application-specific files")

	flag.Parse()

	if gen_cnt == 0 && xfr_worker_cnt == 0 && scanner_cnt == 0 {
		fmt.Println("Alrighty then!")
		os.Exit(0)
	} else if port < 0 || port > 65535 {
		fmt.Printf("Port for web server is not in the valid range (0 - 65535): %d\n", port)
		os.Exit(1)
	}

	// Freitag, 08. 01. 2016, 22:39
	// At some point in the future, we are going to have a web interface,
	// in that case, we can ditch this.
	if do_profile {
		go func() {
			log.Println(http.ListenAndServe(":7998", nil))
		}()
	}

	//base_dir := filepath.Join(os.Getenv("HOME"), "guang.d")
	if base_dir != common.BaseDir {
		common.SetBaseDir(base_dir)
	}

	if mlog, err = common.GetLogger("MAIN"); err != nil {
		fmt.Printf("Error creating Logger instance: %s\n",
			err.Error())
		os.Exit(1)
	}

	if db, err = database.OpenDB(common.DbPath); err != nil {
		mlog.Printf("Error opening database at %s: %s\n",
			common.DbPath, err.Error())
		os.Exit(1)
	}

	if gen_cnt > 0 {
		if gen, err = generator.CreateGenerator(gen_cnt); err != nil {
			mlog.Printf("Error creating HostGenerator: %s\n", err.Error())
			os.Exit(1)
		} else {
			gen.Start()
			if common.Debug {
				mlog.Printf("Started generator with %d workers.\n", gen_cnt)
			}
		}

		go func() {
			for {
				var host_present bool

				host := <-gen.HostQueue

				if common.Debug {
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

		if xfrClient, err = xfr.MakeXFRClient(xfr_queue); err != nil {
			mlog.Printf("Error creating XFR client: %s\n", err.Error())
			os.Exit(1)
		} else {
			xfrClient.Start(xfr_worker_cnt)
		}

		if common.Debug {
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

	if port == 0 {
		// Once I got the web frontend more or less working, I am going to run
		// the web server here.
		for {
			time.Sleep(time.Second * 10)
		}
	} else if nexus, err = backend.CreateNexus(gen, scanner, xfrClient); err != nil {
		fmt.Printf("Error creating Nexus: %s\n", err.Error())
		os.Exit(1)
	} else {
		webserver, err = frontend.CreateFrontend("0.0.0.0", uint16(port), nexus)
		webserver.Serve()
	}

} // func main()
