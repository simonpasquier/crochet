package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/client_golang/prometheus"
)

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
	if a.store.getIncident(n.Key()) == nil {
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
	w.Header().Set("Content-Type", "application/json")
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

type incidentAPI struct {
	store *store
}

func newIncidentAPI(store *store) *incidentAPI {
	return &incidentAPI{
		store: store,
	}
}
func (a *incidentAPI) Handle(w http.ResponseWriter, r *http.Request) {
	logger.Printf("Processing %q incident API request from %s", r.Method, r.RemoteAddr)
	switch r.Method {
	case "GET":
		a.list(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *incidentAPI) list(w http.ResponseWriter, r *http.Request) {
	type incident struct {
		Duration float64 `json:"duration"`
		Key      string  `json:"key"`
		Alerts   template.Alerts
	}
	var incidents []incident

	for _, i := range a.store.listIncidents() {
		incidents = append(incidents, incident{
			Duration: i.Duration().Seconds(),
			Key:      i.Key(),
			Alerts:   i.Alerts(),
		})
	}

	enc := json.NewEncoder(w)
	err := enc.Encode(incidents)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
}
