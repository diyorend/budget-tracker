package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/diyorend/budget-tracker/internal/domain"
	"github.com/diyorend/budget-tracker/internal/repository"
)

// AlertPublisher is the contract the transaction service depends on for
// pushing alerts. Satisfied by *alert.Broker. Defined here (consumer side)
// so tests can pass a mock without importing the alert package at all.
type AlertPublisher interface {
	Publish(ctx context.Context, userID string, msg domain.AlertMessage) error
}

type TransactionService struct {
	txRepo     repository.TransactionStore
	budgetRepo repository.BudgetStore
	broker     AlertPublisher
}

func NewTransactionService(
	txRepo repository.TransactionStore,
	budgetRepo repository.BudgetStore,
	broker AlertPublisher,
) *TransactionService {
	return &TransactionService{txRepo: txRepo, budgetRepo: budgetRepo, broker: broker}
}

func (s *TransactionService) Create(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error) {
	created, err := s.txRepo.Create(ctx, t)
	if err != nil {
		return nil, err
	}

	// After creating a transaction, check if any budget is exceeded.
	// Run in a goroutine so the HTTP response isn't delayed by the check.
	go func() {
		// Fresh context: the request context is cancelled the moment the
		// HTTP handler returns, which would kill this check mid-flight.
		checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.checkBudgets(checkCtx, t.UserID, t.Category, t.Date); err != nil {
			slog.Error("budget check failed", "user_id", t.UserID, "category", t.Category, "err", err)
		}
	}()

	return created, nil
}

func (s *TransactionService) List(ctx context.Context, userID string, limit, offset int) ([]*domain.Transaction, error) {
	return s.txRepo.ListByUser(ctx, userID, limit, offset)
}

func (s *TransactionService) checkBudgets(ctx context.Context, userID, category string, date time.Time) error {
	budget, err := s.budgetRepo.GetByUserCategoryMonth(ctx, userID, category, date)
	if err != nil {
		// No budget set for this category — nothing to check, not an error.
		return nil
	}

	if budget.LimitAmount <= 0 {
		return nil
	}

	sums, err := s.txRepo.SumByCategory(ctx, userID, date)
	if err != nil {
		return err
	}

	spent := sums[category]
	if spent == 0 {
		return nil
	}

	percentage := (spent / budget.LimitAmount) * 100

	var alertMsg *domain.AlertMessage
	switch {
	case percentage >= 100:
		alertMsg = &domain.AlertMessage{
			Type:       "budget_alert",
			Category:   category,
			Spent:      spent,
			Limit:      budget.LimitAmount,
			Percentage: percentage,
			Message:    fmt.Sprintf("You have exceeded your %s budget! Spent %.2f of %.2f", category, spent, budget.LimitAmount),
		}
	case percentage >= 80:
		alertMsg = &domain.AlertMessage{
			Type:       "budget_alert",
			Category:   category,
			Spent:      spent,
			Limit:      budget.LimitAmount,
			Percentage: percentage,
			Message:    fmt.Sprintf("You have used %.0f%% of your %s budget", percentage, category),
		}
	}

	if alertMsg != nil {
		return s.broker.Publish(ctx, userID, *alertMsg)
	}
	return nil
}
