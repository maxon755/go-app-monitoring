package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Number of HTTP requests",
	},
	[]string{"path"},
)

var responseStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "response_status_total",
		Help: "Status of HTTP response",
	},
	[]string{"status"},
)

var httpDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "http_response_time_seconds",
		Help: "Duration of HTTP requests.",
	},
	[]string{"path"},
)

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))

		rw := NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		responseStatus.WithLabelValues(strconv.Itoa(rw.statusCode)).Inc()
		totalRequests.WithLabelValues(path).Inc()
		timer.ObserveDuration()
	})
}

func init() {
	prometheus.MustRegister(totalRequests)
	prometheus.MustRegister(responseStatus)
	prometheus.MustRegister(httpDuration)
}

func main() {
	router := mux.NewRouter()
	router.Use(prometheusMiddleware)

	router.Path("/metrics").Handler(promhttp.Handler())

	// Serving static files
	router.Path("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		statuses := [5]int{200, 201, 400, 422, 500}
		delay := rand.Intn(1000)

		time.Sleep(time.Duration(delay) * time.Millisecond)

		w.WriteHeader(statuses[rand.Intn(len(statuses))])
		w.Write([]byte(fmt.Sprintf("Hello http after %d milliseconds", delay)))
	}))

	// Prometheus endpoint

	fmt.Println("Serving requests on port 8080")
	err := http.ListenAndServe(":8080", router)
	log.Fatal(err)
}
