# skystream

Real-time aircraft tracking pipeline. Polls the [OpenSky Network](https://opensky-network.org/) API every 10 seconds and publishes state vectors to NATS JetStream for downstream consumers.

## Architecture

```
OpenSky API → ingestor → NATS JetStream (AIRCRAFT_STATES) → consumers
```

## Services

| Service | Description |
|---------|-------------|
| `ingestor` | Polls OpenSky, publishes to NATS |
| `nats` | JetStream message broker |

## Quick start

```bash
docker compose up
```

Metrics available at `http://localhost:2112/metrics`.

## Ingestor

Polls `https://opensky-network.org/api/states/all` every 10 seconds and publishes each aircraft as JSON to subject `aircraft.states.<icao24>` on the `AIRCRAFT_STATES` stream.

### State vector schema

```json
{
  "icao24":        "a12bc3",
  "callsign":      "UAL123",
  "latitude":      37.6213,
  "longitude":     -122.379,
  "baro_altitude": 10972.8,
  "velocity":      245.1,
  "squawk":        "1200",
  "time_position": 1714000000
}
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |

### Prometheus metrics

| Metric | Type | Description |
|--------|------|-------------|
| `aircraft_states_ingested_total` | Counter | State vectors successfully published |
| `aircraft_active_count` | Gauge | Aircraft returned in last poll |

## Running locally

```bash
# Start NATS with JetStream
docker compose up nats

# Run the ingestor
cd ingestor
NATS_URL=nats://localhost:4222 go run .
```
