// Time-stamp: <2022-11-08 00:31:32 krylon>

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
                                  for (const [port, responses] of Object.entries(response.Results)) {
                                      const tid = `#tbody_${port}`
                                      const tbody = $(tid)[0]

                                      for (const r of responses.values()) {
                                          console.log(r)

                                          // Eventually, I will have to think about how to render that timestamp properly.
                                          const row = `<tr>
                 <td>${r.Host.Name} (${r.Host.Address})</td>
                 <td>${r.Stamp}</td>
                 <td><pre>${r.Reply}</pre></td>
                 </tr>`

                                          tbody.innerHTML += row
                                      }
                                  }

                                  updateStamp = timeStampUnix()
                              }
                          },
                          'json'
                         ).fail((reply, status, text) => {
                             const msg = `Failed to load update: ${status} -- ${reply} -- ${text}`
                             console.log(msg)
                             alert(msg)
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
        console.log(`Invalid argument: ${val} is not an integer`)
    }
} // function updateIntervalSet ()
