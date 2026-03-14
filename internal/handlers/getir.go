package handlers

import (
	"log"
	"net/http"
	"strconv"

	getirSvc "go-entegra/internal/services/getir"
	"go-entegra/internal/models"
	"go-entegra/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetirHandler struct {
	db      *gorm.DB
	webhook *services.WebhookService
}

func NewGetirHandler(db *gorm.DB, webhook *services.WebhookService) *GetirHandler {
	return &GetirHandler{db: db, webhook: webhook}
}

// findGetirRP restaurant_slug ile aktif Getir entegrasyonunu bulur ve x-api-key doğrular
func (h *GetirHandler) findGetirRP(c *gin.Context, restaurantSlug string) (*models.RestaurantProvider, bool) {
	var rp models.RestaurantProvider
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = restaurant_providers.restaurant_id").
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Where("providers.slug = 'getir' AND restaurants.slug = ? AND restaurant_providers.status = '1'", restaurantSlug).
		Preload("Restaurant.Business").
		Preload("Provider").
		First(&rp).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Getir entegrasyonu bulunamadı"})
		return nil, false
	}

	// x-api-key doğrulama (information'da webhookApiKey varsa kontrol et)
	if apiKey, ok := rp.Information["webhookApiKey"].(string); ok && apiKey != "" {
		if c.GetHeader("x-api-key") != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Geçersiz API key"})
			return nil, false
		}
	}

	return &rp, true
}

// IncomingOrder Getir'den gelen sipariş webhook'unu işler
// POST /webhook/getir/:restaurant_slug
func (h *GetirHandler) IncomingOrder(c *gin.Context) {
	restaurantSlug := c.Param("restaurant_slug")

	rp, ok := h.findGetirRP(c, restaurantSlug)
	if !ok {
		return
	}

	var payload getirSvc.IncomingOrder
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz payload"})
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

	status := models.OrderStatusPending
	if active && rp.AutoApprove == "1" {
		status = models.OrderStatusApproved
	}

	order := models.Order{
		RestaurantID:         rp.RestaurantID,
		RestaurantProviderID: rp.ID,
		ProviderOrderID:      payload.ID,
		Status:               status,
		Active:               active,
		RawPayload:           models.JSONMap(map[string]interface{}{"order": payload}),
		TotalAmount:          payload.TotalPrice,
		DiscountAmount:       payload.DiscountAmount,
		CustomerName:         payload.Client.Name,
		CustomerPhone:        payload.Client.PhoneNumber,
		CustomerAddress:      payload.Address.Description,
		Note:                 payload.Note,
		DeliveryType:         payload.DeliveryType,
		PaymentMethod:        payload.PaymentMethod,
		VerificationCode:     payload.VerificationCode,
		ScheduledAt:          payload.ScheduledAt,
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sipariş kaydedilemedi"})
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

	// Aktif sipariş ise: otomatik onay ve webhook
	if active {
		if rp.AutoApprove == "1" && payload.ID != "" {
			go func() {
				client, err := getirSvc.NewClientFromInfo(map[string]interface{}(rp.Information), rp.Service)
				if err != nil {
					log.Printf("Getir client oluşturulamadı: %v", err)
					return
				}
				if err := client.ApproveOrder(payload.ID); err != nil {
					log.Printf("Getir otomatik onay başarısız (order: %s): %v", payload.ID, err)
				}
			}()
		}

		if rp.Restaurant.WebhookURL != "" {
			go h.webhook.SendOrder(&order, rp.Restaurant.WebhookURL)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş alındı", "order_id": order.ID})
}

// ApproveOrder panelden manuel sipariş onayı (Getir API'sine bildirir)
// POST /api/orders/:id/getir/approve
func (h *GetirHandler) ApproveOrder(c *gin.Context) {
	orderID := c.Param("id")

	var order models.Order
	if err := h.db.Preload("RestaurantProvider").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information), order.RestaurantProvider.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.ApproveOrder(order.ProviderOrderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusApproved
	h.db.Save(&order)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş Getir'de onaylandı", "data": order})
}

// PrepareOrder siparişi hazırlamaya başlar
// POST /api/orders/:id/getir/prepare
func (h *GetirHandler) PrepareOrder(c *gin.Context) {
	orderID := c.Param("id")

	var order models.Order
	if err := h.db.Preload("RestaurantProvider").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information), order.RestaurantProvider.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.PrepareOrder(order.ProviderOrderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusPreparing
	h.db.Save(&order)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş hazırlanıyor", "data": order})
}

// HandoverOrder siparişi kuryeye teslim eder (Getir Getirsin / deliveryType: 1)
// POST /api/orders/:id/getir/handover
func (h *GetirHandler) HandoverOrder(c *gin.Context) {
	orderID := c.Param("id")

	var order models.Order
	if err := h.db.Preload("RestaurantProvider").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information), order.RestaurantProvider.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.HandoverOrder(order.ProviderOrderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusReady
	h.db.Save(&order)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş kuryeye teslim edildi", "data": order})
}

// DeliverOrder siparişi müşteriye teslim eder (Restoran Getirsin / deliveryType: 2)
// POST /api/orders/:id/getir/deliver
func (h *GetirHandler) DeliverOrder(c *gin.Context) {
	orderID := c.Param("id")

	var order models.Order
	if err := h.db.Preload("RestaurantProvider").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information), order.RestaurantProvider.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.DeliverOrder(order.ProviderOrderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusDelivered
	h.db.Save(&order)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş müşteriye teslim edildi", "data": order})
}

// CancelOrder panelden manuel sipariş iptali
// POST /api/orders/:id/getir/cancel
func (h *GetirHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")

	var order models.Order
	if err := h.db.Preload("RestaurantProvider").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	var req struct {
		ReasonID int    `json:"reason_id"`
		Note     string `json:"note"`
	}
	c.ShouldBindJSON(&req)

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information), order.RestaurantProvider.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.CancelOrder(order.ProviderOrderID, req.ReasonID, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusCancelled
	h.db.Save(&order)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş Getir'de iptal edildi", "data": order})
}

// SetStatus restoran Getir'de açık/kapalı yapar
// PUT /api/restaurants/:id/getir/status
func (h *GetirHandler) SetStatus(c *gin.Context) {
	restaurantID := c.Param("id")

	var rp models.RestaurantProvider
	err := h.db.
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Where("restaurant_providers.restaurant_id = ? AND providers.slug = 'getir'", restaurantID).
		First(&rp).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Getir entegrasyonu bulunamadı"})
		return
	}

	var req struct {
		IsOpen bool `json:"is_open"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(rp.Information), rp.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.SetRestaurantStatus(req.IsOpen); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Getir restoran durumu güncellendi"})
}

// IncomingStatusChange Getir'den gelen sipariş durum değişikliği webhook'unu işler
// POST /webhook/getir/:restaurant_slug/status
func (h *GetirHandler) IncomingStatusChange(c *gin.Context) {
	restaurantSlug := c.Param("restaurant_slug")

	rp, ok := h.findGetirRP(c, restaurantSlug)
	if !ok {
		return
	}

	var payload getirSvc.IncomingStatusChange
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz payload"})
		return
	}

	var order models.Order
	if err := h.db.Where("provider_order_id = ? AND restaurant_provider_id = ?", payload.ID, rp.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	switch payload.Status {
	case "Cancelled":
		order.Status = models.OrderStatusCancelled
	case "Delivered":
		order.Status = models.OrderStatusDelivered
	case "Preparing":
		order.Status = models.OrderStatusPreparing
	}

	h.db.Save(&order)
	c.JSON(http.StatusOK, gin.H{"message": "Statü güncellendi"})
}

// SetPosStatus restoranın POS entegrasyonunu açar/kapatır
// PUT /api/getir/:integration_id/pos-status
func (h *GetirHandler) SetPosStatus(c *gin.Context) {
	integrationID := c.Param("integration_id")

	var rp models.RestaurantProvider
	if err := h.db.First(&rp, integrationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entegrasyon bulunamadı"})
		return
	}

	var req struct {
		PosStatus int `json:"pos_status"` // 100=aktif, 200=pasif
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.PosStatus != 100 && req.PosStatus != 200) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pos_status alanı gereklidir (100=aktif, 200=pasif)"})
		return
	}

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(rp.Information), rp.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.SetPosStatus(req.PosStatus == 100); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	label := "pasif"
	if req.PosStatus == 100 {
		label = "aktif"
	}
	c.JSON(http.StatusOK, gin.H{"message": "Getir POS durumu " + label + " yapıldı"})
}

// GetCancelOptions siparişin iptal nedenlerini getirir
// GET /api/orders/:id/getir/cancel-options
func (h *GetirHandler) GetCancelOptions(c *gin.Context) {
	orderID := c.Param("id")

	var order models.Order
	if err := h.db.Preload("RestaurantProvider").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information), order.RestaurantProvider.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	options, err := client.GetCancelOptions(order.ProviderOrderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": options})
}

// ApproveScheduledOrder ileri tarihli siparişi onaylar
// POST /api/orders/:id/getir/approve-scheduled
func (h *GetirHandler) ApproveScheduledOrder(c *gin.Context) {
	orderID := c.Param("id")

	var order models.Order
	if err := h.db.Preload("RestaurantProvider").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information), order.RestaurantProvider.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.ApproveScheduledOrder(order.ProviderOrderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusApproved
	h.db.Save(&order)

	c.JSON(http.StatusOK, gin.H{"message": "İleri tarihli sipariş onaylandı", "data": order})
}

// InquireOrder Getir'den sipariş detaylarını sorgular
// GET /api/orders/:id/getir/inquiry
func (h *GetirHandler) InquireOrder(c *gin.Context) {
	orderID := c.Param("id")

	var order models.Order
	if err := h.db.Preload("RestaurantProvider").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information), order.RestaurantProvider.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := client.InquireOrder(order.ProviderOrderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
