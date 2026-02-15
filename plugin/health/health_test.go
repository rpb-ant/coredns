package health

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	h := &health{Addr: ":0"}

	if err := h.OnStartup(); err != nil {
		t.Fatalf("Unable to startup the health server: %v", err)
	}
	defer h.OnFinalShutdown()

	address := fmt.Sprintf("http://%s%s", h.ln.Addr().String(), "/health")

	response, err := http.Get(address)
	if err != nil {
		t.Fatalf("Unable to query %s: %v", address, err)
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("Invalid status code: expecting '200', got '%d'", response.StatusCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Unable to get response body from %s: %v", address, err)
	}
	response.Body.Close()

	if string(content) != http.StatusText(http.StatusOK) {
		t.Errorf("Invalid response body: expecting 'OK', got '%s'", string(content))
	}
}

func TestHealthLameduck(t *testing.T) {
	h := &health{Addr: ":0", lameduck: 250 * time.Millisecond}

	if err := h.OnStartup(); err != nil {
		t.Fatalf("Unable to startup the health server: %v", err)
	}

	h.OnFinalShutdown()
}

// testChecker is a HealthChecker implementation for testing.
type testChecker struct{ healthy bool }

func (tc *testChecker) Healthy() bool { return tc.healthy }

func TestHealthCheckers(t *testing.T) {
	tests := []struct {
		name       string
		checkers   []namedChecker
		wantStatus int
		wantBody   string
	}{
		{
			name:       "no checkers returns 200",
			wantStatus: http.StatusOK,
			wantBody:   "OK",
		},
		{
			name: "all healthy returns 200",
			checkers: []namedChecker{
				{name: "foo", checker: &testChecker{true}},
				{name: "bar", checker: &testChecker{true}},
			},
			wantStatus: http.StatusOK,
			wantBody:   "OK",
		},
		{
			name: "one unhealthy returns 503 with name",
			checkers: []namedChecker{
				{name: "foo", checker: &testChecker{true}},
				{name: "bar", checker: &testChecker{false}},
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "bar",
		},
		{
			name: "all unhealthy returns 503 with names",
			checkers: []namedChecker{
				{name: "foo", checker: &testChecker{false}},
				{name: "bar", checker: &testChecker{false}},
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "foo,bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &health{Addr: ":0", checkers: tt.checkers}
			if err := h.OnStartup(); err != nil {
				t.Fatalf("OnStartup: %v", err)
			}
			defer h.OnFinalShutdown()

			resp, err := http.Get(fmt.Sprintf("http://%s/health", h.ln.Addr()))
			if err != nil {
				t.Fatalf("GET /health: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", string(body), tt.wantBody)
			}
		})
	}
}
