package main

import (
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
	w.responseTimeSum = time.Duration(0)
	w.maxResponseTime = time.Duration(0)
	return w
}

// Return the average availability over the period, in percent
func (w *WebsiteStatistics) getAvailability() float32 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.currentSize == 0 {
		return 0
	}
	return 100.0 * w.lastAvailabilities / float32(w.currentSize)
}
func (w *WebsiteStatistics) getAvgResponseTime() time.Duration {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.currentSize == 0 {
		return time.Duration(0)
	}
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
		// Do O(n) search for the new max, can be avoided but
		// not without useless overhead (new datastructure)
		if discard.responseTime == w.maxResponseTime {
			newMax := time.Duration(0)
			for _, r := range w.lastResponses {
				if r.responseTime > newMax {
					newMax = r.responseTime
				}
			}
			w.maxResponseTime = newMax
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
