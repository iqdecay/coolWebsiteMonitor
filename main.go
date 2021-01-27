package main

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"strings"
	"time"
)

const DATE_FORMAT = "2006-01-02 15:04:05"

// Format duration to avoid unnecessary precision
func formatDuration(duration time.Duration) string {
	durationAsString := duration.String()
	index := strings.IndexRune(durationAsString, 'n')
	// if the time is in nanoseconds
	if index != - 1 {
		return duration.Round(time.Nanosecond).String()
	} else {
		return duration.Round(time.Millisecond).String()
	}
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
		go func() {
			m.monitor(done)
		}()
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
				m.last10Min.mu.RLock()
				avgResp := formatDuration(m.last10Min.getAvgResponseTime())
				maxResp := formatDuration(m.last10Min.maxResponseTime)
				line := fmt.Sprintf("last 10 min %-30s : avg %-7s max %-7s avail %.0f%% ",
					domain, avgResp, maxResp, m.last10Min.getAvailability())
				m.last10Min.mu.RUnlock()
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
				m.lastHour.mu.RLock()
				avgResp := formatDuration(m.lastHour.getAvgResponseTime())
				maxResp := formatDuration(m.lastHour.maxResponseTime)
				line := fmt.Sprintf("last hour %-30s : avg %-7s max %-7s avail %.0f%% ",
					domain, avgResp, maxResp, m.lastHour.getAvailability())
				m.lastHour.mu.RUnlock()
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
					a.url, a.availability, a.since.Format(DATE_FORMAT))
			} else {
				line = fmt.Sprintf("Website %s recovered. availability=%.0f%%, time=%v",
					a.url, a.availability, a.since.Format(DATE_FORMAT))
			}
			displayLine(g, "alerts", line)
		}
	}()
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		panic(err)
	}
}
