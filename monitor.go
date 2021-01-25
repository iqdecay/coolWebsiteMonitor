package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

type HTTPResponse struct {
	responseTime time.Duration
	responseCode int // HTTP response code
}

// Holds the statistics for a given amount of time (e.g. last 10 min)
type WebsiteStatistics struct {
	mu                 sync.RWMutex
	lastResponses      []HTTPResponse
	statusCodeCount    map[int]int
	lastAvailabilities float32
	responseTimeSum    time.Duration
	maxResponseTime    time.Duration
	// The number of responses to store for the given amount of time
	maxSize     int64
	currentSize int64
}

func newStatistics(duration time.Duration, interval time.Duration) *WebsiteStatistics {
	w := new(WebsiteStatistics)
	w.maxSize = int64(duration / interval)
	w.statusCodeCount = make(map[int]int)
	return w
}

// Return the average availability over the period, in percent
func (w *WebsiteStatistics) getAvailability() float32 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return 100.0 * w.lastAvailabilities / float32(w.currentSize)
}
func (w *WebsiteStatistics) getAvgResponseTime() time.Duration {
	w.mu.RLock()
	defer w.mu.RUnlock()
	durationNs := w.responseTimeSum.Nanoseconds()
	return time.Duration(float32(durationNs) / float32(w.currentSize))
}

func (w *WebsiteStatistics) update(r HTTPResponse) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// The queue is full, discard least recent
	if w.currentSize == w.maxSize {
		discard := w.lastResponses[0]
		w.responseTimeSum -= discard.responseTime
		w.lastResponses = w.lastResponses[1:]
		w.statusCodeCount[discard.responseCode]--
		if discard.responseCode != http.StatusServiceUnavailable {
			w.lastAvailabilities--
		}
	} else {
		w.currentSize += 1
	}
	if r.responseTime > w.maxResponseTime {
		w.maxResponseTime = r.responseTime
	}
	w.lastResponses = append(w.lastResponses, r)
	_, ok := w.statusCodeCount[r.responseCode]
	if ok {
		w.statusCodeCount[r.responseCode]++
	} else {
		w.statusCodeCount[r.responseCode] = 1
	}
	if r.responseCode != http.StatusServiceUnavailable {
		w.lastAvailabilities++
	}
	w.responseTimeSum += r.responseTime
}

type Alert struct {
	availability float32
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

func newMonitor(param WebsiteParameter) *WebsiteMonitor {
	m := new(WebsiteMonitor)
	m.interval = param.interval
	m.url = param.url
	m.last2Min = newStatistics(2*time.Minute, m.interval)
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
	past2MinAvail := m.last2Min.getAvailability()
	// Website newly down
	if past2MinAvail < 80.0 && !m.isDown {
		m.alerts <- Alert{
			availability: past2MinAvail,
			since:        time.Now(),
			isDown:       true,
		}
	} else if past2MinAvail >= 80.0 && m.isDown {
		// Website recovered
		m.alerts <- Alert{
			availability: past2MinAvail,
			since:        time.Now(),
			isDown:       false,
		}
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
