package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"metrics-app/internal/domain"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db     *sql.DB
	dbPath string
}

func NewSQLiteStore(path string) *SQLiteStore {
	return &SQLiteStore{dbPath: path}
}

func (s *SQLiteStore) Init() error {
	var err error

	s.db, err = sql.Open("sqlite3", s.dbPath)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	if err = s.db.Ping(); err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS metrics (
		timestamp INTEGER PRIMARY KEY,
		cpu_load REAL,
		concurrency INTEGER
	);`

	_, err = s.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}

	log.Println("SQLiteStore initialized.")
	return nil
}

func (s *SQLiteStore) StoreMetric(ctx context.Context, metric domain.Metric) error {
	stmt, err := s.db.PrepareContext(ctx, "INSERT INTO metrics(timestamp, cpu_load, concurrency) VALUES(?, ?, ?)")
	if err != nil {
		return fmt.Errorf("error preparing insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, metric.Timestamp, metric.CPULoad, metric.Concurrency)
	if err != nil {
		return fmt.Errorf("error inserting metric: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetMetrics(ctx context.Context, startTime, endTime int64, limit, offset int) ([]domain.Metric, error) {
	query := "SELECT timestamp, cpu_load, concurrency FROM metrics WHERE timestamp >= ? AND timestamp <= ? ORDER BY timestamp ASC"
	args := []interface{}{startTime, endTime}

	if limit <= 0 {
		limit = -1
	}
	query += " LIMIT ?"
	args = append(args, limit)

	if offset < 0 {
		offset = 0
	}
	query += " OFFSET ?"
	args = append(args, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying database: %w", err)
	}
	defer rows.Close()

	var fetchedMetrics []domain.Metric

	for rows.Next() {
		var m domain.Metric

		if err := rows.Scan(&m.Timestamp, &m.CPULoad, &m.Concurrency); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fetchedMetrics = append(fetchedMetrics, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}
	return fetchedMetrics, nil
}

func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
