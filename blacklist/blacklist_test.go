// /Users/krylon/go/src/guang/blacklist_test.go
// -*- coding: utf-8; mode: go; -*-
// Created on 21. 06. 2014 by Benjamin Walkenhorst
// (c) 2014 Benjamin Walkenhorst
// Time-stamp: <2022-10-29 18:11:51 krylon>

package blacklist

import (
	"fmt"
	"testing"
)

func TestIPBlacklist(t *testing.T) {
	forbiddenAddresses := []string{
		"127.0.0.1",
		"192.168.0.13",
		"192.168.80.221",
		"224.21.13.91",
		"172.18.41.27",
		"169.254.21.177",
		"203.0.113.113",
		"255.255.255.255",
	}

	allowedAddresses := []string{
		"91.34.78.15",
		"157.21.199.201",
		"101.17.81.34",
		"34.68.136.11",
		"1.2.3.4",
	}

	fmt.Println("Test IP Blacklist...")

	bl := DefaultIPBlacklist()
	var msg string

	for _, addr := range forbiddenAddresses {
		if !bl.Matches(addr) {
			msg = fmt.Sprintf("Blacklist did not match forbidden address %s!", addr)
			t.Error(msg)
		}
	}

	for _, addr := range allowedAddresses {
		if bl.Matches(addr) {
			msg = fmt.Sprintf("Blacklist DID match allowed address %s!", addr)
			t.Error(msg)
		}
	}
} // func TestIPBlacklist(t *testing.T)

func TestNameBlacklist(t *testing.T) {
	forbiddenAddresses := []string{
		"23.invalid.addr.",
		"incorrect.domain.com",
		"abef.pool.heise.de",
		"noname.wellmann-anlagentechnik.de",
		"ppp2391.bitel.de",
		"host231.greenpeace.org",
		"this.ip",
		"internal-host91.some-isp.com",
		"dyn81.t-online.de",
	}

	allowedAddresses := []string{
		"www.heise.de",
		"www02.google.de",
		"srv03.webapp.some-company.de",
		"download.microsoft.com",
		"mx03.1und1.de",
		"wifi81.uni-kassel.de",
		"www.thepiratebay.se",
		"db23.cluster.dec.com",
	}

	fmt.Println("Test Name Blacklist...")

	bl := DefaultNameBlacklist()
	var msg string

	for _, name := range forbiddenAddresses {
		if !bl.Matches(name) {
			msg = fmt.Sprintf("Blacklist did NOT match forbidden name %s!", name)
			t.Error(msg)
		}
	}

	for _, name := range allowedAddresses {
		if bl.Matches(name) {
			msg = fmt.Sprintf("Blacklist DID match allowed name %s!", name)
			t.Error(msg)
		}
	}
} // func TestNameBlacklist(t *testing.T)
