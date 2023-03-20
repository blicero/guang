// Time-stamp: <2023-03-20 19:37:14 krylon>

'use strict;'

let updateStamp = timeStampUnix()

function update_results() {
    try {
        if (!settings.update.active) {
            return
        }

        const addr = `/ajax/port_recent/${updateStamp}`
        const req = $.get(addr,
                          {},
                          (response) => {
                              if (response.Status) {
                                  let cntTotal = 0
                                  for (const [port, responses] of Object.entries(response.Results)) {
                                      const tid = `#tbody_${port}`
                                      const tbody = $(tid)[0]
                                      const cnt = responses.length

                                      const cell = $(`#port_cnt_${port}`)[0]
                                      const cntOld = parseInt(cell.innerText)
                                      const cntNew = cntOld + cnt

                                      cell.innerText = cntNew
                                      cntTotal += cnt

                                      for (const r of responses.values()) {
                                          console.log(r)

                                          // Eventually, I will have to think about how to render that timestamp properly.
                                          const row = `<tr>
                 <td>${r.Host.Name} (${r.Host.Address})</td>
                 <td>${r.Stamp}</td>
                 <td></td>
                 <td></td>
                 <td><pre>${r.Reply}</pre></td>
                 </tr>`

                                          tbody.innerHTML += row
                                      }
                                  }

                                  updateStamp = timeStampUnix()

                                  if (cntTotal > 0) {
                                      const cell = $('#toc_total')[0]
                                      const cntOld = parseInt(cell.innerText)
                                      const cntNew = cntOld + cntTotal
                                      cell.innerText = cntNew
                                  }
                              } else {
                                  appendMsg(res.Message)
                              }
                          },
                          'json'
                         ).fail((reply, status, text) => {
                             const msg = `Failed to load update: ${status} -- ${reply} -- ${text}`
                             console.log(msg)
                             // alert(msg)
                             appendMsg(msg)
                         })

    } finally {
        window.setTimeout(update_results, settings.update.interval)
    }
} // function update_results ()

function updateToggle () {
    settings.update.active = !settings.update.active
    saveSetting('update', 'active', settings.update.active)
} // function updateToggle ()

function updateIntervalSet (val) {
    if (Number.isInteger(val)) {
        settings.update.interval = val
        saveSetting('update', 'interval', val)
    } else {
        const errMsg = `Invalid argument: ${val} is not an integer`
        console.log(errMsg)
        appendMsg(errMsg)
    }
} // function updateIntervalSet ()

function updateIntervalEdit () {
    const interval = $('#update_interval_edit')[0].value * 1000
    updateIntervalSet(interval)
} // function updateIntervalEdit()
