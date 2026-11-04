package servicekit

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var requestDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type requestMetricKey struct {
	service string
	method  string
	code    string
}

type durationMetric struct {
	buckets []uint64
	count   uint64
	sum     float64
}

type durationMetricKey struct {
	service string
	method  string
}

type grpcMetrics struct {
	mu        sync.Mutex
	requests  map[requestMetricKey]uint64
	durations map[durationMetricKey]*durationMetric
}

// NewPrometheusMetrics returns a gRPC interceptor and a handler exposing Prometheus text metrics.
func NewPrometheusMetrics(service string) (grpc.UnaryServerInterceptor, http.Handler) {
	metrics := &grpcMetrics{
		requests:  make(map[requestMetricKey]uint64),
		durations: make(map[durationMetricKey]*durationMetric),
	}

	interceptor := func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		response, err := handler(ctx, request)
		metrics.record(service, info.FullMethod, status.Code(err).String(), time.Since(start).Seconds())
		return response, err
	}
	return interceptor, metrics
}

func (m *grpcMetrics) record(service, method, code string, seconds float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[requestMetricKey{service: service, method: method, code: code}]++
	key := durationMetricKey{service: service, method: method}
	histogram := m.durations[key]
	if histogram == nil {
		histogram = &durationMetric{buckets: make([]uint64, len(requestDurationBuckets))}
		m.durations[key] = histogram
	}
	histogram.count++
	histogram.sum += seconds
	for index, bound := range requestDurationBuckets {
		if seconds <= bound {
			histogram.buckets[index]++
		}
	}
}

func (m *grpcMetrics) ServeHTTP(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	m.mu.Lock()
	requests := make([]struct {
		key   requestMetricKey
		count uint64
	}, 0, len(m.requests))
	for key, count := range m.requests {
		requests = append(requests, struct {
			key   requestMetricKey
			count uint64
		}{key: key, count: count})
	}
	durations := make([]struct {
		key     durationMetricKey
		buckets []uint64
		count   uint64
		sum     float64
	}, 0, len(m.durations))
	for key, histogram := range m.durations {
		durations = append(durations, struct {
			key     durationMetricKey
			buckets []uint64
			count   uint64
			sum     float64
		}{key: key, buckets: append([]uint64(nil), histogram.buckets...), count: histogram.count, sum: histogram.sum})
	}
	m.mu.Unlock()

	sort.Slice(requests, func(i, j int) bool {
		left, right := requests[i].key, requests[j].key
		if left.service != right.service {
			return left.service < right.service
		}
		if left.method != right.method {
			return left.method < right.method
		}
		return left.code < right.code
	})
	fmt.Fprintln(response, "# HELP drukleague_grpc_server_requests_total Total number of gRPC requests handled by the service.")
	fmt.Fprintln(response, "# TYPE drukleague_grpc_server_requests_total counter")
	for _, sample := range requests {
		fmt.Fprintf(response, "drukleague_grpc_server_requests_total{service=%q,method=%q,code=%q} %d\n", sample.key.service, sample.key.method, sample.key.code, sample.count)
	}

	sort.Slice(durations, func(i, j int) bool {
		left, right := durations[i].key, durations[j].key
		if left.service != right.service {
			return left.service < right.service
		}
		return left.method < right.method
	})
	fmt.Fprintln(response, "# HELP drukleague_grpc_server_request_duration_seconds Time spent handling gRPC requests, in seconds.")
	fmt.Fprintln(response, "# TYPE drukleague_grpc_server_request_duration_seconds histogram")
	for _, sample := range durations {
		labels := fmt.Sprintf("service=%q,method=%q", sample.key.service, sample.key.method)
		for index, bound := range requestDurationBuckets {
			fmt.Fprintf(response, "drukleague_grpc_server_request_duration_seconds_bucket{%s,le=%q} %d\n", labels, formatBucket(bound), sample.buckets[index])
		}
		fmt.Fprintf(response, "drukleague_grpc_server_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", labels, sample.count)
		fmt.Fprintf(response, "drukleague_grpc_server_request_duration_seconds_sum{%s} %g\n", labels, sample.sum)
		fmt.Fprintf(response, "drukleague_grpc_server_request_duration_seconds_count{%s} %d\n", labels, sample.count)
	}
}

func formatBucket(bound float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", bound), "0"), ".")
}
