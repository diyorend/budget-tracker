package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/diyorend/budget-tracker/internal/domain"
	"github.com/diyorend/budget-tracker/internal/middleware"
	"github.com/diyorend/budget-tracker/internal/service"
	"github.com/labstack/echo/v4"
)

type TransactionHandler struct {
	txSvc *service.TransactionService
}

func NewTransactionHandler(txSvc *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{txSvc: txSvc}
}

type createTransactionRequest struct {
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"` // "2026-01-15"
}

func (h *TransactionHandler) Create(c echo.Context) error {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		return err
	}

	var req createTransactionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}

	if req.Amount <= 0 || req.Category == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "amount > 0 and category required"})
	}

	date := time.Now()
	if req.Date != "" {
		date, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "date format: 2006-01-02"})
		}
	}

	tx := &domain.Transaction{
		UserID:      userID,
		Amount:      req.Amount,
		Category:    req.Category,
		Description: req.Description,
		Date:        date,
	}

	created, err := h.txSvc.Create(c.Request().Context(), tx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create transaction"})
	}

	return c.JSON(http.StatusCreated, created)
}

func (h *TransactionHandler) List(c echo.Context) error {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	txs, err := h.txSvc.List(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list transactions"})
	}
	if txs == nil {
		txs = []*domain.Transaction{} // return [] not null
	}
	return c.JSON(http.StatusOK, txs)
}
