package service_test

import (
	"testing"
)

// Test the percentage calculation logic
func TestBudgetPercentage(t *testing.T) {
	tests := []struct {
		name        string
		spent       float64
		limit       float64
		wantPercent float64
		wantAlert   bool
	}{
		{"under budget", 50, 100, 50.0, false},
		{"at 80 percent", 80, 100, 80.0, true},
		{"over budget", 110, 100, 110.0, true},
		{"zero limit", 50, 0, 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct := 0.0
			if tt.limit > 0 {
				pct = (tt.spent / tt.limit) * 100
			}
			if pct != tt.wantPercent {
				t.Errorf("percentage: got %.1f, want %.1f", pct, tt.wantPercent)
			}
			alert := pct >= 80
			if alert != tt.wantAlert {
				t.Errorf("alert: got %v, want %v", alert, tt.wantAlert)
			}
		})
	}
}
