package main

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"time"
)

const DATE_FORMAT = "2006-01-02 15:04:05"

func main() {
	parameters := parseParameterFile()
	monitors := make(map[string]*WebsiteMonitor)
	var urls = make([]string, 0)
	alerts := make(chan Alert)
	done := make(chan bool)

	// Initialize monitoring
	for _, v := range parameters {
		param := v
		m := newMonitor(param, alerts)
		monitors[param.url] = m
		urls = append(urls, param.url)
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
	// Every 10 seconds, poll the values of the last minute
	go func() {
		tenSecTicker := time.NewTicker(10 * time.Second)
		defer tenSecTicker.Stop()
		for {
			<-tenSecTicker.C
			for _, url := range urls {
				m := monitors[url]
				m.last10Min.mu.RLock()
				line := fmt.Sprintf("Last 10 min :%s : %d %v resp time  %.0f%% avail %v max",
					url, m.last10Min.currentSize, m.last10Min.getAvgResponseTime(),
					m.last10Min.getAvailability(), m.last10Min.maxResponseTime)
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
			for _, url := range urls {
				m := monitors[url]
				m.lastHour.mu.RLock()
				line := fmt.Sprintf("Last hour : %s : %d %v resp time  %.0f%% avail %v max",
					url, m.lastHour.currentSize, m.lastHour.getAvgResponseTime(),
					m.lastHour.getAvailability(), m.lastHour.maxResponseTime)
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
