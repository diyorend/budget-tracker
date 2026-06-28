package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/diyorend/budget-tracker/internal/domain"
)

type BudgetRepo struct {
	db *pgxpool.Pool
}

func NewBudgetRepo(db *pgxpool.Pool) *BudgetRepo {
	return &BudgetRepo{db: db}
}

func (r *BudgetRepo) Upsert(ctx context.Context, b *domain.Budget) (*domain.Budget, error) {
	var out domain.Budget
	err := r.db.QueryRow(ctx,
		`INSERT INTO budgets (user_id, category, limit_amount, month)
		VALUES ($1, $2, $3, date_trunc('month', $4::date))
		ON CONFLICT (user_id, category, month)
		DO UPDATE SET limit_amount = EXCLUDED.limit_amount
		RETURNING id, user_id, category, limit_amount, month, created_at`,
		b.UserID, b.Category, b.LimitAmount, b.Month,
	).Scan(&out.ID, &out.UserID, &out.Category, &out.LimitAmount, &out.Month, &out.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("BudgetRepo.Upsert: %w", err)
	}

	return &out, nil
}

func (r *BudgetRepo) ListByUserMonth(ctx context.Context, userID string, month time.Time) ([]*domain.Budget, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, category, limit_amount, month, created_at 
		FROM budgets
		WHERE user_id = $1
			AND month = date_trunc('month', $2::date)`,
		userID, month,
	)
	if err != nil {
		return nil, fmt.Errorf("BudgetRepo.ListByUserMonth: %w", err)
	}
	defer rows.Close()

	var budgets []*domain.Budget
	for rows.Next() {
		var b domain.Budget
		if err := rows.Scan(&b.ID, &b.UserID, &b.Category, &b.LimitAmount, &b.Month, &b.CreatedAt); err != nil {
			return nil, err
		}
		budgets = append(budgets, &b)
	}
	return budgets, rows.Err()
}

func (r *BudgetRepo) GetByUserCategoryMonth (ctx context.Context, userID, category string, month time.Time) (*domain.Budget, error) {
	var b domain.Budget
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, category, limit_amount, month, created_at
		FROM budgets
		WHERE user_id = $1 AND category = $2
			AND month = date_trunc('month', $3::date)`,
		userID, category, month,
	).Scan(&b.ID, &b.UserID, &b.Category, &b.LimitAmount, &b.Month, &b.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("BudgetRepo.GetByUserCategoryMonth: %w", err)
	}
	return &b, nil
}
