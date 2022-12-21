// /home/krylon/go/src/github.com/blicero/guang/generator/cache_kyoto.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-12-21 21:13:12 krylon>

// +build ignore

package generator

import (
	"log"

	"github.com/blicero/guang/common"
	"github.com/fsouza/gokabinet/kc"
)

type kyotoCache struct {
	path  string
	log   *log.Logger
	cache *kc.DB
}

func openKyotoCache(path string) (cache, error) {
	var (
		err     error
		kyCache = &kyotoCache{path: path}
	)

	if kyCache.log, err = common.GetLogger("Generator/Cache"); err != nil {
		return nil, err
	} else if kyCache.cache, err = kc.Open(path, kc.WRITE); err != nil {
		kyCache.log.Printf("[CRITICAL] Cannot open cache at %s: %s\n",
			path,
			err.Error())
		return nil, err
	}

	return kyCache, nil
} // func openKyotoCache(path string) (*kyotoCache, error)

func (c *kyotoCache) HasKey(s string) (bool, error) {
	var (
		err error
		res bool
		cnt int
	)

	// ...
	if cnt, err = c.cache.GetInt(s); err != nil {
		return false, err
	}

	res = cnt != 0

	return res, err
} // func (c *kyotoCache) HasKey(s string) (bool, error)

func (c *kyotoCache) AddKey(s string) error {
	var (
		err error
	)

	if err = c.cache.SetInt(s, 1); err != nil {
		c.log.Printf("[ERROR] Cannot add key %q to database: %s\n",
			s,
			err.Error())
	} else {
		c.cache.Commit() // nolint: errcheck
	}

	return err
} // func (c *kyotoCache) AddKey(s string) error
