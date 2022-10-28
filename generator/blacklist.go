// /Users/krylon/go/src/guang/blacklist.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-28 22:33:46 krylon>

package generator

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

type NameBlacklistItem struct {
	Pattern *regexp.Regexp
	Cnt     int64
}

//type NameBlacklist []NameBlacklistItem
type NameBlacklist struct {
	blacklist []NameBlacklistItem
	lock      sync.Mutex
}

func (self *NameBlacklist) Len() int {
	return len(self.blacklist)
}

func (self *NameBlacklist) Swap(a, b int) {
	var tmp NameBlacklistItem = self.blacklist[a]
	self.blacklist[a] = self.blacklist[b]
	self.blacklist[b] = tmp
}

func (self *NameBlacklist) Less(a, b int) bool {
	return self.blacklist[b].Cnt < self.blacklist[a].Cnt
}

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

func (self *NameBlacklist) Matches(x string) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	for idx, item := range self.blacklist {
		if item.Pattern.Match([]byte(x)) {
			self.blacklist[idx].Cnt++
			sort.Sort(self)
			return true
		}
	}
	return false
} // func (self NameBlacklist) Matches(x string) bool

// IP blacklist

type IPBlacklistItem struct {
	Network *net.IPNet
	Cnt     int
}

//type IPBlacklist []IPBlacklistItem
type IPBlacklist struct {
	blacklist []IPBlacklistItem
	lock      sync.Mutex
}

func (self *IPBlacklist) Len() int {
	return len(self.blacklist)
}

func (self *IPBlacklist) Swap(a, b int) {
	self.blacklist[a], self.blacklist[b] = self.blacklist[b], self.blacklist[a]
}

func (self *IPBlacklist) Less(a, b int) bool {
	return self.blacklist[b].Cnt < self.blacklist[a].Cnt
}

func MakeIPBlacklist(networks []string) (*IPBlacklist, error) {
	bl := &IPBlacklist{
		blacklist: make([]IPBlacklistItem, len(networks)),
	}

	for i, n := range networks {
		_, network, err := net.ParseCIDR(n)
		if err != nil {
			fmt.Printf("Error parsing network %s: %s\n", n, err.Error())
			return nil, err
		} else {
			bl.blacklist[i] = IPBlacklistItem{network, 0}
		}
	}

	return bl, nil
}

func (self *IPBlacklist) Matches(x string) bool {
	addr := net.ParseIP(x)

	self.lock.Lock()
	defer self.lock.Unlock()

	for idx, item := range self.blacklist {
		if item.Network.Contains(addr) {
			self.blacklist[idx].Cnt++
			sort.Sort(self)
			return true
		}
	}

	return false
}

func (self *IPBlacklist) MatchesIP(x net.IP) bool {
	self.lock.Lock()
	defer self.lock.Unlock()

	for idx, item := range self.blacklist {
		if item.Network.Contains(x) {
			self.blacklist[idx].Cnt++
			sort.Sort(self)
			return true
		}
	}

	return false
}

func DefaultNameBlacklist() *NameBlacklist {
	bl, err := MakeNameBlacklist(nameBlacklistPatterns)
	if err != nil {
		panic(fmt.Sprintf("Error compiling name blacklist: %s", err.Error()))
	}

	return bl
}

func DefaultIPBlacklist() *IPBlacklist {
	bl, err := MakeIPBlacklist(reservedNetworks)
	if err != nil {
		panic(fmt.Sprintf("Error compiling IP blacklist: %s", err.Error()))
	}

	return bl
}
