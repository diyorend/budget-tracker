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
	Category	string	`json:"category"`
	LimitAmount	float64	`json:"limit_amount"`
	Month		string	`json:"month"` // "2026-01"
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
		month, err = time.Parse("2026-01", req.Month)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "month format: 2006-01"})
		}
	}

	budget := &domain.Budget{
		UserID:		userID,
		Category:	req.Category,
		LimitAmount:	req.LimitAmount,
		Month:		month,
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
		return nil
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

