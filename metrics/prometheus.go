package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	totalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "code"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	totalErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP responses with 5xx codes",
		},
		[]string{"method", "endpoint", "code"},
	)
)

func init() {
	prometheus.MustRegister(totalRequests)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(totalErrors)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			endpoint := routeTemplate(r)

			codeStr := strconv.Itoa(sw.status)
			totalRequests.WithLabelValues(r.Method, endpoint, codeStr).Inc()
			requestDuration.WithLabelValues(r.Method, endpoint).Observe(time.Since(start).Seconds())

			if sw.status >= 500 {
				totalErrors.WithLabelValues(r.Method, endpoint, codeStr).Inc()
			}
		})
	}
}

func routeTemplate(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return r.URL.Path
	}
	tpl, err := route.GetPathTemplate()
	if err != nil || tpl == "" {
		return r.URL.Path
	}
	return tpl
}

