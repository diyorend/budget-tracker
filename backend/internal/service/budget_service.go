package service

import (
	"context"
	"time"

	"github.com/diyorend/budget-tracker/internal/domain"
	"github.com/diyorend/budget-tracker/internal/repository"
)

type BudgetService struct {
	budgetRepo	*repository.BudgetRepo
	txRepo	*repository.TransactionRepo
}

func NewBudgetService(budgetRepo *repository.BudgetRepo, txRepo *repository.TransactionRepo) *BudgetService {
	return &BudgetService{budgetRepo: budgetRepo, txRepo: txRepo}
}

func (s *BudgetService) Upsert(ctx context.Context, b *domain.Budget) (*domain.Budget, error) {
	return s.budgetRepo.Upsert(ctx, b)
}

func (s *BudgetService) GetStatus(ctx context.Context, userID string, month time.Time) ([]*domain.BudgetStatus, error) {
	budgets, err := s.budgetRepo.ListByUserMonth(ctx, userID, month)
	if err != nil {
		return nil, err
	}

	sums, err := s.txRepo.SumByCategory(ctx, userID, month)
	if err != nil {
		return nil, err
	}

	var statuses []*domain.BudgetStatus
	for _, b := range budgets {
		spent := sums[b.Category]
		remaining := b.LimitAmount - spent
		if remaining < 0 {
			remaining = 0
		}
		pct := 0.0
		if b.LimitAmount > 0 {
			pct = (spent / b.LimitAmount) * 100
		}
		statuses = append(statuses, &domain.BudgetStatus{
			Budget: *b,
			Spent: spent,
			Remaining: remaining,
			Percentage: pct,
		})
	}
	return statuses, nil
}

