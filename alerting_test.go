package main

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestAlertQuick(t *testing.T) {
	checkAlerts(t, 50*time.Millisecond, 1*time.Second, time.Duration(2), time.Duration(4))
}

func TestAlert(t *testing.T) {
	checkAlerts(t, 100*time.Millisecond, 1*time.Second, time.Duration(2), time.Duration(4))
}
func TestAlertOdd(t *testing.T) {
	checkAlerts(t, 300*time.Millisecond, 1*time.Second, time.Duration(2), time.Duration(4))
}

func checkAlerts(t *testing.T, interval time.Duration, alertWindow time.Duration,
	dtStart time.Duration, dtEnd time.Duration) {
	parameters := WebsiteParameter{
		url:      "http://localhost:8080",
		interval: interval,
	}
	// What we want to test
	alertChannel := make(chan Alert)
	serverDone := make(chan bool)
	monitorDone := make(chan bool)
	localMonitor := newMonitor(parameters, alertChannel)

	// Mock for faster testing
	last1Sec := newStatistics(alertWindow, parameters.interval)
	localMonitor.last2Min = last1Sec

	go makeFaultyServer(alertWindow, dtStart, dtEnd, serverDone)
	go func() {
		localMonitor.monitor(monitorDone)
	}()

	start := time.Now()
	alertDown := dtStart*alertWindow + 2*alertWindow/10
	alertRecovered := dtEnd*alertWindow + 2*alertWindow/10
	tolerance := 1 * time.Second

	// Check the content of alert at a precise time
	select {
	case a := <-alertChannel:
		since := time.Since(start)
		if alertDown < since && since < alertDown+tolerance {
			if !a.isDown {
				t.Error("Alert received : got 'recovered' want 'unavailable'")
			} else {
				log.Println("Received down alert at the right time")
			}
		} else {
			t.Errorf("Down alert received at %v, want %v", since, alertDown)
		}
	case <-time.After(2 * dtEnd * alertWindow):
		t.Errorf("No down alerts received, timeout")
	}
	select {

	case a := <-alertChannel:
		since := time.Since(start)
		if alertRecovered < since && since < alertRecovered+tolerance {
			if a.isDown {
				t.Error("Alert received : got 'unavailable' want 'recovered'")
			} else {
				log.Println("Received recovered alert at the right time")
			}
		} else {
			t.Errorf("Recovered alert received at %v, want %v", since, alertRecovered)
		}
	case <-time.After(2 * dtEnd * alertWindow):
		t.Errorf("No recovery alerts received, timeout")
	}
	monitorDone <- true
	serverDone <- true
}

func makeFaultyServer(alertWindow time.Duration, dtStart time.Duration, dtEnd time.Duration,
	serverDone chan bool) {
	start := time.Now()
	downtimeStart := dtStart * alertWindow
	downtimeEnd := dtEnd * alertWindow
	log.Printf("Starting server")
	faultyHandler := func(w http.ResponseWriter, req *http.Request) {
		since := time.Since(start)
		if since < downtimeStart || since > downtimeEnd {
			w.WriteHeader(http.StatusOK)
			return
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}
	srv := &http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(faultyHandler),
	}
	// Shutdown for cleaning
	go func() {
		<-serverDone
		log.Printf("Shutting down server")
		err := srv.Shutdown(context.Background())
		if err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
	}()
	srv.ListenAndServe()
}
