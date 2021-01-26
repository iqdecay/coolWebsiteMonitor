package main

import (
	"log"
	"net/http"
	"net/http/httptrace"
	"time"
)

type Alert struct {
	availability float32
	url          string
	since        time.Time
	isDown       bool // if false, the alert is for recovery
}

type WebsiteMonitor struct {
	last2Min  *WebsiteStatistics
	last10Min *WebsiteStatistics
	lastHour  *WebsiteStatistics
	isDown    bool
	alerts    chan Alert
	url       string
	interval  time.Duration
}

func newMonitor(param WebsiteParameter, alerts chan Alert) *WebsiteMonitor {
	m := new(WebsiteMonitor)
	m.interval = param.interval
	m.url = param.url
	m.alerts = alerts
	// TODO : change for production
	m.last2Min = newStatistics(30*time.Second, m.interval)
	m.last10Min = newStatistics(10*time.Minute, m.interval)
	m.lastHour = newStatistics(1*time.Hour, m.interval)
	return m
}

// Return the time to first byte and status code of an url
func getPerformance(url string) HTTPResponse {
	var start time.Time
	var ttfb = new(time.Duration)
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			*ttfb = time.Since(start)
		},
	}
	req, _ := http.NewRequest("GET", url, nil)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	r, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Fatalf("While fetching %s: %v", url, err)
	}
	return HTTPResponse{
		responseTime: *ttfb,
		responseCode: r.StatusCode,
	}
}
func (m *WebsiteMonitor) checkForAlerts() {
	m.last2Min.mu.RLock()
	defer m.last2Min.mu.RUnlock()
	// TODO : change for production
	if m.last2Min.getAge() < 30*time.Second {
		return
	}
	past2MinAvail := m.last2Min.getAvailability()
	alert := Alert{
		availability: past2MinAvail,
		url:          m.url,
		since:        time.Now(),
	}
	if past2MinAvail < 80.0 && !m.isDown {
		// Website newly down
		m.isDown = true
		alert.isDown = true
		m.alerts <- alert
	} else if past2MinAvail >= 80.0 && m.isDown {
		// Website recovered
		m.isDown = false
		alert.isDown = false
		m.alerts <- alert
	}
}

// Main monitoring loop, update each of the WebsiteStatistics
func (m *WebsiteMonitor) monitor() {
	a := Alert{}
	a.isDown = false
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		<-ticker.C
		lastPerf := getPerformance(m.url)
		m.last2Min.update(lastPerf)
		m.last10Min.update(lastPerf)
		m.lastHour.update(lastPerf)
		// TODO Issue if website is down : the response doesn't return, so
		// the check isn't made until after the response comes
		go m.checkForAlerts()
	}
}
