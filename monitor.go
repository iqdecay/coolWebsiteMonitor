package main

import (
	"context"
	"errors"
	"io"
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

// Main monitoring device, specific to a website
type WebsiteMonitor struct {
	last2Min  *WebsiteStatistics
	last10Min *WebsiteStatistics
	lastHour  *WebsiteStatistics
	isDown    bool
	alerts    chan Alert
	url       string
	interval  time.Duration
}

// Create a monitor with 3 statistics collectors
func newMonitor(param UrlWatchParameters, alerts chan Alert) *WebsiteMonitor {
	m := new(WebsiteMonitor)
	m.interval = param.interval
	m.url = param.url
	m.alerts = alerts
	m.last2Min = newStatistics(2*time.Minute, m.interval)
	m.last10Min = newStatistics(10*time.Minute, m.interval)
	m.lastHour = newStatistics(1*time.Hour, m.interval)
	return m
}

// Return the time to first byte and status code of an url
func getPerformance(url string) UrlLastResponse {
	var start time.Time
	var ttfb time.Duration
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			ttfb = time.Since(start)
		},
	}
	req, _ := http.NewRequest("HEAD", url, nil)
	// A website taking more than 1m to answer is considered down
	ctx, cancel := context.WithTimeout(req.Context(), time.Minute)
	defer cancel()
	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))
	start = time.Now()
	r, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Printf("While fetching %s: %v", url, err)
		var responseType UrlResponseType
		if errors.Is(err, context.DeadlineExceeded) {
			responseType = DeadlineExceeded
		} else {
			// treat non-deadline errors as network failures
			responseType = NetworkFailure
		}
		return UrlLastResponse{
			responseCode: -1,
			responseTime: time.Minute,
			responseType: responseType,
		}
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing request body while fetching %s: %v", url, err)
		}
	}(r.Body)
	return UrlLastResponse{
		responseTime: ttfb,
		responseCode: r.StatusCode,
		responseType: ResponseReceived,
	}
}

// Check for alerts if the monitoring started a while ago
func (m *WebsiteMonitor) checkForAlerts(s *WebsiteStatistics) {
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
				//	Need a separate routine to avoid taking into account
				// long response times
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
