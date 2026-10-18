package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	handler := NewHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s",
			response.Code, http.StatusOK, response.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want %q", body["status"], "ok")
	}
}

func TestRunSimulationStopsWhenRequestIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	payload := `{
		"name": "cancelled-run",
		"serviceCapacity": 1,
		"serviceDuration": 1,
		"arrivals": [
			{"requestId": "req-1", "at": 0}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/simulations",
		strings.NewReader(payload),
	).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	NewHandler().ServeHTTP(response, request)

	if response.Body.Len() != 0 {
		t.Fatalf("response body = %q, want empty body for cancelled request", response.Body.String())
	}
}

func TestRunSimulationEndpointReturnsEngineResults(t *testing.T) {
	handler := NewHandler()
	payload := `{
"name": "first-run",
"serviceCapacity": 1,
"serviceDuration": 5,
"arrivals": [
{"requestId": "req-1", "at": 0},
{"requestId": "req-2", "at": 1}
]
}`

	request := httptest.NewRequest(
		http.MethodPost, "/api/v1/simulations", strings.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s",
			response.Code, http.StatusOK, response.Body.String())
	}

	var body struct {
		Name     string `json:"name"`
		Duration int64  `json:"duration"`
		Summary  struct {
			TotalRequests     int `json:"totalRequests"`
			SucceededRequests int `json:"succeededRequests"`
		} `json:"summary"`
		Metrics struct {
			TotalRequests int `json:"totalRequests"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode simulation response: %v", err)
	}

	if body.Name != "first-run" {
		t.Errorf("name = %q, want first-run", body.Name)
	}
	if body.Duration != 10 {
		t.Errorf("duration = %d, want 10", body.Duration)
	}
	if body.Summary.TotalRequests != 2 {
		t.Errorf("summary totalRequests = %d, want 2", body.Summary.TotalRequests)
	}
	if body.Summary.SucceededRequests != 2 {
		t.Errorf("summary succeededRequests = %d, want 2", body.Summary.SucceededRequests)
	}
	if body.Metrics.TotalRequests != 2 {
		t.Errorf("metrics totalRequests = %d, want 2", body.Metrics.TotalRequests)
	}
}

func TestRunSimulationRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		contentType string
	}{
		{
			name:        "malformed JSON",
			payload:     `{"name":`,
			contentType: "application/json",
		},
		{
			name:        "zero capacity",
			payload:     `{"name":"invalid","serviceCapacity":0,"serviceDuration":1,"arrivals":[{"requestId":"r1","at":0}]}`,
			contentType: "application/json",
		},
		{
			name:        "unknown field",
			payload:     `{"name":"valid","serviceCapacity":1,"serviceDuration":1,"arrivals":[{"requestId":"r1","at":0}],"unexpected":true}`,
			contentType: "application/json",
		},
		{
			name:        "wrong content type",
			payload:     `{"name":"valid"}`,
			contentType: "text/plain",
		},
		{
			name:        "multiple JSON values",
			payload:     `{"name":"first"} {"name":"second"}`,
			contentType: "application/json",
		},
	}

	handler := NewHandler()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost, "/api/v1/simulations", strings.NewReader(test.payload))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s",
					response.Code, http.StatusBadRequest, response.Body.String())
			}

			if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}

			var body struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v; body = %s", err, response.Body.String())
			}
			if strings.TrimSpace(body.Error) == "" {
				t.Error("error response has an empty error message")
			}
		})
	}
}

func TestRunSimulationRejectsOversizedBody(t *testing.T) {
	handler := NewHandler()
	payload := bytes.NewReader(bytes.Repeat([]byte(" "), 1<<20+1))
	request := httptest.NewRequest(
		http.MethodPost, "/api/v1/simulations", payload)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertJSONError(t, response, http.StatusRequestEntityTooLarge)
}

func TestEndpointsRejectUnsupportedMethods(t *testing.T) {
	tests := []struct {
		path   string
		method string
	}{
		{path: "/api/v1/health", method: http.MethodPost},
		{path: "/api/v1/simulations", method: http.MethodGet},
	}

	handler := NewHandler()
	for _, test := range tests {
		t.Run(test.path+"_"+test.method, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assertJSONError(t, response, http.StatusMethodNotAllowed)
		})
	}
}

func assertJSONError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()

	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s",
			response.Code, wantStatus, response.Body.String())
	}

	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v; body = %s", err, response.Body.String())
	}
	if strings.TrimSpace(body.Error) == "" {
		t.Error("error response has an empty error message")
	}
}

func TestUnknownRouteReturnsJSONError(t *testing.T) {
	handler := NewHandler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertJSONError(t, response, http.StatusNotFound)
}

func TestRunSimulationReturnsGatewayTimeout(t *testing.T) {
	handler := &handler{executionTimeout: 0}

	payload := `{
		"name": "timeout-run",
		"serviceCapacity": 1,
		"serviceDuration": 1,
		"arrivals": [
			{"requestId": "req-1", "at": 0}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/simulations",
		strings.NewReader(payload),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assertJSONError(t, response, http.StatusGatewayTimeout)
}

func TestRunSimulationHandlesConcurrentRequests(t *testing.T) {
	handler := NewHandler()

	const (
		concurrency = 12
		arrivals    = 20
	)

	start := make(chan struct{})
	var done sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		i := i
		done.Add(1)

		go func() {
			defer done.Done()
			<-start

			var payload strings.Builder
			fmt.Fprintf(&payload,
				`{"name":"concurrent-%d","serviceCapacity":2,"serviceDuration":3,"arrivals":[`,
				i,
			)

			for j := 0; j < arrivals; j++ {
				if j > 0 {
					payload.WriteByte(',')
				}
				fmt.Fprintf(&payload,
					`{"requestId":"req-%d-%d","at":%d}`,
					i, j, j,
				)
			}
			payload.WriteString(`]}`)

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/simulations",
				strings.NewReader(payload.String()),
			)
			request.Header.Set("Content-Type", "application/json")

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				errors <- fmt.Errorf(
					"request %d: status = %d, want %d; body = %s",
					i, response.Code, http.StatusOK, response.Body.String(),
				)
				return
			}

			var body struct {
				Name    string `json:"name"`
				Summary struct {
					TotalRequests     int `json:"totalRequests"`
					SucceededRequests int `json:"succeededRequests"`
				} `json:"summary"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				errors <- fmt.Errorf("request %d: decode response: %w", i, err)
				return
			}

			if body.Name != fmt.Sprintf("concurrent-%d", i) {
				errors <- fmt.Errorf(
					"request %d: response name = %q",
					i, body.Name,
				)
				return
			}
			if body.Summary.TotalRequests != arrivals ||
				body.Summary.SucceededRequests != arrivals {
				errors <- fmt.Errorf(
					"request %d: total/succeeded = %d/%d, want %d/%d",
					i,
					body.Summary.TotalRequests,
					body.Summary.SucceededRequests,
					arrivals,
					arrivals,
				)
			}
		}()
	}

	close(start)
	done.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// Configured objective thresholds must be accepted and their evaluation
// must be included in the API response.
func TestRunSimulationEndpointEvaluatesObjectives(t *testing.T) {
	payload := `{
                "name": "objective-api-test",
                "serviceCapacity": 1,
                "serviceDuration": 1,
                "arrivals": [
                        {"requestId": "req-1", "at": 0}
                ],
                "objectives": {
                        "minSuccessRate": 0.95,
                        "maxAverageLatency": 10
                }
        }`

	request := httptest.NewRequest(
		http.MethodPost, "/api/v1/simulations", strings.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	NewHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s",
			response.Code, http.StatusOK, response.Body.String())
	}

	var body struct {
		Objectives struct {
			Configured bool `json:"configured"`
			Passed     bool `json:"passed"`
			Outcomes   []struct {
				Name      string  `json:"name"`
				Passed    bool    `json:"passed"`
				Actual    float64 `json:"actual"`
				Threshold float64 `json:"threshold"`
				Message   string  `json:"message"`
			} `json:"outcomes"`
		} `json:"objectives"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode simulation response: %v", err)
	}

	if !body.Objectives.Configured {
		t.Fatal("objectives.configured = false, want true")
	}
	if !body.Objectives.Passed {
		t.Errorf("objectives.passed = false, want true")
	}
	if len(body.Objectives.Outcomes) != 2 {
		t.Fatalf("objective outcomes = %d, want 2", len(body.Objectives.Outcomes))
	}
	if body.Objectives.Outcomes[0].Name != "minimum_success_rate" {
		t.Errorf("first objective name = %q", body.Objectives.Outcomes[0].Name)
	}
	if body.Objectives.Outcomes[0].Threshold != 0.95 {
		t.Errorf("success threshold = %v, want 0.95", body.Objectives.Outcomes[0].Threshold)
	}
	if body.Objectives.Outcomes[1].Name != "maximum_average_latency" {
		t.Errorf("second objective name = %q", body.Objectives.Outcomes[1].Name)
	}
	if body.Objectives.Outcomes[1].Threshold != 10 {
		t.Errorf("latency threshold = %v, want 10", body.Objectives.Outcomes[1].Threshold)
	}
}
func TestRunSimulationEndpointReturnsRouteTraces(t *testing.T) {
	payload := `{
		"name": "route-trace-api",
		"serviceCapacity": 1,
		"serviceDuration": 1,
		"routingAlgorithm": "shortest_path",
		"topology": {
			"nodes": [
				{"id": "client", "type": 1},
				{"id": "relay", "type": 3},
				{"id": "worker-slow", "type": 4},
				{"id": "worker-fast", "type": 4}
			],
			"connections": [
				{"from": "client", "to": "worker-slow", "latency": 8},
				{"from": "client", "to": "relay", "latency": 2},
				{"from": "relay", "to": "worker-fast", "latency": 2}
			]
		},
		"arrivals": [
			{"requestId": "request-1", "sourceNode": "client", "at": 0}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost, "/api/v1/simulations", strings.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	NewHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s",
			response.Code, http.StatusOK, response.Body.String())
	}

	var body struct {
		RouteTraces map[string][]struct {
			From    string `json:"from"`
			To      string `json:"to"`
			At      int64  `json:"at"`
			Latency int64  `json:"latency"`
			Status  string `json:"status"`
		} `json:"routeTraces"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode simulation response: %v", err)
	}

	hops := body.RouteTraces["request-1"]
	if len(hops) != 2 {
		t.Fatalf("route traces = %+v, want 2 hops", hops)
	}

	want := []struct {
		From    string
		To      string
		At      int64
		Latency int64
		Status  string
	}{
		{"client", "relay", 0, 2, "forwarded"},
		{"relay", "worker-fast", 2, 2, "forwarded"},
	}

	for i, expected := range want {
		got := hops[i]
		if got.From != expected.From ||
			got.To != expected.To ||
			got.At != expected.At ||
			got.Latency != expected.Latency ||
			got.Status != expected.Status {
			t.Errorf("hop %d = %+v, want %+v", i, got, expected)
		}
	}
}
