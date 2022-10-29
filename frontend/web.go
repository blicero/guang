// /Users/krylon/go/src/guang/frontend/web.go
// -*- coding: utf-8; mode: go; -*-
// Created on 06. 02. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2022-10-29 19:23:48 krylon>

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
	"time"

	"github.com/blicero/guang/backend"
	"github.com/blicero/guang/common"
	"github.com/blicero/guang/data"
	"github.com/blicero/guang/database"

	"github.com/gorilla/mux"
	"github.com/muesli/cache2go"
)

//go:embed html
var assets embed.FS

const DEFAULT_CACHE_TIMEOUT = time.Minute * 5

type WebFrontend struct {
	Port       uint16
	Hostname   string
	srv        http.Server
	router     *mux.Router
	log        *log.Logger
	tmpl       *template.Template
	isRunning  bool         // nolint: unused
	lock       sync.RWMutex // nolint: unused
	suffix_re  *regexp.Regexp
	mime_map   map[string]string
	host_cache *cache2go.CacheTable // nolint: unused
	db_pool    sync.Pool
	nexus      *backend.Nexus
}

type tmpl_data_index struct {
	Debug        bool
	Title        string
	Error        []string
	HostGenCnt   int
	XFRCnt       int
	ScanCnt      int
	HostCnt      int64
	PortReplyCnt int64
}

type report_info_port struct {
	Port    uint16
	Results []data.ScanResult
}

// type host_scan_result struct {
// 	Host  *data.Host
// 	Ports []data.ScanResult
// }

type tmpl_data_by_port struct {
	Debug bool
	Title string
	Error []string
	Count int
	Ports map[uint16]report_info_port
}

// Donnerstag, 18. 08. 2016, 21:10
// Damit ich das in der HTML-Template gescheit verarbeiten kann, müsste ich
// eigentlich eine Liste von Strukturen haben, wo der Host und die Ports drin
// liegen.
// Oder? Ich könnte eine Methode schreiben, die den Host anhand der ID zurück
// gibt? In den Rohdaten aus der Datenbank steht der ja drin.
type tmpl_data_by_host struct {
	Debug bool
	Title string
	Error []string
	Count int
	Hosts []data.HostWithPorts
}

type ResultListByHost []data.ScanResult

func (r ResultListByHost) Len() int {
	return len(r)
} // func (r ResultListByHost) Len() int

func (r ResultListByHost) Swap(i, j int) {
	tmp := r[i]
	r[i] = r[j]
	r[j] = tmp
} // func (r ResultListByHost) Swap(i, j int)

func (r ResultListByHost) Less(i, j int) bool {
	return r[i].Port < r[j].Port
} // func (r ResultListByHost) Less(i, j int) bool

func CreateFrontend(addr string, port uint16, nexus *backend.Nexus) (*WebFrontend, error) {
	var msg string
	var err error

	if common.DEBUG {
		fmt.Printf("Creating Guang Web Frontend to listen on %s:%d\n",
			addr, port)
	}

	frontend := &WebFrontend{
		Port: port,
		mime_map: map[string]string{
			"css": "text/css",
			"js":  "text/javascript",
			"png": "image/png",
		},
		nexus:     nexus,
		suffix_re: regexp.MustCompile("[.]([^.]+)$"),
	}

	if frontend.Hostname, err = os.Hostname(); err != nil {
		return nil, err
	} else if frontend.log, err = common.GetLogger("Web"); err != nil {
		return nil, err
	}

	frontend.router = mux.NewRouter()
	frontend.router.HandleFunc("/{pagename:(?:index|start|main)?$}", frontend.HandleIndex)
	frontend.router.HandleFunc("/by_port", frontend.HandleByPort)
	frontend.router.HandleFunc("/by_host", frontend.HandleByHost)
	frontend.router.HandleFunc("/static/{file}", frontend.HandleStaticFile)

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

	frontend.db_pool = sync.Pool{
		New: func() interface{} {
			var err error
			var db *database.HostDB
			if db, err = database.OpenDB(common.DB_PATH); err != nil {
				return nil
			} else {
				return db
			}
		},
	}

	for i := 0; i < 3; i++ {
		var db *database.HostDB
		if db, err = database.OpenDB(common.DB_PATH); err != nil {
			frontend.log.Printf("Error opening database: %s\n", err.Error())
			return nil, err
		} else {
			frontend.db_pool.Put(db)
		}
	}

	return frontend, nil
} // func CreateFrontend(addr string, port uint16) (*WebFrontend, error)

func (self *WebFrontend) Serve() {
	self.log.Println("The web server is starting to accept requests now.")
	http.Handle("/", self.router)
	self.srv.ListenAndServe() // nolint: errcheck
} // func (self *WebFrontend) Serve()

func (self *WebFrontend) GetDB() (*database.HostDB, error) {
	var tmp = self.db_pool.Get()

	if tmp == nil {
		return nil, nil
	}

	switch item := tmp.(type) {
	case *database.HostDB:
		return item, nil
	default:
		var msg string = fmt.Sprintf("Unexptected type came out of the HostDB pool: %T", tmp)
		self.log.Println(msg)
		return nil, errors.New(msg)
	}
} // func (self *WebFrontend) GetDB() (*HostDB, error)

func (self *WebFrontend) PutDB(db *database.HostDB) {
	self.db_pool.Put(db)
} // func (self *WebFrontend) PutDB(db *backend.HostDB)

func (self *WebFrontend) HandleIndex(w http.ResponseWriter, request *http.Request) {
	var db *database.HostDB
	var tmpl *template.Template
	var err error
	var msg string
	var index_data tmpl_data_index = tmpl_data_index{
		Title: "Guang Web Frontend",
		Error: make([]string, 0),
	}

	if common.DEBUG {
		self.log.Printf("Handling request for %s", request.RequestURI)
	}

	if db, err = self.GetDB(); err != nil {
		msg = fmt.Sprintf("Error getting database from pool: %s", err.Error())
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
		return
	} else {
		defer self.PutDB(db)
		if common.DEBUG {
			self.log.Println("Got database from Pool")
		}
	}

	if common.DEBUG {
		self.log.Println("Getting generator count")
	}
	index_data.HostGenCnt = self.nexus.GetGeneratorCount()
	if common.DEBUG {
		self.log.Println("Getting Scanner count")
	}
	index_data.ScanCnt = self.nexus.GetScannerCount()
	if common.DEBUG {
		self.log.Println("Getting XFR count")
	}
	index_data.XFRCnt = self.nexus.GetXFRCount()

	if common.DEBUG {
		self.log.Println("Getting host count from database.")
	}
	if index_data.HostCnt, err = db.HostGetCount(); err != nil {
		msg = fmt.Sprintf("Error getting number of hosts: %s", err.Error())
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
		return
	} else if index_data.PortReplyCnt, err = db.PortGetReplyCount(); err != nil {
		msg = fmt.Sprintf("Error getting number of scanned ports: %s", err.Error())
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
		return
	}

	if common.DEBUG {
		self.log.Println("Looking up template")
	}
	if tmpl = self.tmpl.Lookup("index"); tmpl == nil {
		msg = "Template 'index' was not found!"
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
	} else {
		w.WriteHeader(200)
		if err = tmpl.Execute(w, index_data); err != nil {
			msg = fmt.Sprintf("Error rendering template or sending output to client: %s",
				err.Error())
			self.log.Println(msg)
		} else if common.DEBUG {
			self.log.Println("We sure showed THAT client a nice index!")
		}
	}
} // func (self *WebFrontend) HandleIndex(w http.ResponseWriter, request *http.Request)

func (self *WebFrontend) HandleByPort(w http.ResponseWriter, request *http.Request) {
	var err error
	var msg string
	var db *database.HostDB
	var tmpl_data tmpl_data_by_port
	var db_res []data.ScanResult
	var tmpl *template.Template

	_ = "breakpoint"

	if common.DEBUG {
		self.log.Printf("Handling request for %s\n", request.RequestURI)
	}
	if db, err = self.GetDB(); err != nil {
		msg = fmt.Sprintf("Error getting database from pool: %s", err.Error())
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
		return
	} else {
		defer self.PutDB(db)
	}

	if db_res, err = db.PortGetOpen(); err != nil {
		msg = fmt.Sprintf("Error getting list of open ports: %s", err.Error())
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
		return
	} else if tmpl = self.tmpl.Lookup("by_port"); tmpl == nil {
		msg = "Template 'by_port' was not found!"
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
		return
	} else {
		tmpl_data = tmpl_data_by_port{
			Debug: common.DEBUG,
			Title: "Hosts by Port",
			Error: make([]string, 0),
			Count: len(db_res),
			//Ports: make(map[uint16]report_info_port),
		}

		if common.DEBUG {
			self.log.Println("*Trying* to sort results.")
		}
		results := make(map[uint16]report_info_port)

		for _, res := range db_res {
			if _, found := results[res.Port]; !found {
				results[res.Port] = report_info_port{
					Port:    res.Port,
					Results: make([]data.ScanResult, 0),
				}
			}

			info := results[res.Port]
			info.Results = append(info.Results, res)
			results[res.Port] = info
		}

		tmpl_data.Ports = results

		w.WriteHeader(200)
		if err = tmpl.Execute(w, tmpl_data); err != nil {
			msg = fmt.Sprintf("Error rendering template or sending output to client: %s",
				err.Error())
			self.log.Println(msg)
		}
	}
} // func (self *WebFrontend) HandleByPort(w http.ResponseWriter, request *http.Request)

func (self *WebFrontend) HandleByHost(w http.ResponseWriter, request *http.Request) {
	var err error
	var msg string
	var db *database.HostDB
	var data tmpl_data_by_host
	var tmpl *template.Template

	if common.DEBUG {
		self.log.Printf("Handling request for %s\n", request.RequestURI)
	}

	if db, err = self.GetDB(); err != nil {
		msg = fmt.Sprintf("Error getting database from Pool: %s", err.Error())
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
		return
	} else {
		defer self.PutDB(db)
	}

	data = tmpl_data_by_host{
		Title: "Scanned Ports by Host",
		Debug: common.DEBUG,
		Error: make([]string, 0),
		Count: len(data.Hosts),
	}

	if data.Hosts, err = db.HostGetByHostReport(); err != nil {
		msg = fmt.Sprintf("Error getting open ports grouped by Host: %s",
			err.Error())
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
		return
	}

	if tmpl = self.tmpl.Lookup("by_host"); tmpl == nil {
		msg = "Error: Template 'by_host' was not found!"
		self.log.Println(msg)
		self.SendErrorMessage(w, msg)
		return
	}

	w.WriteHeader(200)
	if err = tmpl.Execute(w, data); err != nil {
		msg = fmt.Sprintf("Error rendering template or sending output to client: %s",
			err.Error())
		self.log.Println(msg)
	}
} // func (self *WebFrontend) HandleByHost(w http.ResponseWriter, request *http.Request)

// Deliver a static file to the client.
// Currently, all templates and static "files" are actually compiled into the binary,
// so there is no actual "file" access involved.
func (self *WebFrontend) HandleStaticFile(w http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	filename := vars["file"]

	// FIXED - Ich muss irgendwie noch den MIME-Typen ermitteln und mit
	//         übergeben, sonst nimmt der Browser zumindest den Stylesheet nicht an.
	//
	//         Da ich ja nur ein paar Dateien habe, habe ich mir eine map gebaut,
	//         die den Dateinamensendungen den jeweiligen MIME-Typen zuweist.
	//         Nicht besonders elegant, aber funktioniert.
	var mime_type string
	// vars := mux.Vars(request)
	// filename := vars["file"]

	if common.DEBUG {
		self.log.Printf("Delivering static file %s to client\n", filename)
	}

	var match []string

	if match = self.suffix_re.FindStringSubmatch(filename); match == nil {
		mime_type = "text/plain"
	} else if mime, ok := self.mime_map[match[1]]; ok {
		mime_type = mime
	} else {
		self.log.Printf("Kein MIME-Type gefunden fÃ¼r %s\n", filename)
	}

	w.Header().Set("Content-Type", mime_type)

	var (
		err  error
		fh   fs.File
		path = filepath.Join("html", "static", filename)
	)

	if fh, err = assets.Open(path); err != nil {
		msg := fmt.Sprintf("ERROR - cannot find file %s", path)
		self.SendErrorMessage(w, msg)
		return
	}

	defer fh.Close() // nolint: errcheck

	w.WriteHeader(200)
	io.Copy(w, fh) // nolint: errcheck
} // func (self *WebFrontend) HandleStaticFile(w http.ResponseWriter, request *http.Request)

// Meant for cases where something went wrong, render and deliver a simple HTML
// document with an error message to the client.
func (self *WebFrontend) SendErrorMessage(w http.ResponseWriter, msg string) {
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

	self.log.Println(msg)

	output := fmt.Sprintf(html, msg)
	w.WriteHeader(500)
	_, _ = w.Write([]byte(output))
} // func (self *WebFrontend) SendErrorMessage(msg string, w http.ResponseWriter)

////////////////////////////////////
// Functions for use in templates //
////////////////////////////////////

type generator struct {
	values []string
	index  int
	f      func(s []string, i int) string
}

func (seq *generator) Next() string {
	s := seq.f(seq.values, seq.index)
	seq.index++
	return s
} // func (seq *generator) Next() string

func sequenceGen(values []string, i int) string {
	if i >= len(values) {
		return values[len(values)-1]
	} else {
		return values[i]
	}
} // func sequenceGen(values []string, i int) string

func cycleGen(values []string, i int) string {
	return values[i%len(values)]
} // func cycleGen(values []string, i int) string

func sequenceFunc(values ...string) (*generator, error) {
	if len(values) == 0 {
		return nil, errors.New("Sequence must have at least one element")
	} else {
		return &generator{
			values: values,
			index:  0,
			f:      sequenceGen,
		}, nil
	}
} // func sequenceFunc(values ...string) (*generator, error)

func cycleFunc(values ...string) (*generator, error) {
	if len(values) == 0 {
		return nil, errors.New("Cycle must have at least one element")
	} else {
		return &generator{
			values: values,
			index:  0,
			f:      cycleGen,
		}, nil
	}
} // func cycleFunc(values ...string) (*generator, error)

func nowFunc() string {
	return time.Now().Format("2006-01-02 15:04:05")
} // func nowFunc() string
