package middleware

import (
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed.",
		},
	)
)

// uuidOrNumericSegment matches path segments that are UUIDs or numeric IDs.
var uuidOrNumericSegment = regexp.MustCompile(
	`/([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[0-9]+)(?:/|$)`,
)

// normalizePath replaces UUID and numeric path segments with :id to avoid
// high-cardinality labels in Prometheus metrics.
func normalizePath(path string) string {
	return uuidOrNumericSegment.ReplaceAllStringFunc(path, func(match string) string {
		if match[len(match)-1] == '/' {
			return "/:id/"
		}
		return "/:id"
	})
}

// Metrics returns a Gin middleware that records Prometheus HTTP metrics.
// It must be registered before auth middleware so unauthenticated requests
// are also counted.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpRequestsInFlight.Inc()
		start := time.Now()

		c.Next()

		httpRequestsInFlight.Dec()

		status := strconv.Itoa(c.Writer.Status())
		path := normalizePath(c.Request.URL.Path)
		method := c.Request.Method

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path, status).Observe(time.Since(start).Seconds())
	}
}
