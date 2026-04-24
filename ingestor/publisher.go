package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	streamName    = "AIRCRAFT_STATES"
	subjectPrefix = "aircraft.states"
)

type Publisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func NewPublisher(ctx context.Context, url string) (*Publisher, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream context: %w", err)
	}

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subjectPrefix + ".>"},
		MaxAge:   24 * time.Hour,
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create stream %s: %w", streamName, err)
	}

	return &Publisher{nc: nc, js: js}, nil
}

func (p *Publisher) Publish(ctx context.Context, sv StateVector) error {
	data, err := json.Marshal(sv)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	subject := subjectPrefix + "." + sv.ICAO24
	_, err = p.js.Publish(ctx, subject, data)
	return err
}

func (p *Publisher) Close() {
	p.nc.Drain() // flush pending messages before closing
}
