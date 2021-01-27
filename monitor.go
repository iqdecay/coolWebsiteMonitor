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
	req, _ := http.NewRequest("HEAD", url, nil)
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
func (m *WebsiteMonitor) checkForAlerts(s *WebsiteStatistics) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.getAge() < s.windowSize {
		return
	}
	pastPeriodAvail := s.getAvailability()
	alert := Alert{
		availability: pastPeriodAvail,
		url:          m.url,
		since:        time.Now(),
	}
	if pastPeriodAvail < 80.0 && !m.isDown {
		// Website newly down
		m.isDown = true
		alert.isDown = true
		m.alerts <- alert
	} else if pastPeriodAvail >= 80.0 && m.isDown {
		// Website recovered
		m.isDown = false
		alert.isDown = false
		m.alerts <- alert
	}
}

// Main monitoring loop, update each of the WebsiteStatistics
func (m *WebsiteMonitor) monitor(done chan bool) {
	a := Alert{}
	a.isDown = false
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	alertsTicker := time.NewTicker(m.interval)
	defer alertsTicker.Stop()
	alertsDone := make(chan bool)
	go func() {
		for {
			select {
			case <-alertsTicker.C:
				//	Need a separate routine, because if the website is down,
				//	the response might not come right away
				// It might create false positives if the website is slow to answer
				m.checkForAlerts(m.last2Min)
			case <-alertsDone:
				return
			}
		}
	}()
	for {
		select {
		case <-ticker.C:
			lastPerf := getPerformance(m.url)
			m.last2Min.update(lastPerf)
			m.last10Min.update(lastPerf)
			m.lastHour.update(lastPerf)
		case <-done:
			alertsDone <- true
			return
		}
	}
}
