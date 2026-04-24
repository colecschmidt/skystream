#!/bin/bash
docker run --rm --network skystream_default natsio/nats-box nats sub --server nats://nats:4222 "aircraft.states.>"
