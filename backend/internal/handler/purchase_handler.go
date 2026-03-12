package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/saas/city-stories-guide/backend/internal/domain"
	"github.com/saas/city-stories-guide/backend/internal/service"
)

// PurchaseService defines the interface for purchase business logic.
type PurchaseService interface {
	VerifyAndCreate(ctx context.Context, req *service.VerifyPurchaseRequest) (*domain.Purchase, error)
	GetStatus(ctx context.Context, userID string) (*service.PurchaseStatus, error)
}

// PurchaseHandler handles purchase-related endpoints.
type PurchaseHandler struct {
	svc PurchaseService
}

// NewPurchaseHandler creates a new PurchaseHandler.
func NewPurchaseHandler(svc PurchaseService) *PurchaseHandler {
	return &PurchaseHandler{svc: svc}
}

type verifyPurchaseRequest struct {
	Platform      string              `json:"platform" binding:"required"`
	TransactionID string              `json:"transaction_id" binding:"required"`
	Receipt       string              `json:"receipt" binding:"required"`
	Type          domain.PurchaseType `json:"type" binding:"required"`
	CityID        *int                `json:"city_id"`
	Price         float64             `json:"price" binding:"required,gt=0"`
}

// VerifyPurchase handles POST /api/v1/purchases/verify.
// Verifies an IAP receipt and creates a purchase record.
func (h *PurchaseHandler) VerifyPurchase(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		errorJSON(c, http.StatusUnauthorized, "user_id not found in context")
		return
	}

	var req verifyPurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrorResponse(c, err)
		return
	}

	svcReq := &service.VerifyPurchaseRequest{
		UserID:        userID.(string),
		Platform:      req.Platform,
		TransactionID: req.TransactionID,
		Receipt:       req.Receipt,
		Type:          req.Type,
		CityID:        req.CityID,
		Price:         req.Price,
	}

	purchase, err := h.svc.VerifyAndCreate(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateTransaction) {
			errorJSON(c, http.StatusConflict, "transaction already processed")
			return
		}
		if errors.Is(err, service.ErrInvalidReceipt) {
			errorJSON(c, http.StatusBadRequest, "invalid receipt data")
			return
		}
		errorJSON(c, http.StatusInternalServerError, "failed to verify purchase")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": purchase})
}

// GetStatus handles GET /api/v1/purchases/status.
// Returns the current purchase/access status for the authenticated user.
func (h *PurchaseHandler) GetStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		errorJSON(c, http.StatusUnauthorized, "user_id not found in context")
		return
	}

	status, err := h.svc.GetStatus(c.Request.Context(), userID.(string))
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to get purchase status")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": status})
}
