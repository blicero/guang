// /Users/krylon/go/src/guang/frontend/web.go
// -*- coding: utf-8; mode: go; -*-
// Created on 06. 02. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2022-10-31 18:55:36 krylon>

package frontend

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"text/template"

	"github.com/blicero/guang/backend"
	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/blicero/guang/database"

	"github.com/gorilla/mux"
	"github.com/muesli/cache2go"
)

//go:embed html
var assets embed.FS

// WebFrontend wraps the web server and its associated state.
type WebFrontend struct {
	Port      uint16
	Hostname  string
	srv       http.Server
	router    *mux.Router
	log       *log.Logger
	tmpl      *template.Template
	isRunning bool         // nolint: unused
	lock      sync.RWMutex // nolint: unused
	suffixRe  *regexp.Regexp
	mimeTypes map[string]string
	hostCache *cache2go.CacheTable // nolint: unused
	dbPool    sync.Pool
	nexus     *backend.Nexus
}

// CreateFrontend creates a new web frontend.
func CreateFrontend(addr string, port uint16, nexus *backend.Nexus) (*WebFrontend, error) {
	var msg string
	var err error

	if common.Debug {
		fmt.Printf("Creating Guang Web Frontend to listen on %s:%d\n",
			addr, port)
	}

	frontend := &WebFrontend{
		Port: port,
		mimeTypes: map[string]string{
			"css": "text/css",
			"js":  "text/javascript",
			"png": "image/png",
		},
		nexus:    nexus,
		suffixRe: regexp.MustCompile("[.]([^.]+)$"),
	}

	if frontend.Hostname, err = os.Hostname(); err != nil {
		return nil, err
	} else if frontend.log, err = common.GetLogger("Web"); err != nil {
		return nil, err
	}

	frontend.router = mux.NewRouter()
	frontend.router.HandleFunc("/{pagename:(?:index|start|main)?$}", frontend.handleIndex)
	frontend.router.HandleFunc("/by_port", frontend.handleByPort)
	frontend.router.HandleFunc("/by_host", frontend.handleByHost)
	frontend.router.HandleFunc("/static/{file}", frontend.handleStaticFile)

	fmap := template.FuncMap{
		"sequence": sequenceFunc,
		"cycle":    cycleFunc,
		"now":      nowFunc,
	}

	frontend.tmpl = template.New("").Funcs(fmap)

	const tmplFolder = "html/templates"
	var templates []fs.DirEntry
	var tmplRe = regexp.MustCompile("[.]tmpl$")

	if templates, err = assets.ReadDir(tmplFolder); err != nil {
		frontend.log.Printf("[ERROR] Cannot read embedded templates: %s\n",
			err.Error())
		return nil, err
	}

	for _, entry := range templates {
		var (
			content []byte
			path    = filepath.Join(tmplFolder, entry.Name())
		)

		if !tmplRe.MatchString(entry.Name()) {
			continue
		} else if content, err = assets.ReadFile(path); err != nil {
			msg = fmt.Sprintf("Cannot read embedded file %s: %s",
				path,
				err.Error())
			frontend.log.Printf("[CRITICAL] %s\n", msg)
			return nil, errors.New(msg)
		} else if frontend.tmpl, err = frontend.tmpl.Parse(string(content)); err != nil {
			msg = fmt.Sprintf("Could not parse template %s: %s",
				entry.Name(),
				err.Error())
			frontend.log.Println("[CRITICAL] " + msg)
			return nil, errors.New(msg)
		}

		frontend.log.Printf("[TRACE] Template \"%s\" was parsed successfully.\n",
			entry.Name())
	}

	frontend.srv.Addr = fmt.Sprintf("%s:%d", addr, port)
	frontend.srv.ErrorLog = frontend.log
	frontend.srv.Handler = frontend.router

	frontend.dbPool = sync.Pool{
		New: func() interface{} {
			var err error
			var db *database.HostDB
			if db, err = database.OpenDB(common.DbPath); err != nil {
				return nil
			}

			return db
		},
	}

	for i := 0; i < 3; i++ {
		var db *database.HostDB
		if db, err = database.OpenDB(common.DbPath); err != nil {
			frontend.log.Printf("Error opening database: %s\n", err.Error())
			return nil, err
		}

		frontend.dbPool.Put(db)
	}

	return frontend, nil
} // func CreateFrontend(addr string, port uint16) (*WebFrontend, error)

// Serve runs the web server.
func (srv *WebFrontend) Serve() {
	srv.log.Println("The web server is starting to accept requests now.")
	http.Handle("/", srv.router)
	srv.srv.ListenAndServe() // nolint: errcheck
} // func (srv *WebFrontend) Serve()

func (srv *WebFrontend) getDB() (*database.HostDB, error) {
	var tmp = srv.dbPool.Get()

	if tmp == nil {
		return nil, nil
	}

	switch item := tmp.(type) {
	case *database.HostDB:
		return item, nil
	default:
		var msg string = fmt.Sprintf("Unexptected type came out of the HostDB pool: %T", tmp)
		srv.log.Println(msg)
		return nil, errors.New(msg)
	}
} // func (srv *WebFrontend) getDB() (*HostDB, error)

func (srv *WebFrontend) putDB(db *database.HostDB) {
	srv.dbPool.Put(db)
} // func (srv *WebFrontend) PutDB(db *backend.HostDB)

func (srv *WebFrontend) handleIndex(w http.ResponseWriter, request *http.Request) {
	var db *database.HostDB
	var tmpl *template.Template
	var err error
	var msg string
	var indexData tmplDataIndex = tmplDataIndex{
		Title: "Guang Web Frontend",
		Error: make([]string, 0),
	}

	if common.Debug {
		srv.log.Printf("Handling request for %s", request.RequestURI)
	}

	if db, err = srv.getDB(); err != nil {
		msg = fmt.Sprintf("Error getting database from pool: %s", err.Error())
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	defer srv.putDB(db)
	// if common.Debug {
	// 	srv.log.Println("Got database from Pool")
	// }

	if common.Debug {
		srv.log.Println("Getting generator count")
	}
	indexData.HostGenCnt = srv.nexus.GetGeneratorCount()
	if common.Debug {
		srv.log.Println("Getting Scanner count")
	}
	indexData.ScanCnt = srv.nexus.GetScannerCount()
	if common.Debug {
		srv.log.Println("Getting XFR count")
	}
	indexData.XFRCnt = srv.nexus.GetXFRCount()

	if common.Debug {
		srv.log.Println("Getting host count from database.")
	}
	if indexData.HostCnt, err = db.HostGetCount(); err != nil {
		msg = fmt.Sprintf("Error getting number of hosts: %s", err.Error())
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if indexData.PortReplyCnt, err = db.PortGetReplyCount(); err != nil {
		msg = fmt.Sprintf("Error getting number of scanned ports: %s", err.Error())
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	if common.Debug {
		srv.log.Println("Looking up template")
	}
	if tmpl = srv.tmpl.Lookup("index"); tmpl == nil {
		msg = "Template 'index' was not found!"
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
	} else {
		w.WriteHeader(200)
		if err = tmpl.Execute(w, indexData); err != nil {
			msg = fmt.Sprintf("Error rendering template or sending output to client: %s",
				err.Error())
			srv.log.Println(msg)
		} else if common.Debug {
			srv.log.Println("We sure showed THAT client a nice index!")
		}
	}
} // func (srv *WebFrontend) HandleIndex(w http.ResponseWriter, request *http.Request)

func (srv *WebFrontend) handleByPort(w http.ResponseWriter, request *http.Request) {
	var err error
	var msg string
	var db *database.HostDB
	var tmplData tmplDataByPort
	var dbRes []data.ScanResult
	var tmpl *template.Template

	if common.Debug {
		srv.log.Printf("Handling request for %s\n", request.RequestURI)
	}

	if db, err = srv.getDB(); err != nil {
		msg = fmt.Sprintf("Error getting database from pool: %s", err.Error())
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	defer srv.putDB(db)

	if dbRes, err = db.PortGetOpen(); err != nil {
		msg = fmt.Sprintf("Error getting list of open ports: %s", err.Error())
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
		return
	} else if tmpl = srv.tmpl.Lookup("by_port"); tmpl == nil {
		msg = "Template 'by_port' was not found!"
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
		return
	} else {
		tmplData = tmplDataByPort{
			Debug: common.Debug,
			Title: "Hosts by Port",
			Error: make([]string, 0),
			Count: len(dbRes),
		}

		if common.Debug {
			srv.log.Println("*Trying* to sort results.")
		}
		results := make(map[uint16]reportInfoPort)

		for _, res := range dbRes {
			if _, found := results[res.Port]; !found {
				results[res.Port] = reportInfoPort{
					Port:    res.Port,
					Results: make([]data.ScanResult, 0),
				}
			}

			info := results[res.Port]
			info.Results = append(info.Results, res)
			results[res.Port] = info
		}

		tmplData.Ports = results

		w.WriteHeader(200)
		if err = tmpl.Execute(w, tmplData); err != nil {
			msg = fmt.Sprintf("Error rendering template or sending output to client: %s",
				err.Error())
			srv.log.Println(msg)
		}
	}
} // func (srv *WebFrontend) HandleByPort(w http.ResponseWriter, request *http.Request)

func (srv *WebFrontend) handleByHost(w http.ResponseWriter, request *http.Request) {
	var err error
	var msg string
	var db *database.HostDB
	var data tmplDataByHost
	var tmpl *template.Template

	if common.Debug {
		srv.log.Printf("Handling request for %s\n", request.RequestURI)
	}

	if db, err = srv.getDB(); err != nil {
		msg = fmt.Sprintf("Error getting database from Pool: %s", err.Error())
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	defer srv.putDB(db)

	data = tmplDataByHost{
		Title: "Scanned Ports by Host",
		Debug: common.Debug,
		Error: make([]string, 0),
		Count: len(data.Hosts),
	}

	if data.Hosts, err = db.HostGetByHostReport(); err != nil {
		msg = fmt.Sprintf("Error getting open ports grouped by Host: %s",
			err.Error())
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	if tmpl = srv.tmpl.Lookup("by_host"); tmpl == nil {
		msg = "Error: Template 'by_host' was not found!"
		srv.log.Println(msg)
		srv.sendErrorMessage(w, msg)
		return
	}

	w.WriteHeader(200)
	if err = tmpl.Execute(w, data); err != nil {
		msg = fmt.Sprintf("Error rendering template or sending output to client: %s",
			err.Error())
		srv.log.Println(msg)
	}
} // func (srv *WebFrontend) HandleByHost(w http.ResponseWriter, request *http.Request)

// Deliver a static file to the client.
// Currently, all templates and static "files" are actually compiled into the binary,
// so there is no actual "file" access involved.
func (srv *WebFrontend) handleStaticFile(w http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	filename := vars["file"]

	// FIXED - Ich muss irgendwie noch den MIME-Typen ermitteln und mit
	//         übergeben, sonst nimmt der Browser zumindest den Stylesheet nicht an.
	//
	//         Da ich ja nur ein paar Dateien habe, habe ich mir eine map gebaut,
	//         die den Dateinamensendungen den jeweiligen MIME-Typen zuweist.
	//         Nicht besonders elegant, aber funktioniert.
	var mimeType string
	// vars := mux.Vars(request)
	// filename := vars["file"]

	if common.Debug {
		srv.log.Printf("Delivering static file %s to client\n", filename)
	}

	var match []string

	if match = srv.suffixRe.FindStringSubmatch(filename); match == nil {
		mimeType = "text/plain"
	} else if mime, ok := srv.mimeTypes[match[1]]; ok {
		mimeType = mime
	} else {
		srv.log.Printf("Kein MIME-Type gefunden fÃ¼r %s\n", filename)
	}

	w.Header().Set("Content-Type", mimeType)

	var (
		err  error
		fh   fs.File
		path = filepath.Join("html", "static", filename)
	)

	if fh, err = assets.Open(path); err != nil {
		msg := fmt.Sprintf("ERROR - cannot find file %s", path)
		srv.sendErrorMessage(w, msg)
		return
	}

	defer fh.Close() // nolint: errcheck

	w.WriteHeader(200)
	io.Copy(w, fh) // nolint: errcheck
} // func (srv *WebFrontend) HandleStaticFile(w http.ResponseWriter, request *http.Request)

// Meant for cases where something went wrong, render and deliver a simple HTML
// document with an error message to the client.
func (srv *WebFrontend) sendErrorMessage(w http.ResponseWriter, msg string) {
	const html = `
<!DOCTYPE html>
<html>
  <head>
    <title>Internal Error</title>
  </head>
  <body>
    <h1>Internal Error</h1>
    <hr />
    We are sorry to inform you an internal application error has occured:<br />
    %s
    <hr />
    &copy; 2016 <a href="mailto:krylon@gmx.net">Benjamin Walkenhorst</a>
  </body>
</html>
`

	srv.log.Println(msg)

	output := fmt.Sprintf(html, msg)
	w.WriteHeader(500)
	_, _ = w.Write([]byte(output))
} // func (srv *WebFrontend) SendErrorMessage(msg string, w http.ResponseWriter)
