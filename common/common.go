// /Users/krylon/go/src/guangv2/common.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-30 21:52:15 krylon>

// Package common provides constants, variables and functions used
// throughout the application.
package common

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

//var Debug bool = true

//go:generate ./build_time_stamp.pl

// Debug indicated whether to emit additional log messages and perform
// additional sanity checks.
// Version is the version number to display.
// AppName is the name of the application.
// TimeFormat is the format string used to render datetime values.
const (
	Debug      = true
	Version    = "0.1.0"
	AppName    = "Guang"
	TimeFormat = "2006-01-02 15:04:05"
)

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
	HostCachePath = filepath.Join(BaseDir, "ip_cache.kch")
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
