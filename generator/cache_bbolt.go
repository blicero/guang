// /home/krylon/go/src/github.com/blicero/guang/generator/cache_bbolt.go
// -*- mode: go; coding: utf-8; -*-
// Created on 25. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-26 02:02:13 krylon>

package generator

import (
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

func openBoltCache(path string) (*bboltCache, error) {
	var (
		err    error
		exists bool
		opt    bbolt.Options
		c      = &bboltCache{path: path}
	)

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

func (c *bboltCache) hasKey(s string) (bool, error) {
	var (
		err    error
		exists bool
	)

	return exists, err
} // func (c *bboltCache) hasKey(s string) (bool, error)

func bs(s string) []byte {
	return []byte(s)
}
