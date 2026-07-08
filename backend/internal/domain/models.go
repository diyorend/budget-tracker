package domain

import (
	"fmt"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Transaction struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Amount      float64   `json:"amount"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	CreatedAt   time.Time `json:"created_at"`
}

type Budget struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Category    string    `json:"category"`
	LimitAmount float64   `json:"limit_amount"`
	Month       time.Time `json:"month"`
	CreatedAt   time.Time `json:"created_at"`
}

type BudgetStatus struct {
	Budget
	Spent      float64 `json:"spent"`
	Remaining  float64 `json:"remaining"`
	Percentage float64 `json:"percentage"`
}

type AlertMessage struct {
	Type       string  `json:"type"`
	Category   string  `json:"category"`
	Spent      float64 `json:"spent"`
	Limit      float64 `json:"limit"`
	Percentage float64 `json:"percentage"`
	Message    string  `json:"message"`
}

// NewBudgetAlert builds the alert payload for a category once spending
// crosses the 80% or 100% threshold. Returns nil below 80% — nothing to
// alert on. Shared by the two places that need this exact same threshold
// logic: TransactionService.checkBudgets (fires right after a new
// transaction) and WSHandler's connect-time catch-up (fires for anyone
// reconnecting to a budget that's already past the line), so the two
// can't drift out of sync on wording or thresholds.
func NewBudgetAlert(category string, spent, limit, percentage float64) *AlertMessage {
	switch {
	case percentage >= 100:
		return &AlertMessage{
			Type:       "budget_alert",
			Category:   category,
			Spent:      spent,
			Limit:      limit,
			Percentage: percentage,
			Message:    fmt.Sprintf("You have exceeded your %s budget! Spent %.2f of %.2f", category, spent, limit),
		}
	case percentage >= 80:
		return &AlertMessage{
			Type:       "budget_alert",
			Category:   category,
			Spent:      spent,
			Limit:      limit,
			Percentage: percentage,
			Message:    fmt.Sprintf("You have used %.0f%% of your %s budget", percentage, category),
		}
	default:
		return nil
	}
}

// Sentinel Errors
var (
	ErrNotFound      = fmt.Errorf("not found")
	ErrUnauthorized  = fmt.Errorf("unauthorized")
	ErrAlreadyExists = fmt.Errorf("already exists")
)
