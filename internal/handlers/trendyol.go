package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"go-entegra/internal/middleware"
	"go-entegra/internal/models"
	"go-entegra/internal/services"
	trendyolSvc "go-entegra/internal/services/trendyol"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TrendyolHandler struct {
	db      *gorm.DB
	webhook *services.WebhookService
}

func NewTrendyolHandler(db *gorm.DB, webhook *services.WebhookService) *TrendyolHandler {
	return &TrendyolHandler{db: db, webhook: webhook}
}

// IncomingOrder Trendyol'dan gelen sipariş webhook'unu işler
// POST /webhook/trendyol/:restaurant_slug
func (h *TrendyolHandler) IncomingOrder(c *gin.Context) {
	restaurantSlug := c.Param("restaurant_slug")
	log.Printf("[TRENDYOL] [WEBHOOK] Yeni sipariş isteği | restaurant=%s | ip=%s", restaurantSlug, c.ClientIP())

	var rp models.RestaurantProvider
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = restaurant_providers.restaurant_id").
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Where("providers.slug = 'trendyol' AND restaurants.slug = ? AND restaurant_providers.status = '1'", restaurantSlug).
		Preload("Restaurant.Business").
		Preload("Provider").
		First(&rp).Error

	if err != nil {
		log.Printf("[TRENDYOL] [WEBHOOK] Entegrasyon bulunamadı | restaurant=%s", restaurantSlug)
		c.JSON(http.StatusNotFound, gin.H{"error": "Trendyol entegrasyonu bulunamadı"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "İstek okunamadı"})
		return
	}

	var order trendyolSvc.WebhookOrder
	if err := json.Unmarshal(body, &order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz sipariş payload"})
		return
	}

	// Kontör kontrolü
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

	dbOrder := models.Order{
		RestaurantID:         rp.RestaurantID,
		RestaurantProviderID: rp.ID,
		ProviderOrderID:      order.ID,
		Status:               status,
		Active:               active,
		RawPayload:           models.JSONMap(map[string]interface{}{"order": order}),
		TotalAmount:          order.TotalAmount,
		CustomerName:         order.Customer.FirstName + " " + order.Customer.LastName,
		CustomerPhone:        order.Customer.PhoneNumber,
		CustomerAddress:      order.Customer.DeliveryAddress,
		Note:                 order.Note,
		PaymentMethod:        order.PaymentType,
	}

	if err := h.db.Create(&dbOrder).Error; err != nil {
		log.Printf("[TRENDYOL] [WEBHOOK] Sipariş kaydedilemedi | restaurant=%s | provider_order_id=%s | hata=%v", restaurantSlug, order.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sipariş kaydedilemedi"})
		return
	}

	log.Printf("[TRENDYOL] [WEBHOOK] Sipariş kaydedildi | order_id=%d | provider_order_id=%s | restaurant=%s | müşteri=%s | tutar=%.2f₺ | durum=%s | aktif=%v",
		dbOrder.ID, order.ID, restaurantSlug, dbOrder.CustomerName, dbOrder.TotalAmount, dbOrder.Status, active)

	// Kontör hareketi
	if active && !rp.Restaurant.Business.IsSuper {
		orderID := dbOrder.ID
		h.db.Create(&models.CreditTransaction{
			RestaurantID: rp.RestaurantID,
			Amount:       -1,
			Balance:      newBalance,
			Type:         models.CreditTxUse,
			Source:       models.CreditSrcOrder,
			Description:  "Sipariş #" + strconv.Itoa(int(dbOrder.ID)),
			OrderID:      &orderID,
		})
	}

	// Otomatik onay
	if active && rp.AutoApprove == "1" && order.ID != "" {
		go func() {
			client, err := trendyolSvc.NewClientFromInfo(map[string]interface{}(rp.Information))
			if err != nil {
				log.Printf("Trendyol client oluşturulamadı: %v", err)
				return
			}
			if err := client.UpdateOrderStatus(order.ID, "Verified"); err != nil {
				log.Printf("Trendyol otomatik onay başarısız (order: %s): %v", order.ID, err)
			}
		}()
	}

	if active && rp.Restaurant.WebhookURL != "" {
		go h.webhook.SendOrder(&dbOrder, rp.Restaurant.WebhookURL)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş alındı", "order_id": dbOrder.ID})
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
		log.Printf("[TRENDYOL] [ONAY] API hatası | order_id=%d | hata=%v", order.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusApproved
	h.db.Save(order)
	log.Printf("[TRENDYOL] [ONAY] Sipariş onaylandı | order_id=%d | provider_order_id=%s", order.ID, order.ProviderOrderID)

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
		log.Printf("[TRENDYOL] [HAZIRLAMA] API hatası | order_id=%d | hata=%v", order.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusPreparing
	h.db.Save(order)
	log.Printf("[TRENDYOL] [HAZIRLAMA] Sipariş hazırlanıyor | order_id=%d | provider_order_id=%s", order.ID, order.ProviderOrderID)

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
		log.Printf("[TRENDYOL] [TESLİMAT] API hatası | order_id=%d | hata=%v", order.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusDelivered
	h.db.Save(order)
	log.Printf("[TRENDYOL] [TESLİMAT] Sipariş teslim edildi | order_id=%d | provider_order_id=%s", order.ID, order.ProviderOrderID)

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
		log.Printf("[TRENDYOL] [İPTAL] API hatası | order_id=%d | hata=%v", order.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusCancelled
	h.db.Save(order)
	log.Printf("[TRENDYOL] [İPTAL] Sipariş iptal edildi | order_id=%d | provider_order_id=%s | neden_id=%d", order.ID, order.ProviderOrderID, req.ReasonID)

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş iptal edildi", "data": order})
}
