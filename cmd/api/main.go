package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"metrics-app/internal/domain"
	"metrics-app/internal/repository"
	"metrics-app/internal/router"
	"metrics-app/internal/util"
)

const dbPath = "../db/metrics.db"

func LoggerInitialize() (util.MetricsLogger, error) {

	var metricsLogger util.MetricsLogger

	ConstructAndCreateLogFolder()

	if err := metricsLogger.Init("webService.log", false); err != nil {
		fmt.Println("Failed to initialize logger:", err)
		return util.MetricsLogger{}, err
	}

	metricsLogger.LogEvent(util.LOG_LEVEL_INFO, "Service started")

	currentTime := time.Now().Format(time.RFC3339)

	fmt.Fprintf(os.Stderr, "\n%s: MetricsApp Agent started \n", currentTime)

	return metricsLogger, nil

}

func main() {

	logger, err := LoggerInitialize()
	if err != nil {
		fmt.Println("Error while initializing the logger..", err)
		return
	}

	storageType := "sqlite" // Or "inmemory"

	var metricStore domain.MetricStore

	switch storageType {
	case "sqlite":
		metricStore = repository.NewSQLiteStore(dbPath)
	default:
		log.Fatalf("Unknown storage type: %s", storageType)
	}

	if err := metricStore.Init(); err != nil {
		log.Fatalf("Failed to initialize metric store: %v", err)
	}
	defer metricStore.Close()

	router.Run(metricStore, &logger)
}

func ConstructAndCreateLogFolder() {
	logPath := ".." + string(os.PathSeparator) + "log"
	util.SetLoggerPath(logPath)
	util.CheckAndCreateLogFolder(logPath)
	util.SetCommonLoggerAttributes(3)
}
