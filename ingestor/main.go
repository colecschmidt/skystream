package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const pollInterval = 10 * time.Second

var (
	statesIngested = promauto.NewCounter(prometheus.CounterOpts{
		Name: "aircraft_states_ingested_total",
		Help: "Total number of aircraft state vectors published to NATS.",
	})
	activeCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "aircraft_active_count",
		Help: "Number of aircraft returned by the last successful OpenSky poll.",
	})
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pub, err := NewPublisher(ctx, natsURL)
	if err != nil {
		log.Fatalf("publisher init: %v", err)
	}
	defer pub.Close()

	client := NewClient()

	go serveMetrics(":2112")

	log.Printf("ingestor started — polling every %s, NATS=%s", pollInterval, natsURL)

	poll(ctx, client, pub) // run immediately on startup

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			poll(ctx, client, pub)
		}
	}
}

func poll(ctx context.Context, client *Client, pub *Publisher) {
	const maxRetries = 3

	var (
		vectors []StateVector
		err     error
	)
	for attempt := 1; attempt <= maxRetries; attempt++ {
		vectors, err = client.FetchStates(ctx)
		if err == nil {
			break
		}
		log.Printf("fetch attempt %d/%d failed: %v", attempt, maxRetries, err)
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * 2 * time.Second
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}
	if err != nil {
		log.Printf("all fetch attempts failed, skipping poll: %v", err)
		return
	}

	activeCount.Set(float64(len(vectors)))

	published := 0
	for _, sv := range vectors {
		if err := pub.Publish(ctx, sv); err != nil {
			log.Printf("publish %s: %v", sv.ICAO24, err)
			continue
		}
		statesIngested.Inc()
		published++
	}

	log.Printf("poll complete: %d/%d state vectors published", published, len(vectors))
}

func serveMetrics(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Printf("metrics listening on %s/metrics", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("metrics server: %v", err)
	}
}
