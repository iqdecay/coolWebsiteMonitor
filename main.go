package main

import (
	"log"
	"time"
)

const DATE_FORMAT = "2006-01-02 15:04:05"

func checkEveryMin()

func main() {
	parameters := getParameters()
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
	// Handle incoming alerts
	go func() {
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
	}()

	go func() {
		// Every 10 seconds, poll the values of the last minute
		tenSecTicker := time.NewTicker(10 * time.Second)
		defer tenSecTicker.Stop()
		for {
			<-tenSecTicker.C
			for url, monitor := range monitors {
				log.Printf("%s : %d %v resp time  %.0f%% avail %v max",
					url, monitor.last2Min.currentSize, monitor.last2Min.getAvgResponseTime(),
					monitor.last2Min.getAvailability(), monitor.last2Min.maxResponseTime)
			}
		}
	}()
	go func() {
		// Every minute, poll the values of the past hour
		minuteTicker := time.NewTicker(15 * time.Seconds)
		time.Sleep(time.Minute)
	}
}
