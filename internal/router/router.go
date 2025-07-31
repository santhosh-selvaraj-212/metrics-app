package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"metrics-app/internal/domain"
	"metrics-app/internal/endpoints"
	"metrics-app/internal/util"
)

func NewRouter(metricStore domain.MetricStore, webSlogger *util.MetricsLogger) *mux.Router {
	r := mux.NewRouter()

	addRoutes(r, metricStore, webSlogger)

	r.Use(loggingMiddleware(webSlogger))

	return r
}

func addRoutes(r *mux.Router, metricStore domain.MetricStore, webSlogger *util.MetricsLogger) {

	metricsHandler := &endpoints.Metrics{}
	metricsHandler.Init(metricStore, webSlogger)

	r.HandleFunc("/metrics/{limit}/{offset}", metricsHandler.GetMetricsHandler).Methods("GET")
}

func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func Run(metricStore domain.MetricStore, webSlogger *util.MetricsLogger) {
	appRouter := NewRouter(metricStore, webSlogger)

	server := NewServer(":8080", appRouter)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		println()
		log.Println("Shutting down server...")

		err := gracefulShutdown(server, 25*time.Second)

		if err != nil {
			log.Printf("Server stopped with error: %s", err.Error())
		} else {
			log.Println("Server stopped gracefully.")
		}

		os.Exit(0)
	}()

	log.Printf("Listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func gracefulShutdown(server *http.Server, maximumTime time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), maximumTime)
	defer cancel()

	return server.Shutdown(ctx)
}

func loggingMiddleware(logger *util.MetricsLogger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.LogEvent(util.LOG_LEVEL_INFO, fmt.Sprintf("Request: %s %s", r.Method, r.RequestURI))
			next.ServeHTTP(w, r)
		})
	}
}
