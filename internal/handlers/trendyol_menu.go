package handlers

import (
	"net/http"
	"strconv"

	"go-entegra/internal/middleware"
	"go-entegra/internal/models"
	trendyolSvc "go-entegra/internal/services/trendyol"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TrendyolMenuHandler struct {
	db *gorm.DB
}

func NewTrendyolMenuHandler(db *gorm.DB) *TrendyolMenuHandler {
	return &TrendyolMenuHandler{db: db}
}

func (h *TrendyolMenuHandler) getIntegration(c *gin.Context) (*models.RestaurantProvider, bool) {
	businessID := middleware.GetBusinessID(c)
	integrationID, _ := strconv.Atoi(c.Param("integration_id"))

	var rp models.RestaurantProvider
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = restaurant_providers.restaurant_id").
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Where("restaurant_providers.id = ? AND restaurants.business_id = ? AND providers.slug = 'trendyol'",
			integrationID, businessID).
		First(&rp).Error

	if err != nil {
		// Entegrasyon hiç yoksa veya provider Trendyol değilse
		var anyRp models.RestaurantProvider
		if h.db.Where("restaurant_providers.id = ?", integrationID).First(&anyRp).Error == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bu entegrasyon Trendyol'a ait değil"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entegrasyon bulunamadı"})
		}
		return nil, false
	}
	return &rp, true
}

func (h *TrendyolMenuHandler) newClient(c *gin.Context, rp *models.RestaurantProvider) (*trendyolSvc.Client, bool) {
	client, err := trendyolSvc.NewClientFromInfo(map[string]interface{}(rp.Information))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, false
	}
	return client, true
}

// SetStatus restoranı Trendyol'da açar/kapatır
// PUT /api/trendyol/:integration_id/status
func (h *TrendyolMenuHandler) SetStatus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	var req struct {
		IsOpen bool `json:"is_open"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.SetRestaurantStatus(req.IsOpen); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	status := "kapatıldı"
	if req.IsOpen {
		status = "açıldı"
	}
	c.JSON(http.StatusOK, gin.H{"message": "Restoran Trendyol'da " + status})
}

// GetMenu menüyü Trendyol'dan çeker
// GET /api/trendyol/:integration_id/menu
func (h *TrendyolMenuHandler) GetMenu(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	menu, err := client.GetMenu()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": menu})
}

// UpdateProductStatus ürünü aktif/pasif yapar
// PUT /api/trendyol/:integration_id/products/:product_id/status
func (h *TrendyolMenuHandler) UpdateProductStatus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	productID := c.Param("product_id")

	var req struct {
		IsAvailable bool `json:"is_available"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.UpdateItemStatus(productID, req.IsAvailable); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	status := "pasif"
	if req.IsAvailable {
		status = "aktif"
	}
	c.JSON(http.StatusOK, gin.H{"message": "Ürün " + status + " yapıldı"})
}

// UpdateProductPrice ürün fiyatını günceller
// PUT /api/trendyol/:integration_id/products/:product_id/price
func (h *TrendyolMenuHandler) UpdateProductPrice(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	productID := c.Param("product_id")

	var req struct {
		Price float64 `json:"price" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.UpdateItemPrice(productID, req.Price); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fiyat güncellendi", "price": req.Price})
}
