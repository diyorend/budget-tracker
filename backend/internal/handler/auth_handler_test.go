package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/diyorend/budget-tracker/internal/domain"
	"github.com/diyorend/budget-tracker/internal/handler"
	"github.com/diyorend/budget-tracker/internal/service"
)

// mockUserStore implements repository.UserStore in-memory, no Postgres needed.
type mockUserStore struct {
	byEmail map[string]*domain.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{byEmail: make(map[string]*domain.User)}
}

func (m *mockUserStore) Create(ctx context.Context, email, hashedPassword string) (*domain.User, error) {
	if _, exists := m.byEmail[email]; exists {
		return nil, domain.ErrAlreadyExists
	}
	u := &domain.User{ID: "u-" + email, Email: email, Password: hashedPassword}
	m.byEmail[email] = u
	return u, nil
}

func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func newTestAuthHandler() *handler.AuthHandler {
	store := newMockUserStore()
	authSvc := service.NewAuthService(store, "test-secret-at-least-32-characters-long", 24)
	return handler.NewAuthHandler(authSvc)
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"missing password", `{"email":"a@b.com"}`, http.StatusBadRequest},
		{"short password", `{"email":"a@b.com","password":"short"}`, http.StatusBadRequest},
		{"valid registration", `{"email":"a@b.com","password":"password123"}`, http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			h := newTestAuthHandler()

			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if err := h.Register(c); err != nil {
				t.Fatalf("handler returned error: %v", err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestRegister_DuplicateEmailReturns409(t *testing.T) {
	e := echo.New()
	h := newTestAuthHandler()

	body := `{"email":"dupe@b.com","password":"password123"}`

	// First registration succeeds
	req1 := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	if err := h.Register(c1); err != nil {
		t.Fatalf("first register returned error: %v", err)
	}
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first register status = %d, want %d", rec1.Code, http.StatusCreated)
	}

	// Second registration with same email must conflict
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	if err := h.Register(c2); err != nil {
		t.Fatalf("second register returned error: %v", err)
	}
	if rec2.Code != http.StatusConflict {
		t.Errorf("second register status = %d, want %d", rec2.Code, http.StatusConflict)
	}
}

func TestLogin(t *testing.T) {
	e := echo.New()
	h := newTestAuthHandler()

	// Register a user first
	regBody := `{"email":"login@b.com","password":"password123"}`
	regReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(regBody))
	regReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	regRec := httptest.NewRecorder()
	if err := h.Register(e.NewContext(regReq, regRec)); err != nil {
		t.Fatalf("setup register failed: %v", err)
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"correct credentials", `{"email":"login@b.com","password":"password123"}`, http.StatusOK},
		{"wrong password", `{"email":"login@b.com","password":"wrongpass"}`, http.StatusUnauthorized},
		{"unknown email", `{"email":"nobody@b.com","password":"password123"}`, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if err := h.Login(c); err != nil {
				t.Fatalf("handler returned error: %v", err)
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp["token"] == nil || resp["token"] == "" {
					t.Error("expected non-empty token in response")
				}
			}
		})
	}
}
