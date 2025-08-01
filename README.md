# 📊 Metrics App API

## 🚀 Overview
The **Metrics App** is a Go-based service for collecting, storing, and retrieving system metrics such as CPU load and concurrency. It provides a clean API suitable for integration with monitoring agents.

This application uses a **persistent SQLite database** as its storage backend—ideal for production environments and long-term analysis.

---

## 🧬 Data Model

The primary data structure is `Metric`, which represents a snapshot in time:

| Field        | Type      | Description                                     |
|--------------|-----------|-------------------------------------------------|
| `timestamp`  | `int64`   | Unix timestamp (in seconds)                     |
| `cpu_load`   | `float64` | CPU load at the time of recording               |
| `concurrency`| `int`     | Number of concurrent processes or requests      |

---

## 📥 Store Metric

### 🔁 Ingest Process
The app automatically stores the last 5 minutes of metric readings to ensure fresh and consistent data is maintained.

```json
{
  "timestamp": 1722441990,
  "cpu_load": 45.75,
  "concurrency": 100
}
```

---

## 📤 Retrieve Stored Metrics

### 🧭 Endpoint
```
GET /metrics/{limit}/{offset}
```

### 🔧 Parameters
- `limit`: Max number of metrics to return (e.g., `50`)
- `offset`: Starting point for retrieval (e.g., `0`)

### 📨 Request Body
```json
{
  "start": 1722441990,
  "end": 1722442290
}
```

### 📦 Response Example
```json
{
  "status": true,
  "value": [
    { "timestamp": 1722441990, "cpu_load": 45.75, "concurrency": 100 },
    { "timestamp": 1722441991, "cpu_load": 46.10, "concurrency": 102 }
  ],
  "error_code": 303000
}
```

---

## 🛠️ Development & Testing

This project uses **Go modules**. The following `make` commands streamline development and testing:

| Command       | Description                                           |
|---------------|-------------------------------------------------------|
| `make run`    | Builds and runs the application                      |
| `make test`   | Executes unit & integration tests with coverage      |
| `make build`  | Compiles the app into a binary                       |
| `make clean`  | Removes build artifacts                              |
| `make all`    | Runs `clean`, `test`, `build`, then `run` in order   |
| `make`        | Default command, equivalent to `make all`            |

### ✅ Run Tests Manually
```bash
go test -race -cover -coverprofile=coverage.txt -covermode=atomic ./...
```
