package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"github.com/diyorend/budget-tracker/internal/domain"
	"github.com/diyorend/budget-tracker/internal/repository"
)

type AuthService struct {
	userRepo *repository.UserRepo
	jwtSecret string
	expiry time.Duration
}

func NewAuthService(repo *repository.UserRepo, secret string, expiryHours int) *AuthService {
	return &AuthService{
		userRepo: repo,
		jwtSecret: secret,
		expiry: time.Duration(expiryHours) * time.Hour,
	}
}

type Claims struct {
	UserID	string	`json:"user_id"`
	Email	string	`json:"email"`
	jwt.RegisteredClaims
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("AuthService.Register hash: %w", err)
	}
	return s.userRepo.Create(ctx, email, string(hash))
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, fmt.Errorf("%w", domain.ErrUnauthorized)
	}
	
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, fmt.Errorf("%w", domain.ErrUnauthorized)
	}
	
	token, err := s.generateToken(user)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *AuthService) generateToken(user *domain.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiry)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w", domain.ErrUnauthorized)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("%w", domain.ErrUnauthorized)
	}
	return claims, nil
}

