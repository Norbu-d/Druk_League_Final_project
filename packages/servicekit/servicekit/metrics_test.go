package servicekit

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPrometheusMetricsInterceptorRecordsRequests(t *testing.T) {
	interceptor, handler := NewPrometheusMetrics("team-service")
	method := "/drukleague.team.v1.TeamService/GetTeam"

	for _, resultErr := range []error{nil, status.Error(codes.NotFound, "missing team")} {
		_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: method}, func(context.Context, any) (any, error) {
			return nil, resultErr
		})
		if err != resultErr {
			t.Fatalf("interceptor error = %v, want %v", err, resultErr)
		}
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	if got := response.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}

	body := response.Body.String()
	for _, want := range []string{
		`drukleague_grpc_server_requests_total{service="team-service",method="/drukleague.team.v1.TeamService/GetTeam",code="OK"} 1`,
		`drukleague_grpc_server_requests_total{service="team-service",method="/drukleague.team.v1.TeamService/GetTeam",code="NotFound"} 1`,
		`drukleague_grpc_server_request_duration_seconds_bucket{service="team-service",method="/drukleague.team.v1.TeamService/GetTeam",le="+Inf"} 2`,
		`drukleague_grpc_server_request_duration_seconds_count{service="team-service",method="/drukleague.team.v1.TeamService/GetTeam"} 2`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}

func TestPrometheusMetricsFormatsEmptyAndSortedSamples(t *testing.T) {
	empty := &grpcMetrics{requests: make(map[requestMetricKey]uint64), durations: make(map[durationMetricKey]*durationMetric)}
	emptyResponse := httptest.NewRecorder()
	empty.ServeHTTP(emptyResponse, httptest.NewRequest("GET", "/metrics", nil))
	if got := emptyResponse.Body.String(); !strings.Contains(got, "# TYPE drukleague_grpc_server_requests_total counter\n") || strings.Contains(got, "drukleague_grpc_server_requests_total{") {
		t.Fatalf("unexpected empty metrics output: %s", got)
	}

	metrics := &grpcMetrics{requests: make(map[requestMetricKey]uint64), durations: make(map[durationMetricKey]*durationMetric)}
	metrics.record("zeta", "/z", "OK", 11)
	metrics.record("alpha", "/b", "Unavailable", 0.005)
	metrics.record("alpha", "/a", "OK", 0.01)
	metrics.record("alpha", "/a", "NotFound", 0.025)

	response := httptest.NewRecorder()
	metrics.ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	body := response.Body.String()

	first := strings.Index(body, `service="alpha",method="/a",code="NotFound"`)
	second := strings.Index(body, `service="alpha",method="/a",code="OK"`)
	third := strings.Index(body, `service="alpha",method="/b",code="Unavailable"`)
	fourth := strings.Index(body, `service="zeta",method="/z",code="OK"`)
	if first < 0 || second <= first || third <= second || fourth <= third {
		t.Fatalf("request samples are not sorted: %s", body)
	}
	for _, want := range []string{
		`drukleague_grpc_server_request_duration_seconds_bucket{service="zeta",method="/z",le="10"} 0`,
		`drukleague_grpc_server_request_duration_seconds_bucket{service="zeta",method="/z",le="+Inf"} 1`,
		`drukleague_grpc_server_request_duration_seconds_sum{service="zeta",method="/z"} 11`,
		`drukleague_grpc_server_request_duration_seconds_count{service="alpha",method="/a"} 2`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}
