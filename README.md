#Metrics App API

#Overview

The metrics-app is a Go-based service designed to collect, store, and retrieve system metrics such as CPU load and concurrency. It provides a simple API that can be integrated with monitoring agents to record data over time and retrieve it for analysis. The application supports two storage backends: an in-memory store for quick prototyping and a persistent SQLite database for production use.

#Data Model

The primary data entity in this application is the Metric struct, which represents a single data point recorded at a specific moment in time.

-Timestamp: A Unix timestamp (in seconds) indicating when the reading was taken.

-CPULoad: A floating-point value representing the CPU load at the time of the reading.

-Concurrency: An integer value representing the number of concurrent processes or requests.

#Store Metric

#Ingest Process

The metrics-app is designed to ingest and store metrics data. The ingest process automatically inserts the last 5 minutes of readings, ensuring a consistent and recent data set is maintained

timestamp: An integer Unix timestamp, e.g., 1722441990.

cpu_load: A float value, e.g., 45.75.

concurrency: An integer value, e.g., 100.

#Get Stored Metrics

This endpoint retrieves a list of metrics within a specified time range, with support for pagination.

#Endpoint: GET /metrics/{limit}/{offset}

#Path Parameters:

limit: An integer value to limit the number of metrics returned, e.g., 50.

offset: An integer value to specify the starting point for the results, e.g., 0.

#Input (Request Body):

{
    "start": <startTime>,
    "end": <endTime>
}

start: The starting Unix timestamp of the desired range.

end: The ending Unix timestamp of the desired range.

#Output:

{
    "status": true,
    "value": [
        { "timestamp": 1722441990, "cpu_load": 45.75, "concurrency": 100 },
        { "timestamp": 1722441991, "cpu_load": 46.10, "concurrency": 102 },
        ...
    ],
    "error_code": 303000
}

#Development

The project is built using Go modules. The following make commands are available to assist with development and testing.

make run: Builds and runs the application.

make test: Executes all unit and integration tests with race detection and coverage reporting.

make build: Compiles the application into a binary.

make clean: Removes the build artifacts.

make all: Runs clean, test, build and then run in sequence.

make: The default command is make all.

You can also run tests directly using the following command:

go test -race -cover -coverprofile=coverage.txt -covermode=atomic ./...