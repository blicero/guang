// /home/krylon/go/src/github.com/blicero/guang/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 03. 11. 2022 by Benjamin Walkenhorst
// (c) 2022 Benjamin Walkenhorst
// Time-stamp: <2022-11-03 00:48:57 krylon>

package database

import "github.com/blicero/guang/database/query"

var dbQueries map[query.ID]string = map[query.ID]string{
	query.HostAdd: `
INSERT INTO host (addr, name, source, add_stamp)
          VALUES (   ?,    ?,      ?,         ?)
`,
	query.HostGetByID: "SELECT addr, name, source, add_stamp FROM host WHERE id = ?",
	query.HostGetRandom: `
SELECT id,
       addr,
       name,
       source,
       add_stamp
FROM host
LIMIT ?
OFFSET ABS(RANDOM()) % MAX((SELECT COUNT(*) FROM host), 1)
`,
	query.HostGetCnt: "SELECT COUNT(id) FROM host",
	query.HostExists: "SELECT COUNT(id) FROM host WHERE addr = ?",
	query.HostPortByPort: `
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
	query.PortAdd: `
INSERT INTO port (host_id, port, timestamp, reply)
          VALUES (      ?,    ?,         ?,     ?)
`,
	query.PortGetByHost: "SELECT id, port, timestamp, reply FROM port WHERE host_id = ?",
	query.XfrAdd:        "INSERT INTO xfr (zone, start, status) VALUES (?, ?, 0)",
	query.XfrGetByZone:  "SELECT id, start, end, status FROM xfr WHERE zone = ?",
	query.XfrFinish:     "UPDATE xfr SET end = ?, status = ? WHERE id = ?",
	query.XfrGetUnfinished: `
SELECT id, 
       zone, 
       start, 
       end, 
       status
FROM xfr
WHERE status = 0
`,
	query.PortGetReplyCnt: "SELECT COUNT(id) FROM port WHERE reply IS NOT NULL",
	query.PortGetOpen: `
SELECT 
  id, 
  host_id, 
  port, 
  timestamp, 
  reply
FROM port
WHERE reply IS NOT NULL
ORDER BY port`,
	query.PortGetRecent: `
SELECT 
  id, 
  host_id, 
  port, 
  timestamp, 
  reply
FROM port
WHERE reply IS NOT NULL AND timestamp > ?
ORDER BY port
`,
}
