package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go-entegra/internal/middleware"
	"go-entegra/internal/models"
	migrosSvc "go-entegra/internal/services/migros"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MigrosHandler struct {
	db *gorm.DB
}

func NewMigrosHandler(db *gorm.DB) *MigrosHandler {
	return &MigrosHandler{db: db}
}

func (h *MigrosHandler) getIntegration(c *gin.Context) (*models.RestaurantProvider, bool) {
	businessID := middleware.GetBusinessID(c)
	integrationID, _ := strconv.Atoi(c.Param("integration_id"))

	var rp models.RestaurantProvider
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = restaurant_providers.restaurant_id").
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Where("restaurant_providers.id = ? AND restaurants.business_id = ? AND providers.slug = 'migros'",
			integrationID, businessID).
		First(&rp).Error

	if err != nil {
		var anyRp models.RestaurantProvider
		if h.db.Where("restaurant_providers.id = ?", integrationID).First(&anyRp).Error == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bu entegrasyon Migros'a ait değil"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entegrasyon bulunamadı"})
		}
		return nil, false
	}
	return &rp, true
}

func (h *MigrosHandler) newClient(c *gin.Context, rp *models.RestaurantProvider) (*migrosSvc.Client, bool) {
	client, err := migrosSvc.NewClientFromInfo(map[string]interface{}(rp.Information))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, false
	}
	return client, true
}

// SetStatus mağazayı Migros'ta açar/kapatır
// PUT /api/migros/:integration_id/status
func (h *MigrosHandler) SetStatus(c *gin.Context) {
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

	if err := client.SetStoreStatus(req.IsOpen); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	status := "kapatıldı"
	if req.IsOpen {
		status = "açıldı"
	}
	c.JSON(http.StatusOK, gin.H{"message": "Mağaza Migros'ta " + status})
}

// GetMenu menüyü Migros'tan çeker
// GET /api/migros/:integration_id/menu
func (h *MigrosHandler) GetMenu(c *gin.Context) {
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
// PUT /api/migros/:integration_id/products/:product_id/status
// product_id formatı: "menuItemId_productId"
func (h *MigrosHandler) UpdateProductStatus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	_, productID, err := parseCompoundID(c.Param("product_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz ürün ID: " + err.Error()})
		return
	}

	var req struct {
		IsAvailable bool `json:"is_available"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.UpdateProductStatus(productID, req.IsAvailable); err != nil {
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
// PUT /api/migros/:integration_id/products/:product_id/price
// product_id formatı: "menuItemId_productId"
func (h *MigrosHandler) UpdateProductPrice(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	menuItemID, productID, err := parseCompoundID(c.Param("product_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz ürün ID: " + err.Error()})
		return
	}

	var req struct {
		Price float64 `json:"price" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.UpdateProductPrice(menuItemID, productID, req.Price); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fiyat güncellendi", "price": req.Price})
}

// parseCompoundID "menuItemId_productId" formatını ayrıştırır
func parseCompoundID(id string) (menuItemID, productID int64, err error) {
	parts := strings.SplitN(id, "_", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("beklenen format: menuItemId_productId")
	}
	menuItemID, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("menuItemId geçersiz")
	}
	productID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("productId geçersiz")
	}
	return
}
