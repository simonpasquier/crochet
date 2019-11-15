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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/simonpasquier/crochet/assets"
)

const (
	statusFiring   = "firing"
	statusResolved = "resolved"
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

// notification represents a notification received from Alertmanager.
type notification struct {
	*webhookPayload
	Remote    string    `json:"remoteAddress"`
	Timestamp time.Time `json:"timestamp"`
}

func (n *notification) Key() string {
	return fmt.Sprintf("%s:%s", n.Receiver, n.GroupKey)
}

type incident struct {
	first, last *notification
}

func (i *incident) Key() string {
	return i.first.Key()
}

func (i *incident) Update(n *notification) {
	if i.first == nil {
		i.first = n
	}
	if i.last == nil || n.Timestamp.After(i.last.Timestamp) {
		i.last = n
	}
}

func (i *incident) Duration() time.Duration {
	if i.first == nil || i.last == nil {
		return time.Duration(0)
	}
	return i.last.Timestamp.Sub(i.first.Timestamp)
}

func (i *incident) IsResolved() bool {
	return i.last.Status == statusResolved
}

// store manages Alertmanager notifications.
type store struct {
	notifications []*notification
	incidents     map[string]*incident

	actionc chan func()
	quitc   chan struct{}
}

func newStore() *store {
	return &store{
		notifications: make([]*notification, 0),
		incidents:     make(map[string]*incident),
		actionc:       make(chan func()),
		quitc:         make(chan struct{}),
	}
}

func (s *store) stop() {
	close(s.quitc)
}

func (s *store) run() {
	for {
		select {
		case <-s.quitc:
			return
		case f := <-s.actionc:
			f()
		}
	}
}

func (s *store) addNotification(n *notification) {
	s.actionc <- func() {
		s.notifications = append(s.notifications, n)
	}
}

func (s *store) listNotifications() []*notification {
	var notifications []*notification
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		notifications = s.notifications
	}
	<-done
	return notifications
}

func (s *store) getIncident(n *notification) *incident {
	var i *incident
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		i, _ = s.incidents[n.Key()]
	}
	<-done
	return i
}

func (s *store) updateIncident(n *notification) *incident {
	var i *incident
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		i, _ = s.incidents[n.Key()]
		if i == nil {
			i = &incident{}
			s.incidents[n.Key()] = i
		}
		i.Update(n)
	}
	<-done
	return i
}

func (s *store) deleteIncident(i *incident) {
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		delete(s.incidents, i.Key())
	}
	<-done
	return
}

type notificationAPI struct {
	store                  *store
	incidentsCreatedTotal  prometheus.Counter
	incidentsResolvedTotal prometheus.Counter
	incidentsDuration      prometheus.Histogram
}

func newNotificationAPI(store *store) *notificationAPI {
	var (
		incidentsCreatedTotal = prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "crochet_incidents_total",
				Help: "Total number of incidents",
			},
		)
		incidentsDuration = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "crochet_incidents_duration_seconds",
				Help:    "Duration of incidents",
				Buckets: []float64{10, 60, 120, 300, 600, 1800, 3600, 7200},
			},
		)
	)
	prometheus.MustRegister(
		incidentsCreatedTotal,
		incidentsDuration,
	)
	return &notificationAPI{
		store:                 store,
		incidentsCreatedTotal: incidentsCreatedTotal,
		incidentsDuration:     incidentsDuration,
	}
}

func (a *notificationAPI) post(w http.ResponseWriter, r *http.Request) {
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

	n := &notification{
		Remote:         r.RemoteAddr,
		Timestamp:      time.Now(),
		webhookPayload: &p,
	}
	a.store.addNotification(n)
	if a.store.getIncident(n) == nil {
		// This is a new incident.
		a.incidentsCreatedTotal.Inc()
	}
	i := a.store.updateIncident(n)
	if !i.IsResolved() {
		return
	}
	// Record metrics about incident resolution.
	a.store.deleteIncident(i)
	a.incidentsDuration.Observe(i.Duration().Seconds())
}

func (a *notificationAPI) list(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)

	err := enc.Encode(a.store.listNotifications())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}
}

func (a *notificationAPI) Handle(w http.ResponseWriter, r *http.Request) {
	logger.Printf("Processing %q notification API request from %s", r.Method, r.RemoteAddr)
	switch r.Method {
	case "GET":
		a.list(w, r)
	case "POST":
		a.post(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	flag.Parse()
	if help {
		fmt.Fprintln(os.Stderr, "Simple API service to receive and serve AlertManager webhook payload")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Setup the datastore.
	ds := newStore()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		ds.run()
		wg.Done()
	}()

	// Setup HTTP handlers.
	http.Handle("/", http.FileServer(assets.Assets))
	http.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)

	notifAPI := newNotificationAPI(ds)
	apiDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "crochet_http_requests_duration_seconds",
			Help: "Histogram of HTTP request latencies.",
		},
		[]string{"code", "method", "path"},
	)
	notifDuration := apiDuration.MustCurryWith(prometheus.Labels{"path": "/api/notifications"})
	prometheus.MustRegister(notifDuration)
	http.Handle("/api/notifications/", promhttp.InstrumentHandlerDuration(notifDuration, http.HandlerFunc(notifAPI.Handle)))

	// Start the HTTP server.
	wg.Add(1)
	logger.Println("Listening on", listen)
	srv := &http.Server{Addr: listen}
	shutdown := make(chan struct{})
	go func() {
		defer wg.Done()
		err := srv.ListenAndServe()
		select {
		case <-shutdown:
			return
		default:
			logger.Fatal(err)
		}
	}()

	// Handler termination.
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	<-term
	close(shutdown)

	logger.Println("Received SIGTERM, exiting gracefully...")
	srv.Shutdown(context.Background())
	ds.stop()
	wg.Wait()
}
