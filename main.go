package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
)

const DateFormat = "2006-01-02 15:04:05"

// Format duration to avoid unnecessary precision
func formatDuration(duration time.Duration) string {
	return duration.Round(time.Millisecond).String()
}

func main() {
	parameters := parseParameterFile()
	monitors := make(map[string]*WebsiteMonitor)
	var domains = make([]string, 0)
	alerts := make(chan Alert)
	done := make(chan bool)

	// Initialize monitoring
	for _, v := range parameters {
		param := v
		m := newMonitor(param, alerts)
		domain := param.url[strings.Index(param.url, "//")+2:]
		monitors[domain] = m
		domains = append(domains, domain)
		// The "done" channel here is not useful since shared, see README
		go m.monitor(done)
	}

	// Init ui
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		panic(err)
	}
	defer g.Close()
	g.Cursor = true
	g.Mouse = true
	g.SetManagerFunc(layout)
	err = initKeyBindings(g)
	if err != nil {
		panic(err)
	}
	// Every 10 seconds, poll the values of the last 10 minutes
	go func() {
		tenSecTicker := time.NewTicker(10 * time.Second)
		defer tenSecTicker.Stop()
		for {
			<-tenSecTicker.C
			for _, domain := range domains {
				m := monitors[domain]
				avgResp := formatDuration(m.last10Min.getAvgResponseTime())
				maxResp := formatDuration(m.last10Min.getMaxResponseTime())
				availability := m.last10Min.getAvailability()
				line := fmt.Sprintf("last 10min %-30s : avg %-7s max %-7s avail %.0f%% ",
					domain, avgResp, maxResp, availability)
				displayLine(g, "logs", line)
			}
		}
	}()
	//Every minute, poll the values of the past hour
	go func() {
		minuteTicker := time.NewTicker(time.Minute)
		defer minuteTicker.Stop()
		for {
			<-minuteTicker.C
			for _, domain := range domains {
				m := monitors[domain]
				avgResp := formatDuration(m.lastHour.getAvgResponseTime())
				maxResp := formatDuration(m.lastHour.getMaxResponseTime())
				availability := m.lastHour.getAvailability()
				line := fmt.Sprintf("last hour %-30s : avg %-7s max %-7s avail %.0f%% ",
					domain, avgResp, maxResp, availability)
				displayLine(g, "logs", line)
			}
		}
	}()
	//Handle incoming alerts
	go func() {
		for {
			a := <-alerts
			var line string
			if a.isDown {
				line = fmt.Sprintf("Website %s is down. availability=%.0f%%, time=%v",
					a.url, a.availability, a.since.Format(DateFormat))
			} else {
				line = fmt.Sprintf("Website %s recovered. availability=%.0f%%, time=%v",
					a.url, a.availability, a.since.Format(DateFormat))
			}
			displayLine(g, "alerts", line)
		}
	}()
	if err := g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		panic(err)
	}
}
