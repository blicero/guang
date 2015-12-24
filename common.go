// /Users/krylon/go/src/guangv2/common.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-24 23:52:55 krylon>

package guang

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

//var DEBUG bool = true

const (
	DEBUG    bool = true
	VERSION       = "0.0.1"
	APP_NAME      = "Guang"
)

var BASE_DIR string = filepath.Join(os.Getenv("HOME"), ".guang")

var LOG_PATH = filepath.Join(BASE_DIR, "guang.log")
var DB_PATH = filepath.Join(BASE_DIR, "guang.db")
var HOST_CACHE_PATH = filepath.Join(BASE_DIR, "ip_cache.kch")

func SetBaseDir(path string) {
	// if isdir, _ := krylib.IsDir(path); !isdir {
	// 	fmt.Printf("Error setting BASE_DIR to %s - folder does not exist!\n",
	// 		path)
	// 	return
	// }

	fmt.Printf("Setting BASE_DIR to %s\n", path)

	BASE_DIR = path
	LOG_PATH = filepath.Join(BASE_DIR, "guang.log")
	DB_PATH = filepath.Join(BASE_DIR, "guang.db")
	HOST_CACHE_PATH = filepath.Join(BASE_DIR, "ip_cache.kch")

	InitApp()
} // func SetBaseDir(path string)

var log_cache map[string]*log.Logger
var log_lock sync.Mutex

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

	log_lock.Lock()
	defer log_lock.Unlock()
	if logger, ok := log_cache[log_name]; ok {
		return logger, nil
	}

	var logfile *os.File
	logfile, err = os.OpenFile(LOG_PATH, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		msg := fmt.Sprintf("Error opening log file: %s\n", err.Error())
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	writer := io.MultiWriter(os.Stdout, logfile)

	logger := log.New(writer, log_name, log.Ldate|log.Ltime|log.Lshortfile)
	log_cache[log_name] = logger
	return logger, nil
} // func GetLogger(name string) (*log.logger, error)

// Perform some basic preparations for the application to run.
// Currently, this means creating the BASE_DIR folder.
func InitApp() error {
	if log_cache == nil {
		log_cache = make(map[string]*log.Logger)
	}

	err := os.Mkdir(BASE_DIR, 0755)
	if err != nil {
		if !os.IsExist(err) {
			msg := fmt.Sprintf("Error creating BASE_DIR %s: %s", BASE_DIR, err.Error())
			return errors.New(msg)
		}
	}

	return nil
} // func InitApp() error
