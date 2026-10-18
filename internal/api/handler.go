package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"example.com/netarena/internal/sim"
)

const maxRequestBodyBytes = 1 << 20
const simulationExecutionTimeout = 25 * time.Second

// NewHandler creates the HTTP handler for the versioned NetArena API.
func NewHandler() http.Handler {
	return &handler{
		executionTimeout: simulationExecutionTimeout,
	}
}

type handler struct {
	executionTimeout time.Duration
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/health":
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ok"})

	case "/api/v1/simulations":
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.runSimulation(w, r)

	default:
		writeError(w, http.StatusNotFound, "route not found")
	}
}

func (h *handler) runSimulation(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		writeError(w, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var scenario sim.Scenario
	if err := decoder.Decode(&scenario); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds 1 MiB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid simulation request: "+err.Error())
		return
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			writeError(w, http.StatusBadRequest, "request must contain exactly one JSON value")
			return
		}
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds 1 MiB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid trailing JSON data")
		return
	}

	if err := r.Context().Err(); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.executionTimeout)
	defer cancel()

	result, err := sim.RunScenarioContext(ctx, scenario)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		if errors.Is(err, context.DeadlineExceeded) {
			writeError(w, http.StatusGatewayTimeout, "simulation execution timed out")
			return
		}

		if errors.Is(err, context.Canceled) {
			return
		}

		writeError(w, http.StatusBadRequest, "invalid simulation scenario: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toSimulationResponse(result))
}

type errorResponse struct {
	Error string `json:"error"`
}

type latencyResponse struct {
	Count   int     `json:"count"`
	Min     int64   `json:"min"`
	Max     int64   `json:"max"`
	Average float64 `json:"average"`
}

type summaryResponse struct {
	TotalRequests        int             `json:"totalRequests"`
	CreatedRequests      int             `json:"createdRequests"`
	QueuedRequests       int             `json:"queuedRequests"`
	RetryWaitingRequests int             `json:"retryWaitingRequests"`
	InFlightRequests     int             `json:"inFlightRequests"`
	SucceededRequests    int             `json:"succeededRequests"`
	FailedRequests       int             `json:"failedRequests"`
	CancelledRequests    int             `json:"cancelledRequests"`
	TimedOutRequests     int             `json:"timedOutRequests"`
	CompletedRequests    int             `json:"completedRequests"`
	Retries              int             `json:"retries"`
	EndToEndLatency      latencyResponse `json:"endToEndLatency"`
	ServiceLatency       latencyResponse `json:"serviceLatency"`
}

type workerMetricsResponse struct {
	Capacity            int     `json:"capacity"`
	BusyWorkers         int     `json:"busyWorkers"`
	WaitingRequests     int     `json:"waitingRequests"`
	Utilization         float64 `json:"utilization"`
	Online              bool    `json:"online"`
	CircuitState        uint8   `json:"circuitState"`
	ConsecutiveFailures int     `json:"consecutiveFailures"`
	ProbeInFlight       bool    `json:"probeInFlight"`
}

type metricsResponse struct {
	TotalRequests     int                              `json:"totalRequests"`
	CreatedRequests   int                              `json:"createdRequests"`
	QueuedRequests    int                              `json:"queuedRequests"`
	InFlightRequests  int                              `json:"inFlightRequests"`
	SucceededRequests int                              `json:"succeededRequests"`
	FailedRequests    int                              `json:"failedRequests"`
	CancelledRequests int                              `json:"cancelledRequests"`
	TimedOutRequests  int                              `json:"timedOutRequests"`
	BusyWorkers       int                              `json:"busyWorkers"`
	ServiceCapacity   int                              `json:"serviceCapacity"`
	WaitingRequests   int                              `json:"waitingRequests"`
	WorkerUtilization float64                          `json:"workerUtilization"`
	Workers           map[string]workerMetricsResponse `json:"workers"`
}

type simulationResponse struct {
	Name        string                         `json:"name"`
	Duration    int64                          `json:"duration"`
	Summary     summaryResponse                `json:"summary"`
	Metrics     metricsResponse                `json:"metrics"`
	RouteTraces map[string][]sim.RouteTraceHop `json:"routeTraces"`
}

func toSimulationResponse(result sim.ScenarioResult) simulationResponse {
	s := result.Summary
	m := result.Metrics

	workers := make(map[string]workerMetricsResponse, len(m.Workers))
	for id, worker := range m.Workers {
		workers[id] = workerMetricsResponse{
			Capacity:            worker.Capacity,
			BusyWorkers:         worker.BusyWorkers,
			WaitingRequests:     worker.WaitingRequests,
			Utilization:         worker.Utilization,
			Online:              worker.Online,
			CircuitState:        uint8(worker.CircuitState),
			ConsecutiveFailures: worker.ConsecutiveFailures,
			ProbeInFlight:       worker.ProbeInFlight,
		}
	}

	return simulationResponse{
		Name:        result.Name,
		Duration:    int64(result.Duration),
		RouteTraces: result.RouteTraces,
		Summary: summaryResponse{
			TotalRequests:        s.TotalRequests,
			CreatedRequests:      s.CreatedRequests,
			QueuedRequests:       s.QueuedRequests,
			RetryWaitingRequests: s.RetryWaitingRequests,
			InFlightRequests:     s.InFlightRequests,
			SucceededRequests:    s.SucceededRequests,
			FailedRequests:       s.FailedRequests,
			CancelledRequests:    s.CancelledRequests,
			TimedOutRequests:     s.TimedOutRequests,
			CompletedRequests:    s.CompletedRequests,
			Retries:              s.Retries,
			EndToEndLatency: latencyResponse{
				Count:   s.EndToEndLatency.Count,
				Min:     int64(s.EndToEndLatency.Min),
				Max:     int64(s.EndToEndLatency.Max),
				Average: s.EndToEndLatency.Average,
			},
			ServiceLatency: latencyResponse{
				Count:   s.ServiceLatency.Count,
				Min:     int64(s.ServiceLatency.Min),
				Max:     int64(s.ServiceLatency.Max),
				Average: s.ServiceLatency.Average,
			},
		},
		Metrics: metricsResponse{
			TotalRequests:     m.TotalRequests,
			CreatedRequests:   m.CreatedRequests,
			QueuedRequests:    m.QueuedRequests,
			InFlightRequests:  m.InFlightRequests,
			SucceededRequests: m.SucceededRequests,
			FailedRequests:    m.FailedRequests,
			CancelledRequests: m.CancelledRequests,
			TimedOutRequests:  m.TimedOutRequests,
			BusyWorkers:       m.BusyWorkers,
			ServiceCapacity:   m.ServiceCapacity,
			WaitingRequests:   m.WaitingRequests,
			WorkerUtilization: m.WorkerUtilization,
			Workers:           workers,
		},
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
