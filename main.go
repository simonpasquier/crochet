package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/simonpasquier/crochet/assets"
)

var (
	help   bool
	listen string
	rnd    = rand.New(rand.NewSource(time.Now().UnixNano()))
	logger = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds)
)

func init() {
	flag.BoolVar(&help, "help", false, "Help message")
	flag.StringVar(&listen, "listen-address", ":8080", "Listen address")
}

// webhookPayload represents the raw data received from Alertmanager.
type webhookPayload struct {
	*template.Data
	Version  string `json:"version"`
	GroupKey string `json:"groupKey"`
}

// request represents a notification received from Alertmanager.
type request struct {
	*webhookPayload
	Remote    string    `json:"remoteAddress"`
	Timestamp time.Time `json:"timestamp"`
}

// store manages Alertmanager notifications.
type store struct {
	requests []*request

	actionc chan func()
	quitc   chan struct{}
}

func (s *store) stop() {
	s.quitc <- struct{}{}
}

func (s *store) loop() {
	for {
		select {
		case <-s.quitc:
			return
		case f := <-s.actionc:
			f()
		}
	}
}

func (s *store) add(r *request) {
	s.actionc <- func() {
		s.requests = append(s.requests, r)
	}
}

func (s *store) list() []*request {
	var reqs []*request
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		reqs = s.requests
	}
	<-done
	return reqs
}

func (s *store) postHandler(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Content-Type") != "application/json" {
		logger.Printf("Invalid Content-Type: %q", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var p webhookPayload
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		logger.Println("Failed to decode payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.add(&request{
		Remote:         r.RemoteAddr,
		Timestamp:      time.Now(),
		webhookPayload: &p,
	})
}

func (s *store) getHandler(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)

	err := enc.Encode(s.list())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}
}

func main() {
	flag.Parse()
	if help {
		fmt.Fprintln(os.Stderr, "Simple API service to receive and serve AlertManager webhook payload")
		flag.PrintDefaults()
		os.Exit(0)
	}

	s := store{
		requests: make([]*request, 0),
		actionc:  make(chan func()),
		quitc:    make(chan struct{}),
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		s.loop()
		wg.Done()
	}()

	http.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	http.HandleFunc("/api/notifications/", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("Processing %q request from %s", r.Method, r.RemoteAddr)
		switch r.Method {
		case "GET":
			s.getHandler(w, r)
		case "POST":
			s.postHandler(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.Handle("/", http.FileServer(assets.Assets))

	wg.Add(1)
	logger.Println("Listening on", listen)
	srv := &http.Server{Addr: listen}
	go func() {
		defer wg.Done()
		err := srv.ListenAndServe()
		select {
		case <-s.quitc:
			return
		default:
			logger.Fatal(err)
		}
	}()

	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	<-term

	logger.Println("Received SIGTERM, exiting gracefully...")
	close(s.quitc)
	srv.Shutdown(context.Background())
	wg.Wait()
}
