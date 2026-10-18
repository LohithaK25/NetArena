package sim

import "testing"

func TestRoutingAlgorithmValidate(t *testing.T) {
	tests := []struct {
		name    string
		algo    RoutingAlgorithm
		wantErr bool
	}{
		{
			name: "round robin",
			algo: RoutingRoundRobin,
		},
		{
			name: "shortest path",
			algo: RoutingShortestPath,
		},
		{
			name:    "empty algorithm",
			algo:    "",
			wantErr: true,
		},
		{
			name:    "unsupported algorithm",
			algo:    RoutingAlgorithm("random"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.algo.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"Validate() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
