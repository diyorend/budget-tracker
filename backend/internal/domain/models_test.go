package domain_test

import (
	"strings"
	"testing"

	"github.com/diyorend/budget-tracker/internal/domain"
)

// TestNewBudgetAlert locks down the 80%/100% threshold behavior that both
// TransactionService (on every new transaction) and WSHandler (on connect,
// as a catch-up for alerts that fired while nobody was listening) depend
// on. If this drifts, both callers silently drift with it.
func TestNewBudgetAlert(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		wantNil    bool
		wantWord   string // substring expected in the message
	}{
		{"under threshold", 50, true, ""},
		{"just under 80", 79.99, true, ""},
		{"at 80 exactly", 80, false, "used"},
		{"between 80 and 100", 92, false, "used"},
		{"at 100 exactly", 100, false, "exceeded"},
		{"over 100", 150, false, "exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := domain.NewBudgetAlert("Food", 100, 100, tt.percentage)
			if tt.wantNil {
				if msg != nil {
					t.Fatalf("expected nil alert at %.2f%%, got %+v", tt.percentage, msg)
				}
				return
			}
			if msg == nil {
				t.Fatalf("expected an alert at %.2f%%, got nil", tt.percentage)
			}
			if msg.Type != "budget_alert" {
				t.Errorf("Type = %q, want %q", msg.Type, "budget_alert")
			}
			if msg.Category != "Food" {
				t.Errorf("Category = %q, want %q", msg.Category, "Food")
			}
			if !strings.Contains(msg.Message, tt.wantWord) {
				t.Errorf("Message = %q, want it to contain %q", msg.Message, tt.wantWord)
			}
		})
	}
}
