// /Users/krylon/go/src/guang/guang.go
// -*- coding: utf-8; mode: go; -*-
// Created on 27. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2024-06-03 19:35:49 krylon>

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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
	var (
		genCnt                        int = 16
		xfrCnt                        int = 2
		scanCnt                       int = 4
		doProfile, doXfr, showVersion bool
		err                           error
		mlog                          *log.Logger
		gen                           *generator.HostGenerator
		xfrClient                     *xfr.Client
		xfrQ                          chan string
		db                            *database.HostDB
		scanner                       *backend.Scanner
		webserver                     *frontend.WebFrontend
		port                          int = 4711
		nexus                         *backend.Nexus
		baseDir                       = common.BaseDir
	)

	flag.IntVar(&genCnt, "generator", genCnt, "Number of Host Generators to run")
	flag.IntVar(&xfrCnt, "xfr", xfrCnt, "Number of XFR workers to run")
	flag.IntVar(&scanCnt, "scanner", scanCnt, "Number of scanner workers to run")
	flag.BoolVar(&doProfile, "profile", doProfile, "Run the builtin profiling server")
	flag.IntVar(&port, "port", port, "Port for the web server to listen on")
	flag.StringVar(&baseDir, "basedir", common.BaseDir, "Base directory for application-specific files")
	flag.BoolVar(&showVersion, "version", false, "Show the version number and exit")

	flag.Parse()

	if common.Debug || showVersion {
		fmt.Printf("%s %s - built on %s\n",
			common.AppName,
			common.Version,
			common.BuildStamp.Format(common.TimestampFormat))
	}

	if showVersion {
		os.Exit(0)
	}

	if genCnt == 0 && xfrCnt == 0 && scanCnt == 0 {
		fmt.Println("Alrighty then!")
		os.Exit(0)
	} else if port < 0 || port > 65535 {
		fmt.Printf("Port for web server is not in the valid range (0 - 65535): %d\n", port)
		os.Exit(1)
	}

	// Freitag, 08. 01. 2016, 22:39
	// At some point in the future, we are going to have a web interface,
	// in that case, we can ditch this.
	if doProfile {
		go func() {
			log.Println(http.ListenAndServe(":7998", nil))
		}()
	}

	//base_dir := filepath.Join(os.Getenv("HOME"), "guang.d")
	if baseDir != common.BaseDir {
		common.SetBaseDir(baseDir)
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

	if genCnt > 0 {
		if gen, err = generator.CreateGenerator(genCnt); err != nil {
			mlog.Printf("Error creating HostGenerator: %s\n", err.Error())
			os.Exit(1)
		} else {
			gen.Start()
			if common.Debug {
				mlog.Printf("Started generator with %d workers.\n", genCnt)
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
				} else if doXfr {
					xfrQ <- host.Name
				}
			}
		}()
	}

	if xfrCnt > 0 {
		doXfr = true
		xfrQ = make(chan string, xfrCnt)

		if xfrClient, err = xfr.MakeXFRClient(xfrQ); err != nil {
			mlog.Printf("Error creating XFR client: %s\n", err.Error())
			os.Exit(1)
		} else {
			xfrClient.Start(xfrCnt)
		}

		if common.Debug {
			mlog.Printf("Started %d XFR workers.\n", xfrCnt)
		}
	}

	if scanCnt > 0 {
		if scanner, err = backend.CreateScanner(scanCnt); err != nil {
			mlog.Printf("Error creating scanner with %d workers: %s\n",
				scanCnt, err.Error())
			os.Exit(1)
		} else {
			scanner.Start()
			go scanner.Loop()
		}
	}

	if nexus, err = backend.CreateNexus(gen, scanner, xfrClient); err != nil {
		fmt.Printf("Error creating Nexus: %s\n", err.Error())
		os.Exit(1)
	} else {
		webserver, err = frontend.Create("0.0.0.0", uint16(port), nexus)
		webserver.Serve()
	}

} // func main()
