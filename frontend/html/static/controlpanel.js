// /home/krylon/go/src/github.com/blicero/guang/frontend/html/static/controlpanel.js
// -*- mode: javascript; coding: utf-8; -*-
// Time-stamp: <2022-11-09 19:19:21 krylon>
// Copyright 2022 Benjamin Walkenhorst

'use strict'

var count = {
    generator: 0,
    scanner: 0,
    xfr: 0,
}

function genCntChange() {
    const inputID = '#cnt_generator'
    const val = $(inputID)[0].value
    const addr = `/ajax/control/generator/cnt/${val}`

    const req = $.get(addr,
                      {},
                      (res) => {
                          if (res.Status) {
                              // Update panel?
                          }
                      },
                      'json'
                     ).fail((reply, status, txt) => {
                         const msg = `Failed to load update: ${status} -- ${reply} -- ${text}`
                         console.log(msg)
                         alert(msg)
                     })
} // function genCntChange()
