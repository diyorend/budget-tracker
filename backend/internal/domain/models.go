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

// Sentinel Errors
var (
	ErrNotFound      = fmt.Errorf("not found")
	ErrUnauthorized  = fmt.Errorf("unauthorized")
	ErrAlreadyExists = fmt.Errorf("already exists")
)
