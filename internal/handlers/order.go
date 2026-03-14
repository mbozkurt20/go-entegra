package handlers

import (
	"net/http"
	"strconv"

	"go-entegra/internal/middleware"
	"go-entegra/internal/models"
	"go-entegra/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	db      *gorm.DB
	webhook *services.WebhookService
}

func NewOrderHandler(db *gorm.DB, webhook *services.WebhookService) *OrderHandler {
	return &OrderHandler{db: db, webhook: webhook}
}

// List godoc
// GET /orders
func (h *OrderHandler) List(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)

	query := h.db.
		Joins("JOIN restaurants ON restaurants.id = orders.restaurant_id").
		Where("restaurants.business_id = ? AND orders.active = true", businessID).
		Preload("Restaurant").
		Preload("RestaurantProvider.Provider")

	if status := c.Query("status"); status != "" {
		query = query.Where("orders.status = ?", status)
	}

	var orders []models.Order
	query.Order("orders.created_at DESC").Find(&orders)

	c.JSON(http.StatusOK, gin.H{"data": orders})
}

// ListByRestaurant godoc
// GET /restaurants/:id/orders
func (h *OrderHandler) ListByRestaurant(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	restaurantID, _ := strconv.Atoi(c.Param("id"))

	query := h.db.
		Joins("JOIN restaurants ON restaurants.id = orders.restaurant_id").
		Where("restaurants.business_id = ? AND orders.restaurant_id = ? AND orders.active = true", businessID, restaurantID).
		Preload("Restaurant").
		Preload("RestaurantProvider.Provider")

	if status := c.Query("status"); status != "" {
		query = query.Where("orders.status = ?", status)
	}

	var orders []models.Order
	query.Order("orders.created_at DESC").Find(&orders)

	c.JSON(http.StatusOK, gin.H{"data": orders})
}

// Get godoc
// GET /orders/:id
func (h *OrderHandler) Get(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var order models.Order
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = orders.restaurant_id").
		Where("orders.id = ? AND restaurants.business_id = ? AND orders.active = true", id, businessID).
		Preload("Restaurant").
		Preload("RestaurantProvider.Provider").
		First(&order).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": order})
}

// UpdateStatus godoc
// PATCH /orders/:id/status
func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var order models.Order
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = orders.restaurant_id").
		Where("orders.id = ? AND restaurants.business_id = ?", id, businessID).
		Preload("Restaurant").
		First(&order).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatus(req.Status)
	h.db.Save(&order)

	c.JSON(http.StatusOK, gin.H{"data": order})
}

// IncomingOrder handles webhook from marketplace providers
// POST /webhook/:provider_slug/:restaurant_slug
func (h *OrderHandler) IncomingOrder(c *gin.Context) {
	providerSlug := c.Param("provider_slug")
	restaurantSlug := c.Param("restaurant_slug")

	var rp models.RestaurantProvider
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = restaurant_providers.restaurant_id").
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Where("providers.slug = ? AND restaurants.slug = ? AND restaurant_providers.status = ?",
			providerSlug, restaurantSlug, "active").
		Preload("Restaurant.Business").
		Preload("Provider").
		First(&rp).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	var rawPayload map[string]interface{}
	if err := c.ShouldBindJSON(&rawPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	// Kontör kontrolü: is_super değilse ve kontör 0 ise sipariş pasif kaydedilir
	active := true
	newBalance := rp.Restaurant.Credits
	if !rp.Restaurant.Business.IsSuper {
		if rp.Restaurant.Credits <= 0 {
			active = false
		} else {
			h.db.Model(&rp.Restaurant).UpdateColumn("credits", gorm.Expr("credits - 1"))
			newBalance = rp.Restaurant.Credits - 1
		}
	}

	order := models.Order{
		RestaurantID:         rp.RestaurantID,
		RestaurantProviderID: rp.ID,
		Status:               models.OrderStatusPending,
		RawPayload:           models.JSONMap(rawPayload),
		Active:               active,
	}

	if active && rp.AutoApprove == "1" {
		order.Status = models.OrderStatusApproved
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save order"})
		return
	}

	// Kontör tüketim hareketi
	if active && !rp.Restaurant.Business.IsSuper {
		orderID := order.ID
		h.db.Create(&models.CreditTransaction{
			RestaurantID: rp.RestaurantID,
			Amount:       -1,
			Balance:      newBalance,
			Type:         models.CreditTxUse,
			Source:       models.CreditSrcOrder,
			Description:  "Sipariş #" + strconv.Itoa(int(order.ID)),
			OrderID:      &orderID,
		})
	}

	if active && rp.Restaurant.WebhookURL != "" {
		go h.webhook.SendOrder(&order, rp.Restaurant.WebhookURL)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order received", "order_id": order.ID})
}
