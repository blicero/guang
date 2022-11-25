// /home/krylon/go/src/github.com/blicero/guang/generator/cache.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-25 01:38:11 krylon>

package generator

type cache interface { // nolint: unused
	hasKey(s string) (bool, error)
	addKey(s string) error
}
