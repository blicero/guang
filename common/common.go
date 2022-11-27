// /Users/krylon/go/src/guangv2/common.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-11-27 00:25:09 krylon>

// Package common provides constants, variables and functions used
// throughout the application.
package common

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/blicero/guang/logdomain"
	"github.com/hashicorp/logutils"
	"github.com/odeke-em/go-uuid"
)

//var Debug bool = true

//go:generate ./build_time_stamp.pl

// Debug indicated whether to emit additional log messages and perform
// additional sanity checks.
// Version is the version number to display.
// AppName is the name of the application.
// TimestampFormat is the format string used to render datetime values.
// HeartBeat is the interval for worker goroutines to wake up and check
// their status.
const (
	Debug                    = true
	Version                  = "0.4.0"
	AppName                  = "Guang"
	TimestampFormat          = "2006-01-02 15:04:05"
	TimestampFormatMinute    = "2006-01-02 15:04"
	TimestampFormatSubSecond = "2006-01-02 15:04:05.0000 MST"
	TimestampFormatDate      = "2006-01-02"
	HeartBeat                = time.Millisecond * 500
	RCTimeout                = time.Millisecond * 10
)

// LogLevels are the names of the log levels supported by the logger.
var LogLevels = []logutils.LogLevel{
	"TRACE",
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
	"CRITICAL",
	"CANTHAPPEN",
	"SILENT",
}

// PackageLevels defines minimum log levels per package.
var PackageLevels = make(map[logdomain.ID]logutils.LogLevel, len(LogLevels))

const MinLogLevel = "TRACE"

func init() {
	for _, id := range logdomain.AllDomains() {
		PackageLevels[id] = MinLogLevel
	}
} // func init()

// BaseDir is the folder where all application-specific files (database,
// log files, etc) are stored.
// LogPath is the file to the log path.
// DbPath is the path of the main database.
// HostCachePath is the path to the IP cache.
// XfrDbgPath is the path of the folder where data on DNS zone transfers
// are stored.
var (
	BaseDir       = filepath.Join(os.Getenv("HOME"), "guang.d")
	LogPath       = filepath.Join(BaseDir, "guang.log")
	DbPath        = filepath.Join(BaseDir, "guang.db")
	HostCachePath = filepath.Join(BaseDir, "ip_cache")
	XfrDbgPath    = filepath.Join(BaseDir, "xfr")
)

// SetBaseDir sets the BaseDir and related variables.
func SetBaseDir(path string) {
	fmt.Printf("Setting BASE_DIR to %s\n", path)

	BaseDir = path
	LogPath = filepath.Join(BaseDir, "guang.log")
	DbPath = filepath.Join(BaseDir, "guang.db")
	HostCachePath = filepath.Join(BaseDir, "ip_cache.kch")
	XfrDbgPath = filepath.Join(BaseDir, "xfr")

	if err := InitApp(); err != nil {
		fmt.Printf("Error initializing application environment: %s\n", err.Error())
	}
} // func SetBaseDir(path string)

// GetLogger Tries to create a named logger instance and return it.
// If the directory to hold the log file does not exist, try to create it.
func GetLogger(name string) (*log.Logger, error) {
	var err error
	err = InitApp()
	if err != nil {
		return nil, fmt.Errorf("Error initializing application environment: %s", err.Error())
	}

	logName := fmt.Sprintf("%s.%s",
		AppName,
		name)

	var logfile *os.File
	logfile, err = os.OpenFile(LogPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		msg := fmt.Sprintf("Error opening log file: %s\n", err.Error())
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	writer := io.MultiWriter(os.Stdout, logfile)

	logger := log.New(writer, logName, log.Ldate|log.Ltime|log.Lshortfile)
	return logger, nil
} // func GetLogger(name string) (*log.logger, error)

// InitApp performs some basic preparations for the application to run.
// Currently, this means creating the BASE_DIR folder.
func InitApp() error {
	err := os.Mkdir(BaseDir, 0755)
	if err != nil {
		if !os.IsExist(err) {
			msg := fmt.Sprintf("Error creating BASE_DIR %s: %s", BaseDir, err.Error())
			return errors.New(msg)
		}
	} else if err = os.Mkdir(XfrDbgPath, 0755); err != nil {
		if !os.IsExist(err) {
			msg := fmt.Sprintf("Error creating XFR_DBG_PATH %s: %s",
				XfrDbgPath, err.Error())
			return errors.New(msg)
		}
	}

	return nil
} // func InitApp() error

// GetUUID returns a randomized UUID
func GetUUID() string {
	return uuid.NewRandom().String()
} // func GetUUID() string

// TimeEqual returns true if the two timestamps are less than one second apart.
func TimeEqual(t1, t2 time.Time) bool {
	var delta = t1.Sub(t2)

	if delta < 0 {
		delta = -delta
	}

	return delta < time.Second
} // func TimeEqual(t1, t2 time.Time) bool

// GetChecksum computes the SHA512 checksum of the given data.
func GetChecksum(data []byte) (string, error) {
	var err error
	var hash = sha512.New()

	if _, err = hash.Write(data); err != nil {
		fmt.Fprintf( // nolint: errcheck
			os.Stderr,
			"Error computing checksum: %s\n",
			err.Error(),
		)
		return "", err
	}

	var checkSumBinary = hash.Sum(nil)
	var checkSumText = fmt.Sprintf("%x", checkSumBinary)

	return checkSumText, nil
} // func getChecksum(data []byte) (string, error)
