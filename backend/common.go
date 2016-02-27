// /Users/krylon/go/src/guangv2/common.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2016-02-13 14:47:23 krylon>

package backend

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

//var DEBUG bool = true

const (
	DEBUG    bool = true
	VERSION       = "0.0.1"
	APP_NAME      = "Guang"
)

var BASE_DIR string = filepath.Join(os.Getenv("HOME"), "guang.d")

var LOG_PATH = filepath.Join(BASE_DIR, "guang.log")
var DB_PATH = filepath.Join(BASE_DIR, "guang.db")
var HOST_CACHE_PATH = filepath.Join(BASE_DIR, "ip_cache.kch")
var XFR_DBG_PATH = filepath.Join(BASE_DIR, "xfr")

func SetBaseDir(path string) {
	fmt.Printf("Setting BASE_DIR to %s\n", path)

	BASE_DIR = path
	LOG_PATH = filepath.Join(BASE_DIR, "guang.log")
	DB_PATH = filepath.Join(BASE_DIR, "guang.db")
	HOST_CACHE_PATH = filepath.Join(BASE_DIR, "ip_cache.kch")
	XFR_DBG_PATH = filepath.Join(BASE_DIR, "xfr")

	if err := InitApp(); err != nil {
		fmt.Printf("Error initializing application environment: %s\n", err.Error())
	}
} // func SetBaseDir(path string)

// Try to create a named logger instance and return it.
// If the directory to hold the log file does not exist, try to create it.
func GetLogger(name string) (*log.Logger, error) {
	var err error
	err = InitApp()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error initializing application environment: %s", err.Error()))
	}

	log_name := fmt.Sprintf("%s.%s",
		APP_NAME,
		name)

	var logfile *os.File
	logfile, err = os.OpenFile(LOG_PATH, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		msg := fmt.Sprintf("Error opening log file: %s\n", err.Error())
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	writer := io.MultiWriter(os.Stdout, logfile)

	logger := log.New(writer, log_name, log.Ldate|log.Ltime|log.Lshortfile)
	return logger, nil
} // func GetLogger(name string) (*log.logger, error)

// Perform some basic preparations for the application to run.
// Currently, this means creating the BASE_DIR folder.
func InitApp() error {
	err := os.Mkdir(BASE_DIR, 0755)
	if err != nil {
		if !os.IsExist(err) {
			msg := fmt.Sprintf("Error creating BASE_DIR %s: %s", BASE_DIR, err.Error())
			return errors.New(msg)
		}
	} else if err = os.Mkdir(XFR_DBG_PATH, 0755); err != nil {
		if !os.IsExist(err) {
			msg := fmt.Sprintf("Error creating XFR_DBG_PATH %s: %s",
				XFR_DBG_PATH, err.Error())
			return errors.New(msg)
		}
	}

	return nil
} // func InitApp() error
