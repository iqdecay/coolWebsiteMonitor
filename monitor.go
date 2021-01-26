package main

import (
	"log"
	"net/http"
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
	// TODO : change for testing
	m.last2Min = newStatistics(30*time.Second, m.interval)
	m.last10Min = newStatistics(10*time.Minute, m.interval)
	m.lastHour = newStatistics(1*time.Hour, m.interval)
	return m
}

// TODO : change to have the actual performance (TTFB) and not just RTT
func getPerformance(url string) HTTPResponse {
	start := time.Now()
	r, err := http.Get(url)
	if err != nil {
		log.Fatalf("While fetching %s: %v", url, err)
	}
	return HTTPResponse{
		responseTime: time.Since(start),
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
		//log.Println(t.Format("05.999"), m.url, lastPerf.responseCode,
		//	lastPerf.responseTime)
	}
}

//func monitor(url string) {
//	var dns, tlsHandshake *time.Time
//	var connect = new(time.Time)
//	var start = new(time.Time)
//	var dns = new(time.Time)
//	trace := &httptrace.ClientTrace{
//		DNSStart: func(dsi httptrace.DNSStartInfo) { *dns = time.Now() },
//		DNSDone: func(ddi httptrace.DNSDoneInfo) {
//			fmt.Printf("DNS Done: %v\n", time.Since(*dns))
//		},
//		TLSHandshakeStart: func() {
//			*tlsHandshake = time.Now()
//			fmt.Printf("TLSHandshake started %s\n", url)
//		},
//		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
//			fmt.Printf("%s TLS Handshake: %v\n", url, time.Since(*tlsHandshake))
//		},
//		ConnectStart: func(network, addr string) { *connect = time.Now() },
//		ConnectDone: func(network, addr string, err error) {
//			fmt.Printf("%s Connect time: %v\n", url, time.Since(*connect))
//		},
//		GotFirstResponseByte: func() {
//			fmt.Printf("%s Time from start to first byte: %v\n", url, time.Since(*start))
//		},
//	}
//	req, _ := http.NewRequest("GET", url, nil)
//	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
//	*start = time.Now()
//	if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
//		log.Fatal(err)
//	}
//	//url = "https://github.com"
//	//fmt.Printf("URL was changed : origin url %s, url %s\n", or, url)
//	//start = time.Now()
//	//if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
//	//	log.Fatal(err)
//	//}
//	fmt.Printf("%s Total time: %v\n", url, time.Since(*start))
//}
