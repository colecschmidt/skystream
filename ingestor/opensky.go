package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const openskyURL = "https://opensky-network.org/api/states/all"

// StateVector holds the fields we care about from one OpenSky state entry.
// The API returns states as positional JSON arrays; indices are documented at
// https://openskynetwork.github.io/opensky-api/rest.html
type StateVector struct {
	ICAO24       string  `json:"icao24"`
	Callsign     string  `json:"callsign"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	BaroAltitude float64 `json:"baro_altitude"`
	Velocity     float64 `json:"velocity"`
	Squawk       string  `json:"squawk"`
	TimePosition int64   `json:"time_position"`
}

type openskyResponse struct {
	Time   int64             `json:"time"`
	States [][]any           `json:"states"`
}

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient() *Client {
	return &Client{
		http:    &http.Client{Timeout: 15 * time.Second},
		baseURL: openskyURL,
	}
}

func (c *Client) FetchStates(ctx context.Context) ([]StateVector, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var raw openskyResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	vectors := make([]StateVector, 0, len(raw.States))
	for _, s := range raw.States {
		if sv := parseStateVector(s); sv != nil {
			vectors = append(vectors, *sv)
		}
	}
	return vectors, nil
}

// parseStateVector maps a positional state array to a StateVector.
// Fields that are null in the API response are left as zero values.
func parseStateVector(s []any) *StateVector {
	if len(s) < 15 {
		return nil
	}
	sv := &StateVector{}

	if v, ok := s[0].(string); ok {
		sv.ICAO24 = v
	}
	if sv.ICAO24 == "" {
		return nil // ICAO24 is the primary key; skip entries without it
	}
	if v, ok := s[1].(string); ok {
		sv.Callsign = strings.TrimSpace(v)
	}
	if v, ok := s[3].(float64); ok { // time_position
		sv.TimePosition = int64(v)
	}
	if v, ok := s[6].(float64); ok { // latitude
		sv.Latitude = v
	}
	if v, ok := s[5].(float64); ok { // longitude
		sv.Longitude = v
	}
	if v, ok := s[7].(float64); ok { // baro_altitude
		sv.BaroAltitude = v
	}
	if v, ok := s[9].(float64); ok { // velocity
		sv.Velocity = v
	}
	if v, ok := s[14].(string); ok { // squawk
		sv.Squawk = v
	}
	return sv
}
