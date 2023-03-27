// /home/krylon/go/src/github.com/blicero/guang/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2023-03-27 09:36:26 krylon>

package database

var initQueries = []string{
	`
CREATE TABLE host (
    id INTEGER PRIMARY KEY,
    addr TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    source INTEGER NOT NULL,
    add_stamp INTEGER NOT NULL,
    location TEXT,
    os TEXT
)`,
	"CREATE INDEX host_addr_idx ON host (addr)",
	"CREATE INDEX host_name_idx ON host (name)",
	"CREATE INDEX host_stamp_idx ON host (add_stamp)",
	"CREATE INDEX host_location_idx ON host (location)",
	"CREATE INDEX host_os_stamp ON host (os)",
	`
CREATE TABLE port (
    id INTEGER PRIMARY KEY,
    host_id INTEGER NOT NULL,
    port INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    reply TEXT,
    UNIQUE (host_id, port),
    FOREIGN KEY (host_id) REFERENCES host (id))`,
	"CREATE INDEX port_host_idx ON port (host_id)",
	"CREATE INDEX port_port_idx ON port (port)",
	"CREATE INDEX port_ts_idx   ON port (timestamp)",

	`
CREATE TABLE xfr (
    id INTEGER PRIMARY KEY,
    zone TEXT UNIQUE NOT NULL,
    start INTEGER NOT NULL,
    end INTEGER NOT NULL DEFAULT 0,
    status INTEGER NOT NULL DEFAULT 0
)`,
	"CREATE INDEX xfr_zone_idx ON xfr (zone)",
	"CREATE INDEX xfr_status_idx ON xfr (status)",
}
