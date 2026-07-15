// Package metrics exposes crashdummy's Prometheus instrumentation: per-route
// request counts and latencies, plus the default Go and process collectors.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const metricsPath = "/metrics"

var (
	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "crashdummy_requests_total",
		Help: "Total HTTP requests handled, labeled by matched route, method, and response status.",
	}, []string{"route", "method", "status"})

	duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "crashdummy_request_duration_seconds",
		Help:    "HTTP request duration in seconds, labeled by matched route and method.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route", "method"})
)

// Handler serves the Prometheus metrics endpoint, including the default Go and
// process collectors registered on the default registry.
func Handler() http.Handler {
	return promhttp.Handler()
}

// Instrument wraps next, recording request count and latency for every request
// except the metrics endpoint itself. The route label is the matched ServeMux
// pattern, which keeps label cardinality bounded to the registered routes.
func Instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == metricsPath {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &recorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		requests.WithLabelValues(route, r.Method, strconv.Itoa(rec.status)).Inc()
		duration.WithLabelValues(route, r.Method).Observe(time.Since(start).Seconds())
	})
}

// recorder captures the response status code for instrumentation. It defaults
// to 200, which a handler that writes a body without calling WriteHeader
// implicitly returns.
type recorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *recorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}
