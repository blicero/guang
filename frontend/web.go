// /Users/krylon/go/src/guang/frontend/web.go
// -*- coding: utf-8; mode: go; -*-
// Created on 06. 02. 2016 by Benjamin Walkenhorst
// (c) 2016 Benjamin Walkenhorst
// Time-stamp: <2016-02-06 17:48:45 krylon>

//go:generate ./build_templates_go.pl

package frontend

import (
	"errors"
	"fmt"
	"guang/backend"
	"log"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/muesli/cache2go"
)

const DEFAULT_CACHE_TIMEOUT = time.Minute * 5

type WebFrontend struct {
	Port       uint16
	Hostname   string
	srv        http.Server
	router     *mux.Router
	log        *log.Logger
	tmpl       *template.Template
	isRunning  bool
	lock       sync.Mutex
	mime_map   map[string]string
	host_cache *cache2go.CacheTable
	db_pool    sync.Pool
}

type tmpl_data_index struct {
	Title        string
	Error        []string
	HostGenCnt   int
	XFRCnt       int
	ScanCnt      int
	HostCnt      int
	PortReplyCnt int
}

func CreateFrontend(addr string, port uint16) (*WebFrontend, error) {
	var msg string
	var err error

	if backend.DEBUG {
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
	}

	if frontend.Hostname, err = os.Hostname(); err != nil {
		return nil, err
	} else if srv.log, err = backend.GetLogger("Web"); err != nil {
		return nil, err
	}

	frontend.router = mux.NewRouter()
	frontend.router.HandleFunc("/{pagename:(?:index|start|main)?$}", frontend.HandleIndex)

	fmap := template.FuncMap{
		"sequence": sequenceFunc,
		"cycle":    cycleFunc,
		"now":      nowFunc,
	}

	frontend.tmpl = template.New("").Funcs(fmap)

	for name, body := range html_data.Templates {
		if frontend.tmpl, err = frontend.tmpl.Parse(body); err != nil {
			msg = fmt.Sprintf("Error parsing template %s: %s", name, err.Error())
			frontend.log.Println(msg)
			return nil, errors.New()
		}
	}

	frontend.srv.Addr = fmt.Sprintf("%s:%d", addr, port)
	frontend.srv.ErrorLog = frontend.Log
	frontend.srv.Handler = frontend.router

	frontend.db_pool = Sync.Pool{
		New: func() interface{} {
			var err error
			var db *HostDB
			if db, err = OpenDB(backend.DB_PATH); err != nil {
				return nil
			} else {
				return db
			}
		},
	}

	return frontend, nil
} // func CreateFrontend(addr string, port uint16) (*WebFrontend, error)

func (self *WebFrontend) HandleIndex(w http.ResponseWriter, request *http.Request) {

} // func (self *WebFrontend) HandleIndex(w http.ResponseWriter, request *http.Request)

// Deliver a static file to the client.
// Currently, all templates and static "files" are actually compiled into the binary,
// so there is no actual "file" access involved.
func (self *WebFrontend) HandleStaticFile(w http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	filename := vars["file"]
	if DEBUG {
		self.log.Printf("Delivering static file %s to client\n", filename)
	}

	if body, ok := html_data.Static[filename]; ok {
		w.WriteHeader(200)
		w.Write([]byte(body))
	} else {
		msg := fmt.Sprintf("ERROR - cannot find file %s", filename)
		self.SendErrorMessage(w, msg)
	}
} // func (self *WebFrontend) HandleStaticFile(w http.ResponseWriter, request *http.Request)

// Meant for cases where something went wrong, render and deliver a simple HTML
// document with an error message to the client.
func (self *WebFrontend) SendErrorMessage(w http.ResponseWriter, msg string) {
	html := `
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
    &copy; 2015 <a href="mailto:krylon@gmx.net">Benjamin Walkenhorst</a>
  </body>
</html>
`

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
