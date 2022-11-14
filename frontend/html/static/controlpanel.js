// /home/krylon/go/src/github.com/blicero/guang/frontend/html/static/controlpanel.js
// -*- mode: javascript; coding: utf-8; -*-
// Time-stamp: <2022-11-14 22:06:05 krylon>
// Copyright 2022 Benjamin Walkenhorst

'use strict'

var count = {
    'Generator': 0,
    'Scanner': 0,
    'XFR': 0,
}

const cntID = {
    'Generator': '#cnt_gen',
    'Scanner': '#cnt_scan',
    'XFR': '#cnt_xfr',
}

function spawn(fac) {
    const addr = `/ajax/spawn_worker/${facilities[fac]}/1`

    const req = $.get(addr,
                      {},
                      (res) => {
                          if (res.Status) {
                              const counterID = cntID[fac]
                              // Update panel?
                              $(counterID)[0].innerHTML = res.NewCnt
                          } else {
                              alert(res.Message)
                          }
                      },
                      'json'
                     ).fail((reply, status, txt) => {
                         const msg = `Failed to load update: ${status} -- ${reply} -- ${text}`
                         console.log(msg)
                         alert(msg)
                     })
} // function spawn(fac)

function stop(fac) {
    const addr = `/ajax/stop_worker/${facilities[fac]}/1`

    const req = $.get(
        addr,
        {},
        (res) => {
            if (res.Status) {
                const counterID = cntID[fac]

                // Update panel
                $(counterID)[0].innerHTML = res.NewCnt
            } else {
                alert(res.Message)
            }
        },
        'json'
    ).fail((reply, status, txt) => {
        const msg = `Failed to load update: ${status} -- ${reply} -- ${text}`
        console.log(msg)
        alert(msg)
    })
} // function stop(fac)

function loadWorkerCount() {
    const addr = '/ajax/worker_count'
    
    let req = $.get(
        addr,
        {},
        (res) => {
            if (res.Status) {
                for (const [fac, id] of Object.entries(cntID)) {
                    $(id)[0].innerHTML = res[fac]
                }
            } else {
                const msg = `${res.Timestamp} - Error requesting worker count: ${res.Message}`
                console.log(msg)
                alert(msg)
            }
        },
        'json'
    ).fail((reply, status, text) => {
        const msg = `Failed to load worker count: ${status} -- ${reply} -- ${text}`
        console.log(msg)
        alert(msg)
    })
} // function loadWorkerCount()
