package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/simonpasquier/crochet/assets"
)

var (
	help   bool
	listen string
	logger = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds)
)

func init() {
	flag.BoolVar(&help, "help", false, "Help message")
	flag.StringVar(&listen, "listen-address", ":8080", "Listen address")
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

	apiDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "crochet_http_requests_duration_seconds",
			Help: "Histogram of HTTP request latencies.",
		},
		[]string{"code", "method", "path"},
	)
	prometheus.MustRegister(apiDuration)

	// Setup HTTP handlers.
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			http.Error(w, fmt.Sprintf("method %q not allowed", req.Method), http.StatusMethodNotAllowed)
			return
		}
		http.FileServer(assets.Assets).ServeHTTP(w, req)
	})

	http.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)

	notifAPI := newNotificationAPI(ds)
	notifDuration := apiDuration.MustCurryWith(prometheus.Labels{"path": "/api/notifications"})
	http.Handle("/api/notifications/", promhttp.InstrumentHandlerDuration(notifDuration, http.HandlerFunc(notifAPI.Handle)))

	incidentAPI := newIncidentAPI(ds)
	incidentDuration := apiDuration.MustCurryWith(prometheus.Labels{"path": "/api/incidents"})
	http.Handle("/api/incidents/", promhttp.InstrumentHandlerDuration(incidentDuration, http.HandlerFunc(incidentAPI.Handle)))

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
