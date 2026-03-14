package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"go-entegra/internal/middleware"
	"go-entegra/internal/models"
	"go-entegra/internal/services"
	migrosSvc "go-entegra/internal/services/migros"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MigrosHandler struct {
	db      *gorm.DB
	webhook *services.WebhookService
}

func NewMigrosHandler(db *gorm.DB, webhook *services.WebhookService) *MigrosHandler {
	return &MigrosHandler{db: db, webhook: webhook}
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

// GetInfo mağaza bilgisini döner
// GET /api/migros/:integration_id/info
func (h *MigrosHandler) GetInfo(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	info, err := client.GetStoreDetail()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": info})
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

// IncomingOrder Migros'tan gelen sipariş webhook'unu işler
// POST /webhook/migros/:restaurant_slug
func (h *MigrosHandler) IncomingOrder(c *gin.Context) {
	restaurantSlug := c.Param("restaurant_slug")
	log.Printf("[MİGROS] [WEBHOOK] Yeni sipariş isteği | restaurant=%s | ip=%s", restaurantSlug, c.ClientIP())

	var rp models.RestaurantProvider
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = restaurant_providers.restaurant_id").
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Where("providers.slug = 'migros' AND restaurants.slug = ? AND restaurant_providers.status = '1'", restaurantSlug).
		Preload("Restaurant.Business").
		Preload("Provider").
		First(&rp).Error

	if err != nil {
		log.Printf("[MİGROS] [WEBHOOK] Entegrasyon bulunamadı | restaurant=%s", restaurantSlug)
		c.JSON(http.StatusNotFound, gin.H{"error": "Migros entegrasyonu bulunamadı"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "İstek okunamadı"})
		return
	}

	var order migrosSvc.IncomingOrder
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
		ProviderOrderID:      strconv.FormatInt(order.ID, 10),
		Status:               status,
		Active:               active,
		RawPayload:           models.JSONMap(map[string]interface{}{"order": order}),
		TotalAmount:          float64(order.Prices.Discounted.AmountAsPenny) / 100,
		CustomerName:         order.Customer.FullName,
		CustomerPhone:        order.Customer.PhoneNumber,
		CustomerAddress:      order.Customer.DeliveryAddress.Detail,
		Note:                 order.ExtendedProps.OrderNote,
		PaymentMethod:        order.Payment.Type.Name,
	}

	if err := h.db.Create(&dbOrder).Error; err != nil {
		log.Printf("[MİGROS] [WEBHOOK] Sipariş kaydedilemedi | restaurant=%s | provider_order_id=%d | hata=%v", restaurantSlug, order.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sipariş kaydedilemedi"})
		return
	}

	log.Printf("[MİGROS] [WEBHOOK] Sipariş kaydedildi | order_id=%d | provider_order_id=%d | restaurant=%s | müşteri=%s | tutar=%.2f₺ | durum=%s | aktif=%v",
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
	if active && rp.AutoApprove == "1" && order.ID > 0 && order.Status == "NEW_PENDING" {
		go func() {
			client, err := migrosSvc.NewClientFromInfo(map[string]interface{}(rp.Information))
			if err != nil {
				log.Printf("Migros client oluşturulamadı: %v", err)
				return
			}
			if err := client.UpdateOrderStatus(order.ID, migrosSvc.OrderStatusApproved); err != nil {
				log.Printf("Migros otomatik onay başarısız (order: %d): %v", order.ID, err)
			}
		}()
	}

	if active && rp.Restaurant.WebhookURL != "" {
		go h.webhook.SendOrder(&dbOrder, rp.Restaurant.WebhookURL)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş alındı", "order_id": dbOrder.ID})
}

// getOrderAndMigrosClient sipariş + Migros client'ı döner
func (h *MigrosHandler) getOrderAndMigrosClient(c *gin.Context) (*models.Order, *migrosSvc.Client, bool) {
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

	client, err := migrosSvc.NewClientFromInfo(map[string]interface{}(order.RestaurantProvider.Information))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil, false
	}

	return &order, client, true
}

// ApproveOrder siparişi onaylar
// POST /api/orders/:id/migros/approve
func (h *MigrosHandler) ApproveOrder(c *gin.Context) {
	order, client, ok := h.getOrderAndMigrosClient(c)
	if !ok {
		return
	}

	orderID, _ := strconv.ParseInt(order.ProviderOrderID, 10, 64)
	if err := client.UpdateOrderStatus(orderID, migrosSvc.OrderStatusApproved); err != nil {
		log.Printf("[MİGROS] [ONAY] API hatası | order_id=%d | hata=%v", order.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusApproved
	h.db.Save(order)
	log.Printf("[MİGROS] [ONAY] Sipariş onaylandı | order_id=%d | provider_order_id=%s", order.ID, order.ProviderOrderID)
	c.JSON(http.StatusOK, gin.H{"message": "Sipariş onaylandı", "data": order})
}

// PrepareOrder siparişi hazırlamaya başlar
// POST /api/orders/:id/migros/prepare
func (h *MigrosHandler) PrepareOrder(c *gin.Context) {
	order, client, ok := h.getOrderAndMigrosClient(c)
	if !ok {
		return
	}

	orderID, _ := strconv.ParseInt(order.ProviderOrderID, 10, 64)
	if err := client.UpdateOrderStatus(orderID, migrosSvc.OrderStatusPrepared); err != nil {
		log.Printf("[MİGROS] [HAZIRLAMA] API hatası | order_id=%d | hata=%v", order.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusPreparing
	h.db.Save(order)
	log.Printf("[MİGROS] [HAZIRLAMA] Sipariş hazırlanıyor | order_id=%d | provider_order_id=%s", order.ID, order.ProviderOrderID)
	c.JSON(http.StatusOK, gin.H{"message": "Sipariş hazırlanıyor", "data": order})
}

// DeliverOrder siparişi teslim edildi olarak işaretler
// POST /api/orders/:id/migros/deliver
func (h *MigrosHandler) DeliverOrder(c *gin.Context) {
	order, client, ok := h.getOrderAndMigrosClient(c)
	if !ok {
		return
	}

	orderID, _ := strconv.ParseInt(order.ProviderOrderID, 10, 64)
	if err := client.UpdateOrderStatus(orderID, migrosSvc.OrderStatusDelivery); err != nil {
		log.Printf("[MİGROS] [TESLİMAT] API hatası | order_id=%d | hata=%v", order.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusDelivered
	h.db.Save(order)
	log.Printf("[MİGROS] [TESLİMAT] Sipariş teslim edildi | order_id=%d | provider_order_id=%s", order.ID, order.ProviderOrderID)
	c.JSON(http.StatusOK, gin.H{"message": "Sipariş teslim edildi", "data": order})
}

// CancelOrder siparişi iptal eder
// POST /api/orders/:id/migros/cancel
func (h *MigrosHandler) CancelOrder(c *gin.Context) {
	order, client, ok := h.getOrderAndMigrosClient(c)
	if !ok {
		return
	}

	var req struct {
		ReasonID   int64 `json:"reason_id"`
		NotifyUser bool  `json:"notify_user"`
	}
	c.ShouldBindJSON(&req)

	orderID, _ := strconv.ParseInt(order.ProviderOrderID, 10, 64)
	if err := client.CancelOrder(orderID, req.ReasonID, req.NotifyUser); err != nil {
		log.Printf("[MİGROS] [İPTAL] API hatası | order_id=%d | hata=%v", order.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatusCancelled
	h.db.Save(order)
	log.Printf("[MİGROS] [İPTAL] Sipariş iptal edildi | order_id=%d | provider_order_id=%s | neden_id=%d", order.ID, order.ProviderOrderID, req.ReasonID)
	c.JSON(http.StatusOK, gin.H{"message": "Sipariş iptal edildi", "data": order})
}

// GetCancelReasons Migros iptal sebeplerini döner
// GET /api/migros/:integration_id/cancel-reasons
func (h *MigrosHandler) GetCancelReasons(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}
	reasons, err := client.GetCancelReasons()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reasons})
}

// GetStoreViewStatus mağazanın geçici kapalılık durumunu döner
// GET /api/migros/:integration_id/view-status
func (h *MigrosHandler) GetStoreViewStatus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}
	status, err := client.GetStoreViewStatus()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

// AddStoreOffDate mağazayı geçici kapatır
// POST /api/migros/:integration_id/off-date
func (h *MigrosHandler) AddStoreOffDate(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}
	var req struct {
		Option string `json:"option"` // ONE_HOUR | FOUR_HOUR | NEXT_SHIFT_START
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Option == "" {
		req.Option = migrosSvc.StoreOffDateOneHour
	}
	if err := client.AddStoreOffDate(req.Option); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Mağaza geçici olarak kapatıldı"})
}

// RemoveStoreOffDate geçici kapatmayı kaldırır
// DELETE /api/migros/:integration_id/off-date
func (h *MigrosHandler) RemoveStoreOffDate(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}
	if err := client.RemoveStoreOffDate(); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Geçici kapatma kaldırıldı"})
}

// GetWorkingHours çalışma saatlerini döner
// GET /api/migros/:integration_id/working-hours
func (h *MigrosHandler) GetWorkingHours(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}
	hours, err := client.GetWorkingHours()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": hours})
}

// UpdateWorkingHours çalışma saatlerini günceller
// PUT /api/migros/:integration_id/working-hours
func (h *MigrosHandler) UpdateWorkingHours(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}
	var req struct {
		Slots []migrosSvc.WorkingHourItem `json:"slots"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Slots) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slots alanı gereklidir"})
		return
	}
	if err := client.UpdateWorkingHours(req.Slots); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Çalışma saatleri güncellendi"})
}

// GetPaymentMethods ödeme yöntemlerini döner
// GET /api/migros/:integration_id/payment-methods
func (h *MigrosHandler) GetPaymentMethods(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}
	methods, err := client.GetPaymentMethods()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": methods})
}

// UpdatePaymentMethods ödeme yöntemlerini günceller
// PUT /api/migros/:integration_id/payment-methods
func (h *MigrosHandler) UpdatePaymentMethods(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}
	var req struct {
		Updates []migrosSvc.PaymentStatusUpdate `json:"updates"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "updates alanı gereklidir"})
		return
	}
	if err := client.UpdatePaymentMethods(req.Updates); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Ödeme yöntemleri güncellendi"})
}

// UpdateOptionStatus opsiyon durumunu günceller
// PUT /api/migros/:integration_id/options/:option_id/status
func (h *MigrosHandler) UpdateOptionStatus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}
	optionID, err := strconv.ParseInt(c.Param("option_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz opsiyon ID"})
		return
	}
	var req struct {
		IsAvailable bool `json:"is_available"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := client.UpdateOptionItemStatus(optionID, req.IsAvailable); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	status := "pasif"
	if req.IsAvailable {
		status = "aktif"
	}
	c.JSON(http.StatusOK, gin.H{"message": "Opsiyon " + status + " yapıldı"})
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
