package location

import (
	"testing"

	"carepath/apps/api/internal/platform/apperr"
)

func floatPtr(f float64) *float64 { return &f }

func TestObservationValidate(t *testing.T) {
	tests := []struct {
		name string
		obs  Observation
		want error // nil = valid
	}{
		{
			name: "minimal fix is valid",
			obs:  Observation{Source: SourceQR, NodeID: "I-1301/node-reception"},
		},
		{
			name: "full fix with optional metadata is valid",
			obs: Observation{
				Source:     SourceZigbee,
				NodeID:     "I-1301/node-lift",
				FloorID:    "I-1301",
				Confidence: floatPtr(0.87),
			},
		},
		{
			name: "missing source",
			obs:  Observation{NodeID: "I-1301/node-reception"},
			want: apperr.New(apperr.KindInvalid, "observation missing source"),
		},
		{
			name: "missing node",
			obs:  Observation{Source: SourceManual},
			want: apperr.New(apperr.KindInvalid, "observation missing nodeId"),
		},
		{
			name: "confidence above one",
			obs: Observation{
				Source:     SourceZigbee,
				NodeID:     "I-1301/node-lift",
				Confidence: floatPtr(1.5),
			},
			want: apperr.New(apperr.KindInvalid, "observation confidence must be within [0,1]"),
		},
		{
			name: "confidence below zero",
			obs: Observation{
				Source:     SourceZigbee,
				NodeID:     "I-1301/node-lift",
				Confidence: floatPtr(-0.1),
			},
			want: apperr.New(apperr.KindInvalid, "observation confidence must be within [0,1]"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.obs.Validate()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if apperr.KindOf(err) != apperr.KindInvalid || err.Error() != tt.want.Error() {
				t.Fatalf("Validate() = %v, want %v", err, tt.want)
			}
		})
	}
}
