package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/diyorend/budget-tracker/internal/domain"
)

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

func (r *TransactionRepo) Create(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error) {
	var out domain.Transaction
	err := r.db.QueryRow(ctx,
		`INSERT INTO transactions (user_id, amount, category, description, date)
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id, user_id, amount, category, description, date, created_at`,
		t.UserID, t.Amount, t.Category, t.Description, t.Date,
	).Scan(&out.ID, &out.UserID, &out.Amount, &out.Category, &out.Description, &out.Date, &out.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("TransactionRepo.Create: %w", err)
	}
	return &out, nil
}

func (r *TransactionRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, amount, category, description, date, created_at 
		FROM transactions 
		WHERE user_id = $1 
		ORDER BY date DESC, created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("TransactionRepo.ListByUser: %w", err)
	}
	
	defer rows.Close()

	var txs []*domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Amount, &t.Category, &t.Description, &t.Date, &t.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, &t)
	}
	return txs, rows.Err()
}

// SumByCategory returns total spending per category for a given user/month
// This is the key query - called after every transaction insert to check budgets
func (r *TransactionRepo) SumByCategory(ctx context.Context, userID string, month time.Time) (map[string]float64, error) {
	rows, err := r.db.Query(ctx,
		`SELECT category, COALESCE(SUM(amount), 0) as total 
		FROM transactions 
		WHERE user_id = $1 
			AND date >= date_trunc('month', $2::date)
			AND date < date_trunc('month', $2::date) + INTERVAL '1 month'
		GROUP BY category`,
		userID, month,
	)

	if err != nil {
		return nil, fmt.Errorf("TransactionRepo.SumByCategory: %w", err)
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var category string
		var total float64
		if err := rows.Scan(&category, &total); err != nil {
			return nil, err
		}
		result[category] = total
	}
	return result, rows.Err()
}
