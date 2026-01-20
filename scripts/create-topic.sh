#!/bin/bash
echo "Waiting for Kafka to be ready..."
while ! nc -z kafka 9092; do
  sleep 0.1
done
echo "Kafka is ready!"
echo "Creating Kafka topics..."
kafka-topics --create --topic rep_tracker_changes --bootstrap-server kafka:9092 --partitions 1 --replication-factor 1 --if-not-exists
echo "Kafka topics created successfully!"
