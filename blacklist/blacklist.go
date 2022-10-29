// /Users/krylon/go/src/guang/blacklist.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-29 18:11:59 krylon>

package blacklist

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"sync"
)

var reservedNetworks = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	//"224.0.0.0/3",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.0.0.0/8",
}

var nameBlacklistPatterns = []string{
	"\\bdiu?p-?\\d*\\.",
	"(?:versanet|telekom|uni-paderborn|upb)\\.(?:de|net|com|biz|eu)\\.?$",
	"[.]?nothing[.]",
	"[.]example[.](?:org|net|com)[.]?$",
	"[avs]?dsl",
	"\\.in-addr\\.",
	"\\.invalid\\.?",
	"\\b(?:wireless|wlan|wimax|wan|vpn|vlan)",
	"\\b\\d{1,3}.\\d{1,3}.\\d{1,3}.\\d{1,3}\\b",
	"\\bincorrect(?:ly)?\\b",
	"\\bnot.configured\\b",
	"\\bpools?\\b",
	"\\bunn?ass?igned\\b",
	"^(?:client|host)(?:-?\\d+)?",
	"^(?:un|not-)(?:known|ass?igned|alloc(?:ated)?|registered|provisioned|used|defined|delegated)",
	"^[^.]+.$",
	"^\\.$",
	"^\\*",
	"^\\w*eth(?:ernet)[^.]*\\.",
	"^\\w\\d+\\[\\-.]",
	"^customer-",
	"^customer\\.",
	"^dyn\\d+",
	"^generic-?host",
	"^h\\d+s\\d+",
	"^host\\d+\\.",
	"^illegal",
	"^internal-host",
	"^ip(?:-?\\d+|addr)",
	"^mobile",
	"^no(?:-reverse)?-dns",
	"^(?:no-?)?reverse",
	"^no.ptr",
	"^softbank\\d+\\.bbtec",
	"^this.ip",
	"^user-?\\d+\\.",
	"aol\\.com\\.?$",
	"cable",
	"dhcp",
	"dial-?(?:in|up)?",
	"dyn(?:amic)?[-.0-9]",
	"dyn(?:amic)ip",
	"early.registration",
	"(?:edu)?roam",
	"localhost",
	"myvzw\\.com",
	"no-dns(?:-yet)?",
	"non-routed",
	"ppp",
	"rr\\.com\\.?$",
	"umts",
	"wanadoo\\.[a-z]{2,3}\\.?$",
	"^\\w*[.]$",
	"reverse-not-set",
	"uu[.]net[.]?$",
	"(?:ne|ad)[.]jp[.]?$",
	"[.](?:cn|mil)[.]?$",
	"^noname[.]",
}

// type blacklist interface {
// 	Matches(x string) bool
// }

// NameBlacklistItem is a blacklist item that matches hostnames.
type NameBlacklistItem struct {
	Pattern *regexp.Regexp
	Cnt     int64
}

//NameBlacklist is a blacklist that matches host names against a list
//of NameBlacklistItems.
type NameBlacklist struct {
	blacklist []NameBlacklistItem
	lock      sync.Mutex
}

func (bl *NameBlacklist) Len() int {
	return len(bl.blacklist)
}

func (bl *NameBlacklist) Swap(a, b int) {
	var tmp NameBlacklistItem = bl.blacklist[a]
	bl.blacklist[a] = bl.blacklist[b]
	bl.blacklist[b] = tmp
}

func (bl *NameBlacklist) Less(a, b int) bool {
	return bl.blacklist[b].Cnt < bl.blacklist[a].Cnt
}

// MakeNameBlacklist creates a new NameBlacklist from the list of patterns.
func MakeNameBlacklist(patterns []string) (*NameBlacklist, error) {
	bl := &NameBlacklist{
		blacklist: make([]NameBlacklistItem, len(patterns)),
	}

	for i, s := range patterns {
		re, err := regexp.Compile("(?i)" + s)
		if err != nil {
			fmt.Println("Error compiling blacklist pattern: ", err.Error())
			return nil, err
		}
		bl.blacklist[i] = NameBlacklistItem{re, 0}
	}

	return bl, nil
} // func MakeNameBlacklist(patterns []string) (*NameBlacklist, error)

// Matches returns true if the given string matches any of the patterns
// in the blacklist.
func (bl *NameBlacklist) Matches(x string) bool {
	bl.lock.Lock()
	defer bl.lock.Unlock()
	for idx, item := range bl.blacklist {
		if item.Pattern.Match([]byte(x)) {
			bl.blacklist[idx].Cnt++
			sort.Sort(bl)
			return true
		}
	}
	return false
} // func (bl NameBlacklist) Matches(x string) bool

// IP blacklist

// IPBlacklistItem is a blacklist item that matches IP addresses against
// a network.
type IPBlacklistItem struct {
	Network *net.IPNet
	Cnt     int
}

// IPBlacklist is a blacklist that matches IP addresses against a list
// of networks.
type IPBlacklist struct {
	blacklist []IPBlacklistItem
	lock      sync.Mutex
}

func (bl *IPBlacklist) Len() int {
	return len(bl.blacklist)
}

func (bl *IPBlacklist) Swap(a, b int) {
	bl.blacklist[a], bl.blacklist[b] = bl.blacklist[b], bl.blacklist[a]
}

func (bl *IPBlacklist) Less(a, b int) bool {
	return bl.blacklist[b].Cnt < bl.blacklist[a].Cnt
}

// MakeIPBlacklist creates an IPBlacklist from the list of networks
// given in CIDR notation.
func MakeIPBlacklist(networks []string) (*IPBlacklist, error) {
	bl := &IPBlacklist{
		blacklist: make([]IPBlacklistItem, len(networks)),
	}

	for i, n := range networks {
		_, network, err := net.ParseCIDR(n)
		if err != nil {
			fmt.Printf("Error parsing network %s: %s\n", n, err.Error())
			return nil, err
		}

		bl.blacklist[i] = IPBlacklistItem{network, 0}
	}

	return bl, nil
}

// Matches returns true if the given IP address is a member of any of
// the networks in the blacklist.
func (bl *IPBlacklist) Matches(x string) bool {
	addr := net.ParseIP(x)

	bl.lock.Lock()
	defer bl.lock.Unlock()

	for idx, item := range bl.blacklist {
		if item.Network.Contains(addr) {
			bl.blacklist[idx].Cnt++
			sort.Sort(bl)
			return true
		}
	}

	return false
} // func (bl *IPBlacklist) Matches(x string) bool

// MatchesIP returns true if the given IP address is a member of any of
// the networks in the blacklist.
func (bl *IPBlacklist) MatchesIP(x net.IP) bool {
	bl.lock.Lock()
	defer bl.lock.Unlock()

	for idx, item := range bl.blacklist {
		if item.Network.Contains(x) {
			bl.blacklist[idx].Cnt++
			sort.Sort(bl)
			return true
		}
	}

	return false
} // func (bl *IPBlacklist) MatchesIP(x net.IP) bool

// DefaultNameBlacklist returns a new NameBlacklist created from the
// default list of names.
func DefaultNameBlacklist() *NameBlacklist {
	bl, err := MakeNameBlacklist(nameBlacklistPatterns)
	if err != nil {
		panic(fmt.Sprintf("Error compiling name blacklist: %s", err.Error()))
	}

	return bl
} // func DefaultNameBlacklist() *NameBlacklist

// DefaultIPBlacklist returns a new IPBlacklist created from the list
// of reserved networks.
func DefaultIPBlacklist() *IPBlacklist {
	bl, err := MakeIPBlacklist(reservedNetworks)
	if err != nil {
		panic(fmt.Sprintf("Error compiling IP blacklist: %s", err.Error()))
	}

	return bl
} // func DefaultIPBlacklist() *IPBlacklist
