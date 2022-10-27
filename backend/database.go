// /Users/krylon/go/src/guang/database.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2022-10-27 18:17:56 krylon>
//
// Samstag, 20. 08. 2016, 21:27
// Ich würde für Hosts gern a) anhand der Antworten, die ich erhalte, das
// Betriebssystem ermitteln, und b) anhand der IP-Adresse den ungefähren
// Standort.

package backend

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/blicero/krylib"

	_ "github.com/mattn/go-sqlite3" // Import the database driver
	"github.com/muesli/cache2go"
)

var open_lock sync.Mutex

var init_query []string = []string{
	`
CREATE TABLE host (id INTEGER PRIMARY KEY,
                   addr TEXT UNIQUE NOT NULL,
                   name TEXT NOT NULL,
                   source INTEGER NOT NULL,
                   add_stamp INTEGER NOT NULL)`,

	`
CREATE TABLE port (id INTEGER PRIMARY KEY,
                   host_id INTEGER NOT NULL,
                   port INTEGER NOT NULL,
                   timestamp INTEGER NOT NULL,
                   reply TEXT,
                   UNIQUE (host_id, port),
                   FOREIGN KEY (host_id) REFERENCES host (id))`,

	`
CREATE TABLE xfr (id INTEGER PRIMARY KEY,
                  zone TEXT UNIQUE NOT NULL,
                  start INTEGER NOT NULL,
                  end INTEGER NOT NULL DEFAULT 0,
                  status INTEGER NOT NULL DEFAULT 0)`,

	"CREATE INDEX host_addr_idx ON host (addr)",
	"CREATE INDEX host_name_idx ON host (name)",

	"CREATE INDEX port_host_idx ON port (host_id)",
	"CREATE INDEX port_port_idx ON port (port)",

	"CREATE INDEX xfr_zone_idx ON xfr (zone)",
	"CREATE INDEX xfr_status_idx ON xfr (status)",
}

type QueryID int

const (
	STMT_HOST_ADD QueryID = iota
	STMT_HOST_GET_BY_ID
	STMT_HOST_GET_RANDOM
	STMT_HOST_GET_CNT
	STMT_HOST_EXISTS
	STMT_HOST_PORT_BY_HOST
	STMT_PORT_ADD
	STMT_PORT_GET_BY_HOST
	STMT_PORT_GET_REPLY_CNT
	STMT_PORT_GET_OPEN
	STMT_XFR_ADD
	STMT_XFR_GET_BY_ZONE
	STMT_XFR_FINISH
	STMT_XFR_GET_UNFINISHED
)

var stmt_table map[QueryID]string = map[QueryID]string{
	STMT_HOST_ADD: `
INSERT INTO host (addr, name, source, add_stamp)
          VALUES (   ?,    ?,      ?,         ?)
`,
	STMT_HOST_GET_BY_ID: "SELECT addr, name, source, add_stamp FROM host WHERE id = ?",

	//STMT_HOST_GET_RANDOM: "SELECT id, addr, name, source, add_stamp FROM host WHERE RANDOM() > 8301034833169298432 LIMIT ?",
	STMT_HOST_GET_RANDOM: `
SELECT id,
       addr,
       name,
       source,
       add_stamp
FROM host
LIMIT ?
OFFSET ABS(RANDOM()) % MAX((SELECT COUNT(*) FROM host), 1)
`,

	STMT_HOST_GET_CNT: "SELECT COUNT(id) FROM host",

	STMT_HOST_EXISTS: "SELECT COUNT(id) FROM host WHERE addr = ?",

	STMT_HOST_PORT_BY_HOST: `
SELECT 
  P.id,
  P.host_id,
  P.port,
  P.timestamp,
  P.reply
  H.adddr,
  H.name
FROM port P
INNER JOIN host H ON port.host_id = host.id
WHERE port.reply IS NOT NULL
`,

	STMT_PORT_ADD: `
INSERT INTO port (host_id, port, timestamp, reply)
          VALUES (      ?,    ?,         ?,     ?)
`,

	STMT_PORT_GET_BY_HOST: "SELECT id, port, timestamp, reply FROM port WHERE host_id = ?",

	STMT_XFR_ADD:         "INSERT INTO xfr (zone, start, status) VALUES (?, ?, 0)",
	STMT_XFR_GET_BY_ZONE: "SELECT id, start, end, status FROM xfr WHERE zone = ?",
	STMT_XFR_FINISH:      "UPDATE xfr SET end = ?, status = ? WHERE id = ?",
	STMT_XFR_GET_UNFINISHED: `
SELECT id, 
       zone, 
       start, 
       end, 
       status
FROM xfr
WHERE status = 0
`,
	STMT_PORT_GET_REPLY_CNT: "SELECT COUNT(id) FROM port WHERE reply IS NOT NULL",

	STMT_PORT_GET_OPEN: `
SELECT 
  id, 
  host_id, 
  port, 
  timestamp, 
  reply
FROM port
WHERE reply IS NOT NULL
ORDER BY port`,
}

var retry_pat *regexp.Regexp = regexp.MustCompile("(?i)(database is locked|busy)")

const RETRY_DELAY = 10 * time.Millisecond

const cache_timeout = time.Second * 1200

type HostDB struct {
	db         *sql.DB
	stmt_table map[QueryID]*sql.Stmt
	tx         *sql.Tx
	log        *log.Logger
	path       string
	host_cache *cache2go.CacheTable
}

func OpenDB(path string) (*HostDB, error) {
	var err error
	var msg string
	var db_exists bool

	db := &HostDB{
		path:       path,
		stmt_table: make(map[QueryID]*sql.Stmt),
		host_cache: cache2go.Cache("host"),
	}

	if db.log, err = GetLogger("HostDB"); err != nil {
		msg = fmt.Sprintf("Error creating logger for HostDB: %s", err.Error())
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	open_lock.Lock()
	defer open_lock.Unlock()

	if db_exists, err = krylib.Fexists(path); err != nil {
		msg = fmt.Sprintf("Error checking if HostDB exists at %s: %s", path, err.Error())
		db.log.Println(msg)
		return nil, errors.New(msg)
	} else if db.db, err = sql.Open("sqlite3", path); err != nil {
		msg = fmt.Sprintf("Error opening database at %s: %s", path, err.Error())
		db.log.Println(msg)
		return nil, errors.New(msg)
	} else {
		db.db.Exec("PRAGMA foreign_keys = on")
		db.db.Exec("PRAGMA journal_mode = wal")
	}

	if !db_exists {
		db.log.Printf("Initializing fresh database at %s...\n", path)
		if err = db.initialize(); err != nil {
			msg = fmt.Sprintf("Error initializing database at %s: %s",
				path, err.Error())
			db.log.Println(msg)
			db.db.Close()
			os.Remove(path)
			return nil, errors.New(msg)
		}
	}

	return db, nil
} // func OpenDB(path string) (*HostDB, error)

func (self *HostDB) worth_a_retry(err error) bool {
	return retry_pat.MatchString(err.Error())
} // func (self *HostDB) worth_a_retry(err error) bool

func (self *HostDB) getStatement(stmt_id QueryID) (*sql.Stmt, error) {
	if stmt, ok := self.stmt_table[stmt_id]; ok {
		return stmt, nil
	}

	var stmt *sql.Stmt
	var err error
	var msg string

PREPARE_QUERY:
	if stmt, err = self.db.Prepare(stmt_table[stmt_id]); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto PREPARE_QUERY
		} else {
			msg = fmt.Sprintf("Error preparing query %s: %s",
				stmt_table[stmt_id], err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		self.stmt_table[stmt_id] = stmt
		return stmt, nil
	}
} // func (self *HostDB) getStatement(stmt_id QueryID) (*sql.Stmt, error)

func (self *HostDB) Begin() error {
	var err error
	var msg string
	var tx *sql.Tx

	if self.tx != nil {
		msg = "Cannot start transaction: A transaction is already in progress!"
		self.log.Println(msg)
		return errors.New(msg)
	}

BEGIN:
	if tx, err = self.db.Begin(); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto BEGIN
		} else {
			msg = fmt.Sprintf("Cannot start transaction: %s", err.Error())
			self.log.Println(msg)
			return errors.New(msg)
		}
	} else {
		self.tx = tx
		return nil
	}
} // func (self *HostDB) Begin() error

func (self *HostDB) Rollback() error {
	var err error
	var msg string

	if self.tx == nil {
		msg = "Cannot roll back transaction: No transaction is active!"
		self.log.Println(msg)
		return errors.New(msg)
	} else if err = self.tx.Rollback(); err != nil {
		msg = fmt.Sprintf("Cannot roll back transaction: %s", err.Error())
		self.log.Println(msg)
		return errors.New(msg)
	} else {
		self.tx = nil
		return nil
	}
} // func (self *HostDB) Rollback() error

func (self *HostDB) Commit() error {
	var err error
	var msg string

	if self.tx == nil {
		msg = "Cannot commit transaction: No transaction is active!"
		self.log.Println(msg)
		return errors.New(msg)
	} else if err = self.tx.Commit(); err != nil {
		msg = fmt.Sprintf("Cannot commit transaction: %s", err.Error())
		self.log.Println(msg)
		return errors.New(msg)
	} else {
		self.tx = nil
		return nil
	}
} // func (self *HostDB) Commit() error

// Initialize a fresh database, i.e. create all the tables and indices.
// Commit if everythings works as planned, otherwise, roll back, close
// the database, delete the database file, and return an error.
func (self *HostDB) initialize() error {
	var err error

	err = self.Begin()
	if err != nil {
		msg := fmt.Sprintf("Error starting transaction to initialize database: %s",
			err.Error())
		self.log.Println(msg)
		return errors.New(msg)
	}

	for _, query := range init_query {
		if _, err = self.tx.Exec(query); err != nil {
			msg := fmt.Sprintf("Error executing query %s: %s",
				query, err.Error())
			self.log.Println(msg)
			self.db.Close()
			self.db = nil
			os.Remove(self.path)
			return errors.New(msg)
		}
	}

	self.Commit()
	return nil
} // func (self *HostDB) initialize() error

func (self *HostDB) Close() {
	for _, stmt := range self.stmt_table {
		stmt.Close()
	}

	self.stmt_table = nil

	if self.tx != nil {
		self.tx.Rollback()
		self.tx = nil
	}

	self.db.Close()
} // func (self *HostDB) Close()

func (self *HostDB) HostAdd(host *Host) error {
	var err error
	var msg string
	var stmt *sql.Stmt
	var tx *sql.Tx
	var ad_hoc bool

GET_QUERY:
	if stmt, err = self.getStatement(STMT_HOST_ADD); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_HOST_ADD: %s", err.Error())
			self.log.Println(msg)
			return errors.New(msg)
		}
	} else if self.tx != nil {
		tx = self.tx
	} else {
		ad_hoc = true
	START_AD_HOC_TX:
		if tx, err = self.db.Begin(); err != nil {
			if self.worth_a_retry(err) {
				time.Sleep(RETRY_DELAY)
				goto START_AD_HOC_TX
			} else {
				msg = fmt.Sprintf("Error starting ad-hoc transaction: %s", err.Error())
				self.log.Println(msg)
				return errors.New(msg)
			}
		}
	}

	stmt = tx.Stmt(stmt)
	now := time.Now()

	var res sql.Result
	var id int64

EXEC_QUERY:
	res, err = stmt.Exec(
		host.Address.String(),
		host.Name,
		host.Source,
		now)
	if err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error add host %s (%s) to database: %s",
				host.Name, host.Address.String(), err.Error())
			self.log.Println(msg)

			if ad_hoc {
				tx.Rollback()
			}

			return errors.New(msg)
		}
	}

	host.Added = now
	if id, err = res.LastInsertId(); err != nil {
		msg = fmt.Sprintf("Error getting ID of freshly added host %s (%s): %s",
			host.Name, host.Address.String(), err.Error())
		self.log.Println(msg)
		if ad_hoc {
			tx.Rollback()
		}
		return errors.New(msg)
	} else {
		host.ID = krylib.ID(id)
		self.host_cache.Add(host.Address.String(), cache_timeout, true)
	}

	if ad_hoc {
		tx.Commit()
	}

	return nil
} // func (self *HostDB) HostAdd(host *Host) error

func (self *HostDB) HostGetByID(id krylib.ID) (*Host, error) {
	var msg string
	var err error
	var stmt *sql.Stmt

GET_QUERY:
	if stmt, err = self.getStatement(STMT_HOST_GET_BY_ID); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_HOST_GET_BY_ID: %s", err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if self.tx != nil {
		stmt = self.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying host by ID %d: %s",
				id, err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	if rows.Next() {
		var host *Host = &Host{ID: id}
		var addr string
		var stamp int64

		if err = rows.Scan(&addr, &host.Name, &host.Source, &stamp); err != nil {
			msg = fmt.Sprintf("Error scanning Host from row: %s",
				err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		} else {
			host.Address = net.ParseIP(addr)
			host.Added = time.Unix(stamp, 0)
			return host, nil
		}
	} else {
		return nil, nil
	}
} // func (self *HostDB) HostGetByID(id krylib.ID) (*Host, error)

func (self *HostDB) HostGetRandom(max int) ([]Host, error) {
	var err error
	var msg string
	var stmt *sql.Stmt
	var hosts []Host

GET_QUERY:
	if stmt, err = self.getStatement(STMT_HOST_GET_RANDOM); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_HOST_GET_RANDOM: %s",
				err.Error())
			self.log.Println(msg)
		}
	} else if self.tx != nil {
		stmt = self.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(max); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying %d random hosts: %s",
				max, err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	hosts = make([]Host, max)
	idx := 0

	for rows.Next() {
		var id, stamp, source int64
		var host Host
		var addr_str string

	SCAN_ROW:
		err = rows.Scan(
			&id,
			&addr_str,
			&host.Name,
			&source,
			&stamp)
		if err != nil {
			if self.worth_a_retry(err) {
				time.Sleep(RETRY_DELAY)
				goto SCAN_ROW
			} else {
				msg = fmt.Sprintf("Error scanning row: %s", err.Error())
				self.log.Println(msg)
				return nil, errors.New(msg)
			}
		} else {
			host.ID = krylib.ID(id)
			host.Source = HostSource(source)
			host.Address = net.ParseIP(addr_str)
			host.Added = time.Unix(stamp, 0)
			hosts[idx] = host
			idx++
		}
	}

	if idx < max {
		return hosts[0:idx], nil
	} else {
		return hosts, nil
	}
} // func (self *HostDB) HostGetRandom(max int) ([]Host, error)

func (self *HostDB) HostExists(addr string) (bool, error) {
	var err error
	var msg string
	var stmt *sql.Stmt

	if _, err = self.host_cache.Value(addr); err == nil {
		return true, nil
	}

GET_QUERY:
	if stmt, err = self.getStatement(STMT_HOST_EXISTS); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_HOST_EXISTS: %s",
				err.Error())
			self.log.Println(msg)
			return false, errors.New(msg)
		}
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(addr); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying if host %s is already in the database: %s",
				addr, err.Error())
			self.log.Println(msg)
			return false, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	if rows.Next() {
		var cnt int64

		if err = rows.Scan(&cnt); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s",
				err.Error())
			self.log.Println(msg)
			return false, errors.New(msg)
		} else if cnt > 0 {
			self.host_cache.Add(addr, cache_timeout, true)
			return true, nil
		} else {
			return false, nil
		}
	} else {
		msg = fmt.Sprintf("CANTHAPPEN: No result rows looking for presence of host %s!",
			addr)
		self.log.Println(msg)
		return false, errors.New(msg)
	}
} // func (self *HostDB) HostExists(addr string) (bool, error)

func (self *HostDB) XfrAdd(xfr *XFR) error {
	var err error
	var msg string
	var stmt *sql.Stmt
	var tx *sql.Tx
	var ad_hoc bool

GET_STATEMENT:
	if stmt, err = self.getStatement(STMT_XFR_ADD); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_STATEMENT
		} else {
			msg = fmt.Sprintf("Error getting query STMT_XFR_ADD: %s", err.Error())
			self.log.Println(msg)
			return errors.New(msg)
		}
	} else if self.tx == nil {
		ad_hoc = true
	BEGIN_AD_HOC:
		if tx, err = self.db.Begin(); err != nil {
			if self.worth_a_retry(err) {
				time.Sleep(RETRY_DELAY)
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting ad-hoc transaction for adding XFR of %s: %s",
					xfr.Zone, err.Error())
				self.log.Println(msg)
				return errors.New(msg)
			}
		}
	} else {
		tx = self.tx
	}

	stmt = tx.Stmt(stmt)

	var result sql.Result

EXEC_QUERY:
	if result, err = stmt.Exec(xfr.Zone, xfr.Start); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error adding XFR for %s: %s",
				xfr.Zone, err.Error())
			self.log.Println(msg)

			if ad_hoc {
				tx.Rollback()
			}

			return errors.New(msg)
		}
	} else {
		var id int64

		if id, err = result.LastInsertId(); err != nil {
			msg = fmt.Sprintf("Error getting ID of freshly added XFR (%s): %s",
				xfr.Zone, err.Error())
			self.log.Println(msg)

			if ad_hoc {
				tx.Rollback()
			}

			return errors.New(msg)
		} else {
			xfr.ID = krylib.ID(id)
			if ad_hoc {
				tx.Commit()
			}
			return nil
		}
	}
} // func (self *HostDB) XfrAdd(xfr *XFR) error

func (self *HostDB) XfrFinish(xfr *XFR, status XfrStatus) error {
	var msg string
	var err error
	var stmt *sql.Stmt
	var tx *sql.Tx
	var ad_hoc bool

GET_QUERY:
	if stmt, err = self.getStatement(STMT_XFR_FINISH); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_XFR_FINISH: %s", err.Error())
			self.log.Println(msg)
			return errors.New(msg)
		}
	} else if self.tx == nil {
		ad_hoc = true
	BEGIN_AD_HOC:
		if tx, err = self.db.Begin(); err != nil {
			if self.worth_a_retry(err) {
				time.Sleep(RETRY_DELAY)
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting ad-hoc transaction for finishing XFR of %s: %s",
					xfr.Zone, err.Error())
				self.log.Println(msg)
				return errors.New(msg)
			}
		}
	} else {
		tx = self.tx
	}

	stmt = tx.Stmt(stmt)

	// var result sql.Result
	// var rows_affected int64
	var now time.Time = time.Now()

EXEC_QUERY:
	if _, err = stmt.Exec(now, status, xfr.ID); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error finishing XFR of %s (%d): %s",
				xfr.Zone, xfr.ID, err.Error())
			self.log.Println(msg)
			if ad_hoc {
				tx.Rollback()
			}
			return errors.New(msg)
		}
	} else if ad_hoc {
		tx.Commit()
	}

	xfr.End = now
	return nil
} // func (self *HostDB) XfrFinish(xfr *XFR, status XfrStatus) error

func (self *HostDB) XfrGetByZone(zone string) (*XFR, error) {
	var msg string
	var err error
	var stmt *sql.Stmt

GET_QUERY:
	if stmt, err = self.getStatement(STMT_XFR_GET_BY_ZONE); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_XFR_GET_BY_ZONE: %s", err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if self.tx != nil {
		stmt = self.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(zone); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying XFR for %s: %s",
				zone, err.Error())
			self.log.Println(msg)
		}
	} else {
		defer rows.Close()
	}

	if rows.Next() {
		xfr := &XFR{Zone: zone}
		var id, start, end, status int64

		if err = rows.Scan(&id, &start, &end, &status); err != nil {
			msg = fmt.Sprintf("Error scanning row into result: %s", err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		} else {
			xfr.ID = krylib.ID(id)
			xfr.Start = time.Unix(start, 0)
			xfr.End = time.Unix(end, 0)
			xfr.Status = XfrStatus(status)
			return xfr, nil
		}
	} else {
		self.log.Printf("No XFR found for %s\n", zone)
		return nil, nil
	}
} // func (self *HostDB) XfrGetByZone(zone string) (*XFR, error)

func (self *HostDB) PortAdd(res *ScanResult) error {
	var err error
	var msg string
	var stmt *sql.Stmt
	var tx *sql.Tx
	var ad_hoc bool

GET_QUERY:
	if stmt, err = self.getStatement(STMT_PORT_ADD); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_PORT_ADD: %s",
				err.Error())
			self.log.Println(msg)
			return errors.New(msg)
		}
	} else if self.tx != nil {
		tx = self.tx
	} else {
		ad_hoc = true
	BEGIN_AD_HOC:
		if tx, err = self.db.Begin(); err != nil {
			if self.worth_a_retry(err) {
				time.Sleep(RETRY_DELAY)
				goto BEGIN_AD_HOC
			} else {
				msg = fmt.Sprintf("Error starting ad-hoc TX for adding Port: %s",
					err.Error())
				self.log.Println(msg)
				return errors.New(msg)
			}
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(res.Host.ID, res.Port, res.Stamp.Unix(), res.Reply); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error adding ScanResult to database: %s",
				err.Error())
			self.log.Println(msg)
			if ad_hoc {
				tx.Rollback()
			}
			return errors.New(msg)
		}
	} else if ad_hoc {
		tx.Commit()
	}

	return nil
} // func (self *HostDB) PortAdd(res *ScanResult) error

func (self *HostDB) PortGetByHost(host_id krylib.ID) ([]Port, error) {
	var err error
	var msg string
	var stmt *sql.Stmt
	var rows *sql.Rows
	var ports []Port

GET_QUERY:
	if stmt, err = self.getStatement(STMT_PORT_GET_BY_HOST); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_PORT_GET_BY_HOST: %s",
				err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if self.tx != nil {
		stmt = self.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(host_id); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying ports for Host #%d: %s",
				host_id, err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
		ports = make([]Port, 0)
	}

	for rows.Next() {
		var port_id, stamp int64
		var port Port = Port{
			HostID: host_id,
		}

	SCAN_ROW:
		if err = rows.Scan(&port_id, &port.Port, &stamp, &port.Reply); err != nil {
			if self.worth_a_retry(err) {
				time.Sleep(RETRY_DELAY)
				goto SCAN_ROW
			} else {
				msg = fmt.Sprintf("Error scanning result row into Port: %s",
					err.Error())
				self.log.Println(msg)
				return nil, errors.New(msg)
			}
		} else {
			port.ID = krylib.ID(port_id)
			port.Timestamp = time.Unix(stamp, 0)
			ports = append(ports, port)
		}
	}

	return ports, nil
} // func (self *HostDB) PortGetByHost(id krylib.ID) ([]Port, error)

func (self *HostDB) PortGetReplyCount() (int64, error) {
	var msg string
	var err error
	var stmt *sql.Stmt
	var rows *sql.Rows
	var cnt int64

GET_QUERY:
	if stmt, err = self.getStatement(STMT_PORT_GET_REPLY_CNT); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_PORT_GET_REPLY_CNT: %s",
				err.Error())
			self.log.Println(msg)
			return 0, errors.New(msg)
		}
	} else if self.tx != nil {
		stmt = self.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying number of scanned open ports: %s",
				err.Error())
			self.log.Println(msg)
			return 0, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	rows.Next()

	if err = rows.Scan(&cnt); err != nil {
		msg = fmt.Sprintf("Error scanning result set for number of scanned ports: %s",
			err.Error())
		self.log.Println(msg)
		return 0, errors.New(msg)
	} else {
		return cnt, nil
	}
} // func (self *HostDB) PortGetReplyCount() (int64, error)

func (self *HostDB) PortGetOpen() ([]ScanResult, error) {
	var msg string
	var err error
	var stmt *sql.Stmt
	var rows *sql.Rows
	var result []ScanResult
	var host_cache map[krylib.ID]*Host

GET_QUERY:
	if stmt, err = self.getStatement(STMT_PORT_GET_OPEN); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_PORT_GET_OPEN: %s", err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if self.tx != nil {
		stmt = self.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying for open ports: %s",
				err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	result = make([]ScanResult, 0)
	host_cache = make(map[krylib.ID]*Host)

	for rows.Next() {
		var id, host_id, timestamp, port int64
		var reply string

	SCAN_ROW:
		if err = rows.Scan(&id, &host_id, &port, &timestamp, &reply); err != nil {
			if self.worth_a_retry(err) {
				time.Sleep(RETRY_DELAY)
				goto SCAN_ROW
			} else {
				msg = fmt.Sprintf("Error scanning result set: %s",
					err.Error())
				self.log.Println(msg)
				return nil, errors.New(msg)
			}
		} else {
			var host *Host // = new(Host)
			var ok bool
			if host, ok = host_cache[krylib.ID(host_id)]; !ok {
				if host, err = self.HostGetByID(krylib.ID(host_id)); err != nil {
					// CANTHAPPEN!!!
					msg = fmt.Sprintf("CANTHAPPEN: Error looking up host for port: %s",
						err.Error())
					self.log.Println(msg)
					continue
				} else if host == nil {
					// CANTHAPPEN!!!
					msg = fmt.Sprintf("CANTHAPPEN: Did not find host for port #%d (Host #%d)",
						id, host_id)
					continue
				} else {
					host_cache[krylib.ID(host_id)] = host
				}
			}

			var res ScanResult = ScanResult{
				Host:  *host,
				Port:  uint16(port),
				Reply: &reply,
				Stamp: time.Unix(timestamp, 0),
				Err:   nil,
			}

			result = append(result, res)
		}
	}

	return result, nil
} // func (self *HostDB) PortGetOpen() ([]ScanResult, error)

func (self *HostDB) HostGetCount() (int64, error) {
	var msg string
	var err error
	var stmt *sql.Stmt
	var rows *sql.Rows
	var cnt int64

GET_QUERY:
	if stmt, err = self.getStatement(STMT_HOST_GET_CNT); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_HOST_GET_CNT: %s",
				err.Error())
			self.log.Println(msg)
			return -1, errors.New(msg)
		}
	} else if self.tx != nil {
		stmt = self.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying host count: %s",
				err.Error())
			self.log.Println(msg)
			return 0, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	rows.Next()

	if err = rows.Scan(&cnt); err != nil {
		msg = fmt.Sprintf("Error scanning result: %s", err.Error())
		self.log.Println(msg)
		return -1, errors.New(msg)
	} else {
		return cnt, nil
	}
} // func (self *HostDB) HostGetCount() (int64, error)

func (self *HostDB) HostGetByHostReport() ([]HostWithPorts, error) {
	var err error
	var msg string
	var stmt *sql.Stmt
	var rows *sql.Rows
	var ports map[krylib.ID][]Port = make(map[krylib.ID][]Port)

GET_QUERY:
	if stmt, err = self.getStatement(STMT_PORT_GET_OPEN); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error preparing query STMT_HOST_PORT_BY_HOST: %s",
				err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if self.tx != nil {
		stmt = self.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if self.worth_a_retry(err) {
			time.Sleep(RETRY_DELAY)
			goto EXEC_QUERY
		}
	} else {
		defer rows.Close()
	}

	for rows.Next() {
		var port_id, host_id, stamp, port_no int64
		var ok bool
		var reply *string
		var portlist []Port

		if err = rows.Scan(&port_id, &host_id, &port_no, &stamp, &reply); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s", err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		}

		port := Port{
			ID:        krylib.ID(port_id),
			HostID:    krylib.ID(host_id),
			Port:      uint16(port_no),
			Timestamp: time.Unix(stamp, 0),
			Reply:     reply,
		}

		if portlist, ok = ports[port.HostID]; ok {
			ports[port.HostID] = append(portlist, port)
		} else {
			ports[port.HostID] = []Port{port}
		}
	}

	var res []HostWithPorts = make([]HostWithPorts, len(ports))
	var idx int = 0

	for host_id, portlist := range ports {
		var host *Host

		if host, err = self.HostGetByID(host_id); err != nil {
			msg = fmt.Sprintf("Error retrieving host #%d from database: %s",
				host_id, err.Error())
			self.log.Println(msg)
			return nil, errors.New(msg)
		} else {
			res[idx] = HostWithPorts{
				Host:  *host,
				Ports: portlist,
			}
			idx++
		}
	}

	return res, nil
} // func (self *HostDB) HostGetByHostReport() ([]HostWithPorts, error)
