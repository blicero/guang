// /Users/krylon/go/src/guang/database.go
// -*- coding: utf-8; mode: go; -*-
// Created on 23. 12. 2015 by Benjamin Walkenhorst
// (c) 2015 Benjamin Walkenhorst
// Time-stamp: <2023-03-27 11:16:19 krylon>
//
// Samstag, 20. 08. 2016, 21:27
// Ich würde für Hosts gern a) anhand der Antworten, die ich erhalte, das
// Betriebssystem ermitteln, und b) anhand der IP-Adresse den ungefähren
// Standort.

// Package database provides an interface to the underlying database.
package database

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

	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/blicero/guang/database/query"
	"github.com/blicero/guang/xfr/xfrstatus"
	"github.com/blicero/krylib"

	_ "github.com/mattn/go-sqlite3" // Import the database driver
	"github.com/muesli/cache2go"
)

var (
	openLock sync.Mutex
	retryPat *regexp.Regexp = regexp.MustCompile("(?i)(database is locked|busy)")
)

const (
	retryDelay   = 10 * time.Millisecond
	cacheTimeout = time.Second * 1200
)

// HostDB is a wrapper around the database connection.
type HostDB struct {
	db        *sql.DB
	stmtTable map[query.ID]*sql.Stmt
	tx        *sql.Tx
	log       *log.Logger
	path      string
	hostCache *cache2go.CacheTable
}

// OpenDB opens a new database connection.
func OpenDB(path string) (*HostDB, error) {
	var err error
	var msg string
	var dbExists bool

	db := &HostDB{
		path:      path,
		stmtTable: make(map[query.ID]*sql.Stmt),
		hostCache: cache2go.Cache("host"),
	}

	if db.log, err = common.GetLogger("HostDB"); err != nil {
		msg = fmt.Sprintf("Error creating logger for HostDB: %s", err.Error())
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	var connstring = fmt.Sprintf("%s?_locking=NORMAL&_journal=WAL&_fk=1&recursive_triggers=0",
		path)

	openLock.Lock()
	defer openLock.Unlock()

	if dbExists, err = krylib.Fexists(path); err != nil {
		msg = fmt.Sprintf("Error checking if HostDB exists at %s: %s", path, err.Error())
		db.log.Println(msg)
		return nil, errors.New(msg)
	} else if db.db, err = sql.Open("sqlite3", connstring); err != nil {
		msg = fmt.Sprintf("Error opening database at %s: %s", path, err.Error())
		db.log.Println(msg)
		return nil, errors.New(msg)
	} else if !dbExists {
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

func (db *HostDB) worthARetry(err error) bool {
	return retryPat.MatchString(err.Error())
} // func (db *HostDB) worth_a_retry(err error) bool

func (db *HostDB) getStatement(qid query.ID) (*sql.Stmt, error) {
	if stmt, ok := db.stmtTable[qid]; ok {
		return stmt, nil
	}

	var stmt *sql.Stmt
	var err error
	var msg string

PREPARE_QUERY:
	if stmt, err = db.db.Prepare(dbQueries[qid]); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto PREPARE_QUERY
		} else {
			msg = fmt.Sprintf("Error preparing query %s %s\n\n%s\n",
				qid, err.Error(), dbQueries[qid])
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		db.stmtTable[qid] = stmt
		return stmt, nil
	}
} // func (db *HostDB) getStatement(stmt_id query.QueryID) (*sql.Stmt, error)

// Begin starts a transaction
func (db *HostDB) Begin() error {
	var err error
	var msg string
	var tx *sql.Tx

	if db.tx != nil {
		msg = "Cannot start transaction: A transaction is already in progress!"
		db.log.Println(msg)
		return errors.New(msg)
	}

BEGIN:
	if tx, err = db.db.Begin(); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto BEGIN
		} else {
			msg = fmt.Sprintf("Cannot start transaction: %s", err.Error())
			db.log.Println(msg)
			return errors.New(msg)
		}
	} else {
		db.tx = tx
		return nil
	}
} // func (db *HostDB) Begin() error

// Rollback aborts a transaction
func (db *HostDB) Rollback() error {
	var err error
	var msg string

	if db.tx == nil {
		msg = "Cannot roll back transaction: No transaction is active!"
		db.log.Println(msg)
		return errors.New(msg)
	} else if err = db.tx.Rollback(); err != nil {
		msg = fmt.Sprintf("Cannot roll back transaction: %s", err.Error())
		db.log.Println(msg)
		return errors.New(msg)
	} else {
		db.tx = nil
		return nil
	}
} // func (db *HostDB) Rollback() error

// Commit finishes a transaction
func (db *HostDB) Commit() error {
	var err error
	var msg string

	if db.tx == nil {
		msg = "Cannot commit transaction: No transaction is active!"
		db.log.Println(msg)
		return errors.New(msg)
	} else if err = db.tx.Commit(); err != nil {
		msg = fmt.Sprintf("Cannot commit transaction: %s", err.Error())
		db.log.Println(msg)
		return errors.New(msg)
	} else {
		db.tx = nil
		return nil
	}
} // func (db *HostDB) Commit() error

// Initialize a fresh database, i.e. create all the tables and indices.
// Commit if everythings works as planned, otherwise, roll back, close
// the database, delete the database file, and return an error.
func (db *HostDB) initialize() error {
	var err error

	err = db.Begin()
	if err != nil {
		msg := fmt.Sprintf("Error starting transaction to initialize database: %s",
			err.Error())
		db.log.Println(msg)
		return errors.New(msg)
	}

	for _, query := range initQueries {
		if _, err = db.tx.Exec(query); err != nil {
			msg := fmt.Sprintf("Error executing query %s: %s",
				query, err.Error())
			db.log.Println(msg)
			db.db.Close()
			db.db = nil
			os.Remove(db.path)
			return errors.New(msg)
		}
	}

	db.Commit() // nolint: errcheck
	return nil
} // func (db *HostDB) initialize() error

// Close closes the database connection
func (db *HostDB) Close() {
	for _, stmt := range db.stmtTable {
		stmt.Close()
	}

	db.stmtTable = nil

	if db.tx != nil {
		db.tx.Rollback() // nolint: errcheck
		db.tx = nil
	}

	db.db.Close()
} // func (db *HostDB) Close()

// HostAdd adds a new Host to the database.
func (db *HostDB) HostAdd(host *data.Host) error {
	var err error
	var msg string
	var stmt *sql.Stmt
	var tx *sql.Tx
	var adHoc bool

GET_QUERY:
	if stmt, err = db.getStatement(query.HostAdd); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_HOST_ADD: %s", err.Error())
			db.log.Println(msg)
			return errors.New(msg)
		}
	} else if db.tx != nil {
		tx = db.tx
	} else {
		adHoc = true
	START_ADHOC_TX:
		if tx, err = db.db.Begin(); err != nil {
			if db.worthARetry(err) {
				time.Sleep(retryDelay)
				goto START_ADHOC_TX
			} else {
				msg = fmt.Sprintf("Error starting ad-hoc transaction: %s", err.Error())
				db.log.Println(msg)
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
		now.Unix())
	if err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error add host %s (%s) to database: %s",
				host.Name, host.Address.String(), err.Error())
			db.log.Println(msg)

			if adHoc {
				tx.Rollback() // nolint: errcheck
			}

			return errors.New(msg)
		}
	}

	host.Added = now
	if id, err = res.LastInsertId(); err != nil {
		msg = fmt.Sprintf("Error getting ID of freshly added host %s (%s): %s",
			host.Name, host.Address.String(), err.Error())
		db.log.Println(msg)
		if adHoc {
			tx.Rollback() // nolint: errcheck
		}
		return errors.New(msg)
	}

	host.ID = krylib.ID(id)
	db.hostCache.Add(host.Address.String(), cacheTimeout, true)

	if adHoc {
		tx.Commit() // nolint: errcheck
	}

	return nil
} // func (db *HostDB) HostAdd(host *Host) error

// HostGetByID loads a Host by its ID
func (db *HostDB) HostGetByID(id krylib.ID) (*data.Host, error) {
	var msg string
	var err error
	var stmt *sql.Stmt

GET_QUERY:
	if stmt, err = db.getStatement(query.HostGetByID); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_HOST_GET_BY_ID: %s", err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(id); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying host by ID %d: %s",
				id, err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	if rows.Next() {
		var host *data.Host = &data.Host{ID: id}
		var addr string
		var stamp int64

		if err = rows.Scan(&addr, &host.Name, &host.Source, &stamp); err != nil {
			msg = fmt.Sprintf("Error scanning Host from row: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}

		host.Address = net.ParseIP(addr)
		host.Added = time.Unix(stamp, 0)
		return host, nil
	}

	return nil, nil
} // func (db *HostDB) HostGetByID(id krylib.ID) (*Host, error)

// HostGetAll returns ALL hosts from the database.
func (db *HostDB) HostGetAll() ([]data.Host, error) {
	const qid query.ID = query.HostGetAll
	var err error
	var msg string
	var stmt *sql.Stmt
	var hosts []data.Host

GET_QUERY:
	if stmt, err = db.getStatement(qid); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query %s: %s",
				qid,
				err.Error())
			db.log.Println(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying all hosts: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	hosts = make([]data.Host, 0)

	for rows.Next() {
		var id, stamp, source int64
		var host data.Host
		var addrStr string

	SCAN_ROW:
		err = rows.Scan(
			&id,
			&addrStr,
			&host.Name,
			&source,
			&stamp)
		if err != nil {
			if db.worthARetry(err) {
				time.Sleep(retryDelay)
				goto SCAN_ROW
			} else {
				msg = fmt.Sprintf("Error scanning row: %s", err.Error())
				db.log.Println(msg)
				return nil, errors.New(msg)
			}
		} else {
			host.ID = krylib.ID(id)
			host.Source = data.HostSource(source)
			host.Address = net.ParseIP(addrStr)
			host.Added = time.Unix(stamp, 0)
			hosts = append(hosts, host)
		}
	}

	return hosts, nil
} // func (db *HostDB) HostGetAll() ([]data.Host, error)

// HostGetRandom fetches up to <max> randomly chosen hosts from the database.
func (db *HostDB) HostGetRandom(max int) ([]data.Host, error) {
	const qid query.ID = query.HostGetRandom
	var err error
	var msg string
	var stmt *sql.Stmt
	var hosts []data.Host

GET_QUERY:
	if stmt, err = db.getStatement(qid); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query %s: %s",
				qid,
				err.Error())
			db.log.Println(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(max); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying %d random hosts: %s",
				max, err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	hosts = make([]data.Host, max)
	idx := 0

	for rows.Next() {
		var id, stamp, source int64
		var host data.Host
		var addrStr string

	SCAN_ROW:
		err = rows.Scan(
			&id,
			&addrStr,
			&host.Name,
			&source,
			&stamp)
		if err != nil {
			if db.worthARetry(err) {
				time.Sleep(retryDelay)
				goto SCAN_ROW
			} else {
				msg = fmt.Sprintf("Error scanning row: %s", err.Error())
				db.log.Println(msg)
				return nil, errors.New(msg)
			}
		} else {
			host.ID = krylib.ID(id)
			host.Source = data.HostSource(source)
			host.Address = net.ParseIP(addrStr)
			host.Added = time.Unix(stamp, 0)
			hosts[idx] = host
			idx++
		}
	}

	if idx < max {
		return hosts[0:idx], nil
	}

	return hosts, nil
} // func (db *HostDB) HostGetRandom(max int) ([]Host, error)

// HostExists checks if a Host with the given address already exists in the database.
func (db *HostDB) HostExists(addr string) (bool, error) {
	var err error
	var msg string
	var stmt *sql.Stmt

	if _, err = db.hostCache.Value(addr); err == nil {
		return true, nil
	}

GET_QUERY:
	if stmt, err = db.getStatement(query.HostExists); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_HOST_EXISTS: %s",
				err.Error())
			db.log.Println(msg)
			return false, errors.New(msg)
		}
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(addr); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying if host %s is already in the database: %s",
				addr, err.Error())
			db.log.Println(msg)
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
			db.log.Println(msg)
			return false, errors.New(msg)
		} else if cnt > 0 {
			db.hostCache.Add(addr, cacheTimeout, true)
			return true, nil
		} else {
			return false, nil
		}
	} else {
		msg = fmt.Sprintf("CANTHAPPEN: No result rows looking for presence of host %s!",
			addr)
		db.log.Println(msg)
		return false, errors.New(msg)
	}
} // func (db *HostDB) HostExists(addr string) (bool, error)

// XfrAdd starts a new zone transfer
func (db *HostDB) XfrAdd(xfr *data.XFR) error {
	var err error
	var msg string
	var stmt *sql.Stmt
	var tx *sql.Tx
	var adHoc bool

GET_STATEMENT:
	if stmt, err = db.getStatement(query.XfrAdd); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_STATEMENT
		} else {
			msg = fmt.Sprintf("Error getting query STMT_XFR_ADD: %s", err.Error())
			db.log.Println(msg)
			return errors.New(msg)
		}
	} else if db.tx == nil {
		adHoc = true
	BEGIN_ADHOC:
		if tx, err = db.db.Begin(); err != nil {
			if db.worthARetry(err) {
				time.Sleep(retryDelay)
				goto BEGIN_ADHOC
			} else {
				msg = fmt.Sprintf("Error starting ad-hoc transaction for adding XFR of %s: %s",
					xfr.Zone, err.Error())
				db.log.Println(msg)
				return errors.New(msg)
			}
		}
	} else {
		tx = db.tx
	}

	stmt = tx.Stmt(stmt)

	var result sql.Result

EXEC_QUERY:
	if result, err = stmt.Exec(xfr.Zone, xfr.Start.Unix()); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error adding XFR for %s: %s",
				xfr.Zone, err.Error())
			db.log.Println(msg)

			if adHoc {
				tx.Rollback() // nolint: errcheck
			}

			return errors.New(msg)
		}
	} else {
		var id int64

		if id, err = result.LastInsertId(); err != nil {
			msg = fmt.Sprintf("Error getting ID of freshly added XFR (%s): %s",
				xfr.Zone, err.Error())
			db.log.Println(msg)

			if adHoc {
				tx.Rollback() // nolint: errcheck
			}

			return errors.New(msg)
		}

		xfr.ID = krylib.ID(id)
		if adHoc {
			tx.Commit() // nolint: errcheck
		}
		return nil
	}
} // func (db *HostDB) XfrAdd(xfr *XFR) error

// XfrFinish marks a zone transfer as finished.
func (db *HostDB) XfrFinish(xfr *data.XFR, status xfrstatus.XfrStatus) error {
	var msg string
	var err error
	var stmt *sql.Stmt
	var tx *sql.Tx
	var adHoc bool

GET_QUERY:
	if stmt, err = db.getStatement(query.XfrFinish); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_XFR_FINISH: %s", err.Error())
			db.log.Println(msg)
			return errors.New(msg)
		}
	} else if db.tx == nil {
		adHoc = true
	BEGIN_ADHOC:
		if tx, err = db.db.Begin(); err != nil {
			if db.worthARetry(err) {
				time.Sleep(retryDelay)
				goto BEGIN_ADHOC
			} else {
				msg = fmt.Sprintf("Error starting ad-hoc transaction for finishing XFR of %s: %s",
					xfr.Zone, err.Error())
				db.log.Println(msg)
				return errors.New(msg)
			}
		}
	} else {
		tx = db.tx
	}

	stmt = tx.Stmt(stmt)

	var now = time.Now()

EXEC_QUERY:
	if _, err = stmt.Exec(now.Unix(), status, xfr.ID); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error finishing XFR of %s (%d): %s",
				xfr.Zone, xfr.ID, err.Error())
			db.log.Println(msg)
			if adHoc {
				tx.Rollback() // nolint: errcheck
			}
			return errors.New(msg)
		}
	} else if adHoc {
		tx.Commit() // nolint: errcheck
	}

	xfr.End = now
	return nil
} // func (db *HostDB) XfrFinish(xfr *XFR, status XfrStatus) error

// XfrGetByZone fetches an Xfr by the zone's name.
func (db *HostDB) XfrGetByZone(zone string) (*data.XFR, error) {
	var msg string
	var err error
	var stmt *sql.Stmt

GET_QUERY:
	if stmt, err = db.getStatement(query.XfrGetByZone); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_XFR_GET_BY_ZONE: %s", err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

	var rows *sql.Rows

EXEC_QUERY:
	if rows, err = stmt.Query(zone); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying XFR for %s: %s",
				zone, err.Error())
			db.log.Println(msg)
		}
	} else {
		defer rows.Close()
	}

	if rows.Next() {
		xfr := &data.XFR{Zone: zone}
		var id, start, end, status int64

		if err = rows.Scan(&id, &start, &end, &status); err != nil {
			msg = fmt.Sprintf("Error scanning row into result: %s", err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}

		xfr.ID = krylib.ID(id)
		xfr.Start = time.Unix(start, 0)
		xfr.End = time.Unix(end, 0)
		xfr.Status = xfrstatus.XfrStatus(status)
		return xfr, nil
	}

	db.log.Printf("No XFR found for %s\n", zone)
	return nil, nil
} // func (db *HostDB) XfrGetByZone(zone string) (*XFR, error)

// PortAdd adds a new scanned port to the database.
func (db *HostDB) PortAdd(res *data.ScanResult) error {
	var err error
	var msg string
	var stmt *sql.Stmt
	var tx *sql.Tx
	var adHoc bool

GET_QUERY:
	if stmt, err = db.getStatement(query.PortAdd); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_PORT_ADD: %s",
				err.Error())
			db.log.Println(msg)
			return errors.New(msg)
		}
	} else if db.tx != nil {
		tx = db.tx
	} else {
		adHoc = true
	BEGIN_ADHOC:
		if tx, err = db.db.Begin(); err != nil {
			if db.worthARetry(err) {
				time.Sleep(retryDelay)
				goto BEGIN_ADHOC
			} else {
				msg = fmt.Sprintf("Error starting ad-hoc TX for adding Port: %s",
					err.Error())
				db.log.Println(msg)
				return errors.New(msg)
			}
		}
	}

	stmt = tx.Stmt(stmt)

EXEC_QUERY:
	if _, err = stmt.Exec(res.Host.ID, res.Port, res.Stamp.Unix(), res.Reply); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error adding ScanResult to database: %s",
				err.Error())
			db.log.Println(msg)
			if adHoc {
				tx.Rollback() // nolint: errcheck
			}
			return errors.New(msg)
		}
	} else if adHoc {
		tx.Commit() // nolint: errcheck
	}

	return nil
} // func (db *HostDB) PortAdd(res *ScanResult) error

// PortGetByHost loads all the scanned ports of a given Host.
func (db *HostDB) PortGetByHost(hostID krylib.ID) ([]data.Port, error) {
	var err error
	var msg string
	var stmt *sql.Stmt
	var rows *sql.Rows
	var ports []data.Port

GET_QUERY:
	if stmt, err = db.getStatement(query.PortGetByHost); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_PORT_GET_BY_HOST: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(hostID); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying ports for Host #%d: %s",
				hostID, err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
		ports = make([]data.Port, 0)
	}

	for rows.Next() {
		var portID, stamp int64
		var port data.Port = data.Port{
			HostID: hostID,
		}

	SCAN_ROW:
		if err = rows.Scan(&portID, &port.Port, &stamp, &port.Reply); err != nil {
			if db.worthARetry(err) {
				time.Sleep(retryDelay)
				goto SCAN_ROW
			} else {
				msg = fmt.Sprintf("Error scanning result row into Port: %s",
					err.Error())
				db.log.Println(msg)
				return nil, errors.New(msg)
			}
		} else {
			port.ID = krylib.ID(portID)
			port.Timestamp = time.Unix(stamp, 0)
			ports = append(ports, port)
		}
	}

	return ports, nil
} // func (db *HostDB) PortGetByHost(id krylib.ID) ([]Port, error)

// PortGetReplyCount returns the number of open ports found on the given Host
func (db *HostDB) PortGetReplyCount() (int64, error) {
	var msg string
	var err error
	var stmt *sql.Stmt
	var rows *sql.Rows
	var cnt int64

GET_QUERY:
	if stmt, err = db.getStatement(query.PortGetReplyCnt); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_PORT_GET_REPLY_CNT: %s",
				err.Error())
			db.log.Println(msg)
			return 0, errors.New(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying number of scanned open ports: %s",
				err.Error())
			db.log.Println(msg)
			return 0, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	rows.Next()

	if err = rows.Scan(&cnt); err != nil {
		msg = fmt.Sprintf("Error scanning result set for number of scanned ports: %s",
			err.Error())
		db.log.Println(msg)
		return 0, errors.New(msg)
	}

	return cnt, nil
} // func (db *HostDB) PortGetReplyCount() (int64, error)

// PortGetOpen loads a list of all open ports that were scanned
func (db *HostDB) PortGetOpen() ([]data.ScanResult, error) {
	var msg string
	var err error
	var stmt *sql.Stmt
	var rows *sql.Rows
	var result []data.ScanResult
	var hostCache map[krylib.ID]*data.Host

GET_QUERY:
	if stmt, err = db.getStatement(query.PortGetOpen); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_PORT_GET_OPEN: %s", err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying for open ports: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	result = make([]data.ScanResult, 0)
	hostCache = make(map[krylib.ID]*data.Host)

	for rows.Next() {
		var id, hostID, timestamp, port int64
		var reply string

	SCAN_ROW:
		if err = rows.Scan(&id, &hostID, &port, &timestamp, &reply); err != nil {
			if db.worthARetry(err) {
				time.Sleep(retryDelay)
				goto SCAN_ROW
			} else {
				msg = fmt.Sprintf("Error scanning result set: %s",
					err.Error())
				db.log.Println(msg)
				return nil, errors.New(msg)
			}
		} else {
			var host *data.Host // = new(Host)
			var ok bool
			if host, ok = hostCache[krylib.ID(hostID)]; !ok {
				if host, err = db.HostGetByID(krylib.ID(hostID)); err != nil {
					// CANTHAPPEN!!!
					msg = fmt.Sprintf("CANTHAPPEN: Error looking up host for port: %s",
						err.Error())
					db.log.Println(msg)
					continue
				} else if host == nil {
					// CANTHAPPEN!!!
					db.log.Printf("CANTHAPPEN: Did not find host for port #%d (Host #%d)\n",
						id, hostID)
					continue
				} else {
					hostCache[krylib.ID(hostID)] = host
				}
			}

			var res data.ScanResult = data.ScanResult{
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
} // func (db *HostDB) PortGetOpen() ([]ScanResult, error)

// PortGetRecent returns all scanned ports that were scanned since the given time.
func (db *HostDB) PortGetRecent(ref time.Time) ([]data.ScanResult, error) {
	const qid = query.PortGetRecent
	var (
		msg       string
		err       error
		stmt      *sql.Stmt
		rows      *sql.Rows
		result    []data.ScanResult
		hostCache map[krylib.ID]*data.Host
	)

GET_QUERY:
	if stmt, err = db.getStatement(qid); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query %s: %s", qid, err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(ref.Unix()); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying for open ports: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	result = make([]data.ScanResult, 0)
	hostCache = make(map[krylib.ID]*data.Host)

	for rows.Next() {
		var id, hostID, timestamp, port int64
		var reply string

	SCAN_ROW:
		if err = rows.Scan(&id, &hostID, &port, &timestamp, &reply); err != nil {
			if db.worthARetry(err) {
				time.Sleep(retryDelay)
				goto SCAN_ROW
			} else {
				msg = fmt.Sprintf("Error scanning result set: %s",
					err.Error())
				db.log.Println(msg)
				return nil, errors.New(msg)
			}
		} else {
			var host *data.Host // = new(Host)
			var ok bool
			if host, ok = hostCache[krylib.ID(hostID)]; !ok {
				if host, err = db.HostGetByID(krylib.ID(hostID)); err != nil {
					// CANTHAPPEN!!!
					msg = fmt.Sprintf("CANTHAPPEN: Error looking up host for port: %s",
						err.Error())
					db.log.Println(msg)
					continue
				} else if host == nil {
					// CANTHAPPEN!!!
					db.log.Printf("CANTHAPPEN: Did not find host for port #%d (Host #%d)\n",
						id, hostID)
					continue
				} else {
					hostCache[krylib.ID(hostID)] = host
				}
			}

			var res data.ScanResult = data.ScanResult{
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
} // func (db *HostDB) PortGetRecent() ([]ScanResult, error)

// HostGetCount returns the number of Hosts in the database.
func (db *HostDB) HostGetCount() (int64, error) {
	var msg string
	var err error
	var stmt *sql.Stmt
	var rows *sql.Rows
	var cnt int64

GET_QUERY:
	if stmt, err = db.getStatement(query.HostGetCnt); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error getting query STMT_HOST_GET_CNT: %s",
				err.Error())
			db.log.Println(msg)
			return -1, errors.New(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		} else {
			msg = fmt.Sprintf("Error querying host count: %s",
				err.Error())
			db.log.Println(msg)
			return 0, errors.New(msg)
		}
	} else {
		defer rows.Close()
	}

	rows.Next()

	if err = rows.Scan(&cnt); err != nil {
		msg = fmt.Sprintf("Error scanning result: %s", err.Error())
		db.log.Println(msg)
		return -1, errors.New(msg)
	}

	return cnt, nil
} // func (db *HostDB) HostGetCount() (int64, error)

// HostGetByHostReport bla
func (db *HostDB) HostGetByHostReport() ([]data.HostWithPorts, error) {
	var err error
	var msg string
	var stmt *sql.Stmt
	var rows *sql.Rows
	var ports map[krylib.ID][]data.Port = make(map[krylib.ID][]data.Port)

GET_QUERY:
	if stmt, err = db.getStatement(query.PortGetOpen); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto GET_QUERY
		} else {
			msg = fmt.Sprintf("Error preparing query STMT_HOST_PORT_BY_HOST: %s",
				err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}
	} else if db.tx != nil {
		stmt = db.tx.Stmt(stmt)
	}

EXEC_QUERY:
	if rows, err = stmt.Query(); err != nil {
		if db.worthARetry(err) {
			time.Sleep(retryDelay)
			goto EXEC_QUERY
		}
	} else {
		defer rows.Close()
	}

	for rows.Next() {
		var portID, hostID, stamp, portNo int64
		var ok bool
		var reply *string
		var portlist []data.Port

		if err = rows.Scan(&portID, &hostID, &portNo, &stamp, &reply); err != nil {
			msg = fmt.Sprintf("Error scanning row: %s", err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}

		port := data.Port{
			ID:        krylib.ID(portID),
			HostID:    krylib.ID(hostID),
			Port:      uint16(portNo),
			Timestamp: time.Unix(stamp, 0),
			Reply:     reply,
		}

		if portlist, ok = ports[port.HostID]; ok {
			ports[port.HostID] = append(portlist, port)
		} else {
			ports[port.HostID] = []data.Port{port}
		}
	}

	var res []data.HostWithPorts = make([]data.HostWithPorts, len(ports))
	var idx int = 0

	for hostID, portlist := range ports {
		var host *data.Host

		if host, err = db.HostGetByID(hostID); err != nil {
			msg = fmt.Sprintf("Error retrieving host #%d from database: %s",
				hostID, err.Error())
			db.log.Println(msg)
			return nil, errors.New(msg)
		}

		res[idx] = data.HostWithPorts{
			Host:  *host,
			Ports: portlist,
		}
		idx++
	}

	return res, nil
} // func (db *HostDB) HostGetByHostReport() ([]HostWithPorts, error)
