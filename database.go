// /Users/krylon/go/src/guang/database.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2015-12-23 16:45:26 krylon>

package guang

import "sync"

var open_lock sync.Mutex

var init_query []string = []string{
	`
CREATE TABLE host (id INTEGER PRIMARY KEY,
                   addr TEXT UNIQUE NOT NULL,
                   name TEXT NOT NULL,
                   source INTEGER NOT NULL,
                   add_stamp INTEGER NOT NULL)
`,

	`
CREATE TABLE port (id INTEGER PRIMARY KEY,
                   host_id INTEGER NOT NULL,
                   port INTEGER NOT NULL,
                   timestamp INTEGER NOT NULL,
                   reply TEXT,
                   UNIQUE (host_id, port),
                   FOREIGN KEY (host_id) REFERENCES host (id))
`,
}
