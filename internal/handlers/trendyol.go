package handlers

import (
	"net/http"

	"go-entegra/internal/middleware"
	"go-entegra/internal/models"
	trendyolSvc "go-entegra/internal/services/trendyol"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TrendyolHandler struct {
	db *gorm.DB
}

func NewTrendyolHandler(db *gorm.DB) *TrendyolHandler {
	return &TrendyolHandler{db: db}
}

// getOrderAndClient sipariş + Trendyol client'ı döner
func (h *TrendyolHandler) getOrderAndClient(c *gin.Context) (*models.Order, *trendyolSvc.Client, bool) {
	businessID := middleware.GetBusinessID(c)
	orderID := c.Param("id")

	var order models.Order
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = orders.restaurant_id").
		Where("orders.id = ? AND restaurants.business_id = ?", orderID, businessID).
		Preload("RestaurantProvider").
		First(&order).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return nil, nil, false
	}

	client, err := trendyolSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil, false
	}

	return &order, client, true
}

// ApproveOrder siparişi Trendyol'da onaylar (Verified)
// POST /api/orders/:id/trendyol/approve
func (h *TrendyolHandler) ApproveOrder(c *gin.Context) {
	order, client, ok := h.getOrderAndClient(c)
	if !ok {
		return
	}

	if err := client.UpdateOrderStatus(order.ProviderOrderID, "Verified"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusApproved
	h.db.Save(order)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş Trendyol'da onaylandı", "data": order})
}

// PrepareOrder siparişi hazırlamaya başlar
// POST /api/orders/:id/trendyol/prepare
func (h *TrendyolHandler) PrepareOrder(c *gin.Context) {
	order, client, ok := h.getOrderAndClient(c)
	if !ok {
		return
	}

	if err := client.UpdateOrderStatus(order.ProviderOrderID, "Preparing"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusPreparing
	h.db.Save(order)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş hazırlanıyor", "data": order})
}

// DeliverOrder siparişi teslim edildi olarak işaretler
// POST /api/orders/:id/trendyol/deliver
func (h *TrendyolHandler) DeliverOrder(c *gin.Context) {
	order, client, ok := h.getOrderAndClient(c)
	if !ok {
		return
	}

	if err := client.UpdateOrderStatus(order.ProviderOrderID, "Delivered"); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusDelivered
	h.db.Save(order)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş teslim edildi", "data": order})
}

// CancelOrder siparişi iptal eder
// POST /api/orders/:id/trendyol/cancel
func (h *TrendyolHandler) CancelOrder(c *gin.Context) {
	order, client, ok := h.getOrderAndClient(c)
	if !ok {
		return
	}

	var req struct {
		ReasonID int    `json:"reason_id"`
		Note     string `json:"note"`
	}
	c.ShouldBindJSON(&req)

	if err := client.CancelOrder(order.ProviderOrderID, req.ReasonID, req.Note); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusCancelled
	h.db.Save(order)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş iptal edildi", "data": order})
}
