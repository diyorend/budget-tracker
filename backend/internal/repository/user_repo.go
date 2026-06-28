package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/diyorend/budget-tracker/internal/domain"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, email, hashedPassword string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (email, password)
		VALUES ($1, $2)
		RETURNING id, email, password, created_at`,
		email, hashedPassword,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)

	if err != nil {
		// pgx error code 23505 = unique_violation
		if err.Error() != "" && isUniqueViolation(err) {
			return nil, fmt.Errorf("email %s: %w", email, domain.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("UserRepo.Create: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password, created_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user %s: %w", email, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("UserRepo.GetByEmail: %w", err)
	}
	return &u, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && len(err.Error()) > 0 && (contains(err.Error(), "23505") || contains(err.Error(), "unique"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
