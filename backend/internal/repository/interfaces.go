package repository

import (
	"context"
	"time"

	"github.com/diyorend/budget-tracker/internal/domain"
)

// UserStore is the contract the auth service depends on.
// Defined here so both the real repo and any test mock implement it.
type UserStore interface {
	Create(ctx context.Context, email, hashedPassword string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// TransactionStore is the contract the transaction and budget services depend on.
type TransactionStore interface {
	Create(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Transaction, error)
	SumByCategory(ctx context.Context, userID string, month time.Time) (map[string]float64, error)
}

// BudgetStore is the contract the budget service depends on.
type BudgetStore interface {
	Upsert(ctx context.Context, b *domain.Budget) (*domain.Budget, error)
	ListByUserMonth(ctx context.Context, userID string, month time.Time) ([]*domain.Budget, error)
	GetByUserCategoryMonth(ctx context.Context, userID, category string, month time.Time) (*domain.Budget, error)
}
