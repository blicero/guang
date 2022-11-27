// /home/krylon/go/src/github.com/blicero/guang/generator/cache.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-26 23:15:49 krylon>

package generator

type cache interface { // nolint: unused
	HasKey(s string) (bool, error)
	AddKey(s string) error
}

type cacheOpener func(string) (cache, error)
