package sim

import "testing"

func TestEventValidate(t *testing.T) {
	tests := []struct {
		name      string
		event     Event
		wantError bool
	}{
		{
			name: "valid arrival",
			event: Event{
				At:        0,
				Sequence:  1,
				Type:      EventRequestArrives,
				RequestID: "req-1",
			},
		},
		{
			name: "negative tick",
			event: Event{
				At:        -1,
				Sequence:  1,
				Type:      EventRequestArrives,
				RequestID: "req-1",
			},
			wantError: true,
		},
		{
			name: "zero sequence",
			event: Event{
				At:        1,
				Sequence:  0,
				Type:      EventRequestArrives,
				RequestID: "req-1",
			},
			wantError: true,
		},
		{
			name: "unknown event type",
			event: Event{
				At:        1,
				Sequence:  1,
				Type:      EventType(255),
				RequestID: "req-1",
			},
			wantError: true,
		},
		{
			name: "empty request ID",
			event: Event{
				At:        1,
				Sequence:  1,
				Type:      EventRequestArrives,
				RequestID: "",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()

			if tt.wantError && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      string
	}{
		{EventRequestArrives, "request_arrives"},
		{EventRequestTimeout, "request_timeout"},
		{EventServiceCompletes, "service_completes"},
		{EventType(255), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
