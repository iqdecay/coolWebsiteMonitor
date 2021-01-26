package main

import (
	"log"
	"time"
)

const DATE_FORMAT = "2006-01-02 15:04:05"

func main() {
	parameters := parseParameterFile()
	monitors := make(map[string]*WebsiteMonitor)
	var urls = make([]string, 0)
	alerts := make(chan Alert)

	// Initialize monitoring
	for _, v := range parameters {
		param := v
		m := newMonitor(param, alerts)
		monitors[param.url] = m
		urls = append(urls, param.url)
		go func() {
			m.monitor()
		}()
	}

	go func() {
		// Every 10 seconds, poll the values of the last minute
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
			log.Printf("-------------------------------")
		}
	}()
	go func() {
		// Every minute, poll the values of the past hour
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
			}
			log.Printf("-------------------------------")
		}
	}()
	// Handle incoming alerts
	for {
		a := <-alerts
		if a.isDown {
			log.Printf("Website %s is down. availability=%.0f%%, time=%v",
				a.url, a.availability, a.since.Format(DATE_FORMAT))
		} else {
			log.Printf("Website %s recovered. availability=%.0f%%, time=%v",
				a.url, a.availability, a.since.Format(DATE_FORMAT))
		}
	}
}
