package handler

import (
	"net/http"
	"time"

	"github.com/diyorend/budget-tracker/internal/domain"
	"github.com/diyorend/budget-tracker/internal/middleware"
	"github.com/diyorend/budget-tracker/internal/service"
	"github.com/labstack/echo/v4"
)

type BudgetHandler struct {
	budgetSvc *service.BudgetService
}

func NewBudgetHandler(budgetSvc *service.BudgetService) *BudgetHandler {
	return &BudgetHandler{budgetSvc: budgetSvc}
}

type upsertBudgetRequest struct {
	Category    string  `json:"category"`
	LimitAmount float64 `json:"limit_amount"`
	Month       string  `json:"month"` // "2026-01"
}

func (h *BudgetHandler) Upsert(c echo.Context) error {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		return err
	}

	var req upsertBudgetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}

	month := time.Now()
	if req.Month != "" {
		// Go's reference layout is always "2006-01-02 15:04:05" — the
		// numbers are fixed magic values, not a description of this
		// field's format. Using "2026-01" here would silently fail to
		// parse anything (or parse garbage), it's not "this year" plus
		// month. The correct layout token for YYYY-MM is "2006-01".
		month, err = time.Parse("2006-01", req.Month)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "month format: 2006-01"})
		}
	}

	if req.Category == "" || req.LimitAmount <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "category and limit_amount > 0 required"})
	}

	budget := &domain.Budget{
		UserID:      userID,
		Category:    req.Category,
		LimitAmount: req.LimitAmount,
		Month:       month,
	}

	created, err := h.budgetSvc.Upsert(c.Request().Context(), budget)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save budget"})
	}
	return c.JSON(http.StatusOK, created)
}

func (h *BudgetHandler) GetStatus(c echo.Context) error {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		// Bug fixed: this used to `return nil`, which discards the 401
		// RequireAuth already wrote and lets Echo continue as if nothing
		// went wrong (sending a second, empty 200 response on top of it).
		return err
	}

	monthStr := c.QueryParam("month")
	month := time.Now()
	if monthStr != "" {
		month, err = time.Parse("2006-01", monthStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "month format: 2006-01"})
		}
	}

	statuses, err := h.budgetSvc.GetStatus(c.Request().Context(), userID, month)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get budget status"})
	}
	if statuses == nil {
		statuses = []*domain.BudgetStatus{}
	}
	return c.JSON(http.StatusOK, statuses)
}
