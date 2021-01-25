package main

import (
	"log"
	"time"
)

const DATE_FORMAT = "2006-01-02 15:04:05"

func main() {
	parameters := parseParameterFile()
	monitors := make(map[string]*WebsiteMonitor)
	alerts := make(chan Alert)

	for _, v := range parameters {
		param := v
		m := newMonitor(param, alerts)
		monitors[param.url] = m
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
			for url, m := range monitors {
				log.Printf("Last 10 min :%s : %d %v resp time  %.0f%% avail %v max",
					url, m.last10Min.currentSize, m.last10Min.getAvgResponseTime(),
					m.last10Min.getAvailability(), m.last10Min.maxResponseTime)
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
			for url, m := range monitors {
				log.Printf("Last hour : %s : %d %v resp time  %.0f%% avail %v max",
					url, m.lastHour.currentSize, m.lastHour.getAvgResponseTime(),
					m.lastHour.getAvailability(), m.lastHour.maxResponseTime)
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
