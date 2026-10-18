package main

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		p      float64
		want   float64
	}{
		{
			name:   "median of odd count",
			values: []float64{5, 1, 3},
			p:      0.50,
			want:   3,
		},
		{
			name:   "median of even count rounds up",
			values: []float64{4, 1, 3, 2},
			p:      0.50,
			want:   2,
		},
		{
			name:   "95th percentile",
			values: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			p:      0.95,
			want:   10,
		},
		{
			name:   "single value",
			values: []float64{7.25},
			p:      0.50,
			want:   7.25,
		},
		{
			name:   "empty values",
			values: nil,
			p:      0.50,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.values, tt.p)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("percentile(%v, %.2f) = %v, want %v",
					tt.values, tt.p, got, tt.want)
			}
		})
	}
}

func TestValidateResponse(t *testing.T) {
	var valid simulationResponse
	valid.Name = "benchmark-1"
	valid.Summary.TotalRequests = 5
	valid.Summary.SucceededRequests = 5

	tests := []struct {
		name string
		body simulationResponse
		want bool
	}{
		{"valid response", valid, true},
		{"wrong name", func() simulationResponse {
			body := valid
			body.Name = "another-run"
			return body
		}(), false},
		{"wrong total", func() simulationResponse {
			body := valid
			body.Summary.TotalRequests = 4
			return body
		}(), false},
		{"incomplete successes", func() simulationResponse {
			body := valid
			body.Summary.SucceededRequests = 4
			return body
		}(), false},
		{"reported failures", func() simulationResponse {
			body := valid
			body.Summary.FailedRequests = 1
			return body
		}(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateResponse(tt.body, "benchmark-1", 5)
			if got != tt.want {
				t.Fatalf("validateResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummarizeBatch(t *testing.T) {
	tests := []struct {
		name    string
		results []requestResult
		elapsed time.Duration
		want    batchSummary
	}{
		{
			name: "successful requests",
			results: []requestResult{
				{latency: 1 * time.Millisecond, httpOK: true, valid: true},
				{latency: 2 * time.Millisecond, httpOK: true, valid: true},
				{latency: 3 * time.Millisecond, httpOK: true, valid: true},
				{latency: 4 * time.Millisecond, httpOK: true, valid: true},
			},
			elapsed: 2 * time.Second,
			want: batchSummary{
				throughput: 2,
				p50:        2,
				p95:        4,
			},
		},
		{
			name: "HTTP failure and invalid response",
			results: []requestResult{
				{latency: 1 * time.Millisecond, httpOK: false, err: errors.New("HTTP 500")},
				{latency: 2 * time.Millisecond, httpOK: true, valid: false, err: errors.New("invalid JSON")},
			},
			elapsed: 1 * time.Second,
			want: batchSummary{
				throughput:     2,
				p50:            2,
				p95:            2,
				httpErrors:     1,
				invalidResults: 1,
			},
		},
		{
			name:    "zero elapsed time",
			results: []requestResult{},
			elapsed: 0,
			want:    batchSummary{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeBatch(tt.results, tt.elapsed)

			if got != tt.want {
				t.Fatalf("summarizeBatch() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
