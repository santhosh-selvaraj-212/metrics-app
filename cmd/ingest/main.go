package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"metrics-app/internal/domain"
	"metrics-app/internal/repository"
	"metrics-app/internal/util"
)

const dbPath = "../db/metrics.db"

func main() {

	util.CheckAndCreateLogFolder("../db")

	sqliteStore := repository.NewSQLiteStore(dbPath)
	if err := sqliteStore.Init(); err != nil {
		log.Fatalf("Failed to initialize SQLite store for ingestion: %v", err)
	}
	defer sqliteStore.Close()

	generateAndIngest(sqliteStore)
}

func generateAndIngest(s domain.MetricStore) {
	rand.Seed(time.Now().UnixNano())

	endTime := time.Now()
	startTime := endTime.Add(-5 * time.Minute)

	log.Printf("Ingesting data from %s to %s (past 5 minutes)...", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	ctx := context.Background()

	for t := startTime; t.Before(endTime) || t.Equal(endTime); t = t.Add(10 * time.Second) {

		cpuLoad := rand.Float64() * 100.0

		concurrency := rand.Intn(500001)

		timestamp := t.Unix()

		metric := domain.Metric{
			Timestamp:   timestamp,
			CPULoad:     cpuLoad,
			Concurrency: concurrency,
		}

		err := s.StoreMetric(ctx, metric)
		if err != nil {
			log.Printf("Error inserting data for timestamp %d: %v", timestamp, err)
			continue
		}
	}

	log.Println("Data ingestion complete.")
}
