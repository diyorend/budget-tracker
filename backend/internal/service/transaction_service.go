package service

import (
	"context"
	"fmt"
	"time"

	"github.com/diyorend/budget-tracker/internal/alert"
	"github.com/diyorend/budget-tracker/internal/domain"
	"github.com/diyorend/budget-tracker/internal/repository"
)

type TransactionService struct {
	txRepo	*repository.TransactionRepo
	budgetRepo	*repository.BudgetRepo
	broker	*alert.Broker
}

func NewTransactionService(
	txRepo	*repository.TransactionRepo,
	budgetRepo	*repository.BudgetRepo,
	broker	*alert.Broker,
) *TransactionService {
	return &TransactionService{txRepo: txRepo, budgetRepo: budgetRepo, broker: broker}
}

func (s *TransactionService) Create(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error) {
	created, err := s.txRepo.Create(ctx, t)
	if err != nil {
		return nil, err
	}
	// After creating a transaction, check if any budget is exceeded
	// Run in goroutine so the HTTP response isn't delayed by the check
	go func() {
		checkCtx := context.Background() // fresh context, the request context may be done
		if err := s.checkBudgets(checkCtx, t.UserID, t.Category, t.Date); err != nil {
			fmt.Printf("budget check error: %v\n", err)
		}
	}()
	return created, nil
}

func (s *TransactionService) List(ctx context.Context, userID string, limit, offset int) ([]*domain.Transaction, error) {
	return s.txRepo.ListByUser(ctx, userID, limit, offset)
}

func (s *TransactionService) checkBudgets(ctx context.Context, userID, category string, date time.Time) error {
	// Get the budget for this category/month
	budget, err := s.budgetRepo.GetByUserCategoryMonth(ctx, userID, category, date)
	if err != nil {
		// No budget set for this category - nothing to check
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

	// Alert thresholds: 80%, 100%
	var alertMsg *domain.AlertMessage

	switch {
	case percentage >= 100:
		alertMsg = &domain.AlertMessage{
			Type: "budget_alert",
			Category: category,
			Spent: spent,
			Limit: budget.LimitAmount,
			Percentage: percentage,
			Message: fmt.Sprintf("You have exceeded your %s budget! Spent %.2f of %.2f", category, spent, budget.LimitAmount),
		}
	case percentage >= 80:
		alertMsg = &domain.AlertMessage{
			Type: "budget_alert",
			Category: category,
			Spent: spent,
			Limit: budget.LimitAmount,
			Percentage: percentage,
			Message: fmt.Sprintf("You have used %.0f of your %s budget", percentage, category),
		}
	}

	if alertMsg != nil {
		return s.broker.Publish(ctx, userID, *alertMsg)
	}
	return nil
}
