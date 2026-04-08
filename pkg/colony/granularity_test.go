package colony

import "testing"

// ---------------------------------------------------------------------------
// PlanGranularity.Valid() tests
// ---------------------------------------------------------------------------

func TestPlanGranularityValid(t *testing.T) {
	tests := []struct {
		g    PlanGranularity
		want bool
	}{
		{GranularitySprint, true},
		{GranularityMilestone, true},
		{GranularityQuarter, true},
		{GranularityMajor, true},
		{"", false},
		{"invalid", false},
		{"SPRINT", false},
		{"Sprint", false},
		{"banana", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.g), func(t *testing.T) {
			if got := tt.g.Valid(); got != tt.want {
				t.Errorf("PlanGranularity(%q).Valid() = %v, want %v", tt.g, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GranularityRange tests
// ---------------------------------------------------------------------------

func TestGranularityRange(t *testing.T) {
	tests := []struct {
		g       PlanGranularity
		wantMin int
		wantMax int
	}{
		{GranularitySprint, 1, 3},
		{GranularityMilestone, 4, 7},
		{GranularityQuarter, 8, 12},
		{GranularityMajor, 13, 20},
		{"unknown", 1, 3},
		{"", 1, 3},
	}
	for _, tt := range tests {
		name := string(tt.g)
		if name == "" {
			name = "(empty)"
		}
		t.Run(name, func(t *testing.T) {
			min, max := GranularityRange(tt.g)
			if min != tt.wantMin || max != tt.wantMax {
				t.Errorf("GranularityRange(%q) = (%d, %d), want (%d, %d)", tt.g, min, max, tt.wantMin, tt.wantMax)
			}
		})
	}
}
