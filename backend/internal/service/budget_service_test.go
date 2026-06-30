package service_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/diyorend/budget-tracker/internal/domain"
	"github.com/diyorend/budget-tracker/internal/service"
)

// ---- Mocks ----
// These exist only because TransactionService now depends on interfaces
// (repository.TransactionStore, repository.BudgetStore, service.AlertPublisher)
// instead of concrete *repository.X types. That's the whole point of the
// interface refactor: no real Postgres or Redis needed to test business logic.

type mockTxStore struct {
	createFn  func(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error)
	sums      map[string]float64
	listItems []*domain.Transaction
}

func (m *mockTxStore) Create(ctx context.Context, t *domain.Transaction) (*domain.Transaction, error) {
	if m.createFn != nil {
		return m.createFn(ctx, t)
	}
	t.ID = "tx-1"
	return t, nil
}

func (m *mockTxStore) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Transaction, error) {
	return m.listItems, nil
}

func (m *mockTxStore) SumByCategory(ctx context.Context, userID string, month time.Time) (map[string]float64, error) {
	return m.sums, nil
}

type mockBudgetStore struct {
	budget *domain.Budget
	err    error
}

func (m *mockBudgetStore) Upsert(ctx context.Context, b *domain.Budget) (*domain.Budget, error) {
	return b, nil
}

func (m *mockBudgetStore) ListByUserMonth(ctx context.Context, userID string, month time.Time) ([]*domain.Budget, error) {
	if m.budget == nil {
		return nil, nil
	}
	return []*domain.Budget{m.budget}, nil
}

func (m *mockBudgetStore) GetByUserCategoryMonth(ctx context.Context, userID, category string, month time.Time) (*domain.Budget, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.budget, nil
}

type mockBroker struct {
	published []domain.AlertMessage
	ch        chan domain.AlertMessage // optional, lets tests wait for the goroutine
}

func (m *mockBroker) Publish(ctx context.Context, userID string, msg domain.AlertMessage) error {
	m.published = append(m.published, msg)
	if m.ch != nil {
		m.ch <- msg
	}
	return nil
}

// ---- Helpers ----

// almostEqual compares floats with a tolerance, because direct == on
// floating point arithmetic (e.g. 110.0/100.0*100.0) is not reliable —
// that division/multiplication can land on 110.00000000000001.
func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

// ---- Tests ----

func TestBudgetPercentageCalculation(t *testing.T) {
	tests := []struct {
		name        string
		spent       float64
		limit       float64
		wantPercent float64
		wantAlert   bool
	}{
		{"under budget", 50, 100, 50.0, false},
		{"at 80 percent", 80, 100, 80.0, true},
		{"over budget", 110, 100, 110.0, true},
		{"zero limit", 50, 0, 0.0, false},
		{"exactly zero spent", 0, 100, 0.0, false},
	}

	const epsilon = 1e-9

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct := 0.0
			if tt.limit > 0 {
				pct = (tt.spent / tt.limit) * 100
			}
			if !almostEqual(pct, tt.wantPercent, epsilon) {
				t.Errorf("percentage: got %.10f, want %.10f", pct, tt.wantPercent)
			}
			alert := pct >= 80
			if alert != tt.wantAlert {
				t.Errorf("alert: got %v, want %v", alert, tt.wantAlert)
			}
		})
	}
}

// TestTransactionService_Create_PublishesAlertWhenOverBudget exercises the
// real service end to end against mocks: create a transaction, let the
// background goroutine run checkBudgets, assert an alert was published.
func TestTransactionService_Create_PublishesAlertWhenOverBudget(t *testing.T) {
	budget := &domain.Budget{
		ID:          "b1",
		UserID:      "u1",
		Category:    "Food",
		LimitAmount: 100,
		Month:       time.Now(),
	}

	broker := &mockBroker{ch: make(chan domain.AlertMessage, 1)}
	txStore := &mockTxStore{
		sums: map[string]float64{"Food": 150}, // already over budget after this tx
	}
	budgetStore := &mockBudgetStore{budget: budget}

	svc := service.NewTransactionService(txStore, budgetStore, broker)

	tx := &domain.Transaction{UserID: "u1", Category: "Food", Amount: 50, Date: time.Now()}
	_, err := svc.Create(context.Background(), tx)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	select {
	case msg := <-broker.ch:
		if msg.Category != "Food" {
			t.Errorf("alert category = %q, want %q", msg.Category, "Food")
		}
		if msg.Percentage < 100 {
			t.Errorf("alert percentage = %.2f, want >= 100", msg.Percentage)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for alert — checkBudgets goroutine never published")
	}
}

// TestTransactionService_Create_NoAlertWhenNoBudgetSet ensures the service
// doesn't error or publish anything when the user hasn't set a budget yet.
func TestTransactionService_Create_NoAlertWhenNoBudgetSet(t *testing.T) {
	broker := &mockBroker{}
	txStore := &mockTxStore{sums: map[string]float64{"Food": 50}}
	budgetStore := &mockBudgetStore{err: domain.ErrNotFound}

	svc := service.NewTransactionService(txStore, budgetStore, broker)

	tx := &domain.Transaction{UserID: "u1", Category: "Food", Amount: 50, Date: time.Now()}
	_, err := svc.Create(context.Background(), tx)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Give the goroutine a moment to (not) run; nothing to wait on
	// deterministically since there's no budget to trigger a publish.
	time.Sleep(50 * time.Millisecond)

	if len(broker.published) != 0 {
		t.Errorf("expected no alerts published, got %d", len(broker.published))
	}
}

func TestBudgetService_GetStatus(t *testing.T) {
	budget := &domain.Budget{ID: "b1", UserID: "u1", Category: "Food", LimitAmount: 200}
	txStore := &mockTxStore{sums: map[string]float64{"Food": 150}}
	budgetStore := &mockBudgetStore{budget: budget}

	svc := service.NewBudgetService(budgetStore, txStore)

	statuses, err := svc.GetStatus(context.Background(), "u1", time.Now())
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}

	got := statuses[0]
	if !almostEqual(got.Spent, 150, 1e-9) {
		t.Errorf("Spent = %.2f, want 150", got.Spent)
	}
	if !almostEqual(got.Remaining, 50, 1e-9) {
		t.Errorf("Remaining = %.2f, want 50", got.Remaining)
	}
	if !almostEqual(got.Percentage, 75, 1e-9) {
		t.Errorf("Percentage = %.2f, want 75", got.Percentage)
	}
}
