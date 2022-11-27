// /home/krylon/go/src/github.com/blicero/guang/generator/cache_bbolt.go
// -*- mode: go; coding: utf-8; -*-
// Created on 25. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-27 00:21:12 krylon>

package generator

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/blicero/guang/common"
	"github.com/blicero/krylib"
	"go.etcd.io/bbolt"
)

// Re-Implementation of the host cache, but with bbolt.

var openLock sync.Mutex

type bboltCache struct {
	path  string
	log   *log.Logger
	cache *bbolt.DB
}

func openBoltCache(path string) (cache, error) { // nolint: unused
	var (
		err    error
		exists bool
		opt    bbolt.Options
		c      = &bboltCache{path: path}
	)

	path = path + ".blt"

	openLock.Lock()
	defer openLock.Unlock()

	if exists, err = krylib.Fexists(path); err != nil {
		return nil, err
	}

	opt = bbolt.Options{
		Timeout:      time.Second * 90,
		FreelistType: "hashmap",
	}

	if c.log, err = common.GetLogger("Generator/Cache"); err != nil {
		return nil, err
	} else if c.cache, err = bbolt.Open(path, 0600, &opt); err != nil {
		c.log.Printf("[ERROR] Cannot open host cache at %q: %s\n",
			path,
			err.Error())
		return nil, err
	}

	if !exists {
		c.log.Printf("[INFO] Initializing host cache at %s\n",
			path)

		err = c.cache.Update(func(tx *bbolt.Tx) error {
			var x error
			if _, x = tx.CreateBucket(bs("ip")); x != nil {
				c.log.Printf("[ERROR] Cannot create bucket ip: %s\n",
					x.Error())
				return x
			} else if _, x = tx.CreateBucket(bs("name")); x != nil {
				c.log.Printf("[ERROR] Cannot create bucket name: %s\n",
					x.Error())
				return x
			}

			return nil
		})

		if err != nil {
			c.log.Printf("[ERROR] Failed to initialize host cache: %s\n",
				err.Error())
			return nil, err
		}
	}

	return c, nil
} // func openBoltCache(path string) (*bboltCache, error)

func (c *bboltCache) HasKey(s string) (bool, error) {
	var (
		err    error
		exists bool
	)

	err = c.cache.View(func(tx *bbolt.Tx) error {
		var (
			x      error
			bucket *bbolt.Bucket
		)

		if bucket = tx.Bucket(bs("ip")); bucket == nil {
			return fmt.Errorf("Did not find bucket 'ip' in cache, to check address %q",
				s)
		} else if r := bucket.Get(bs(s)); r != nil {
			exists = true
		}

		return x
	})

	return exists, err
} // func (c *bboltCache) HasKey(s string) (bool, error)

func (c *bboltCache) AddKey(s string) error {
	return c.cache.Update(func(tx *bbolt.Tx) error {
		var (
			x error
			b *bbolt.Bucket
		)

		if b = tx.Bucket(bs("ip")); b == nil {
			x = fmt.Errorf("Did not find Bucket 'ip' to add key %q",
				s)
		} else if x = b.Put(bs(s), bs("1")); x != nil {
			c.log.Printf("[ERROR] Cannot add IP %q to host cache: %s\n",
				s,
				x.Error())
		}

		return x
	})
} // func (c *bboltCache) AddKey(s string) error

func bs(s string) []byte {
	return []byte(s)
}
