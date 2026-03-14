package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go-entegra/internal/models"
	"go-entegra/internal/services"
	ysSvc "go-entegra/internal/services/yemeksepeti"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type YemeksepetiHandler struct {
	db      *gorm.DB
	webhook *services.WebhookService
}

func NewYemeksepetiHandler(db *gorm.DB, webhook *services.WebhookService) *YemeksepetiHandler {
	return &YemeksepetiHandler{db: db, webhook: webhook}
}

// newClient integration_id'den YemekSepeti client oluşturur
func (h *YemeksepetiHandler) newClient(rp *models.RestaurantProvider) (*ysSvc.Client, error) {
	info := rp.Information
	username, _ := info["username"].(string)
	password, _ := info["password"].(string)
	chainCode, _ := info["chainCode"].(string)
	posVendorId, _ := info["posVendorId"].(string)
	baseURL, _ := info["baseUrl"].(string)

	if username == "" || password == "" || chainCode == "" || posVendorId == "" {
		return nil, fmt.Errorf("YemekSepeti entegrasyon bilgileri eksik (username, password, chainCode, posVendorId)")
	}
	return ysSvc.NewClient(username, password, chainCode, posVendorId, baseURL), nil
}

// getIntegration integration_id path param'dan RestaurantProvider yükler
func (h *YemeksepetiHandler) getIntegration(c *gin.Context) (*models.RestaurantProvider, bool) {
	id := c.Param("integration_id")
	var rp models.RestaurantProvider
	if err := h.db.
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Where("restaurant_providers.id = ? AND providers.slug = 'yemeksepeti'", id).
		Preload("Restaurant.Business").
		Preload("Provider").
		First(&rp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "YemekSepeti entegrasyonu bulunamadı"})
		return nil, false
	}
	return &rp, true
}

// validateIncomingJWT gelen siparişin JWT token'ını doğrular
// DH, Authorization: Bearer <JWT> gönderir; payload: { "service": "middleware" }
// Secret: information["jwtSecret"]
func validateIncomingJWT(tokenStr string, jwtSecret string) error {
	if jwtSecret == "" {
		return nil // secret tanımlı değilse doğrulama atla
	}
	_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("beklenmeyen imza metodu: %v", t.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	return err
}

// ─────────────────────────────────────────────
// IncomingOrder — DH bize sipariş gönderir
// POST /webhook/yemeksepeti/:pos_vendor_id
// ─────────────────────────────────────────────
func (h *YemeksepetiHandler) IncomingOrder(c *gin.Context) {
	posVendorId := c.Param("pos_vendor_id")

	// Entegrasyonu bul
	var rp models.RestaurantProvider
	err := h.db.
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Preload("Restaurant.Business").
		Preload("Provider").
		Where("providers.slug = 'yemeksepeti' AND restaurant_providers.status = '1'").
		Where("restaurant_providers.information->>'posVendorId' = ?", posVendorId).
		First(&rp).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entegrasyon bulunamadı"})
		return
	}

	// JWT doğrulama
	if jwtSecret, _ := rp.Information["jwtSecret"].(string); jwtSecret != "" {
		authHeader := c.GetHeader("Authorization")
		tokenStr := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenStr = authHeader[7:]
		}
		if err := validateIncomingJWT(tokenStr, jwtSecret); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Geçersiz JWT"})
			return
		}
	}

	var payload ysSvc.IncomingOrder
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz payload"})
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

	// Sipariş durumu
	status := models.OrderStatusPending
	if active && rp.AutoApprove == "1" {
		status = models.OrderStatusApproved
	}

	// Müşteri adı
	customerName := payload.Customer.FirstName
	if payload.Customer.LastName != "" {
		customerName += " " + payload.Customer.LastName
	}

	// Adres
	address := ""
	if payload.Delivery != nil && payload.Delivery.Address != nil {
		a := payload.Delivery.Address
		address = fmt.Sprintf("%s %s %s %s %s", a.Street, a.Number, a.FlatNumber, a.DeliveryArea, a.City)
	}

	// Teslimat tipi
	deliveryType := 1 // delivery
	if payload.ExpeditionType == "pickup" {
		deliveryType = 2
	}

	// Toplam tutar
	totalAmount := 0.0
	fmt.Sscanf(payload.Price.GrandTotal, "%f", &totalAmount)

	order := models.Order{
		RestaurantID:         rp.RestaurantID,
		RestaurantProviderID: rp.ID,
		ProviderOrderID:      payload.Token, // DH orderToken
		Status:               status,
		Active:               active,
		RawPayload:           models.JSONMap(map[string]interface{}{"order": payload}),
		TotalAmount:          totalAmount,
		CustomerName:         customerName,
		CustomerPhone:        payload.Customer.MobilePhone,
		CustomerAddress:      address,
		Note:                 payload.Comments.CustomerComment,
		DeliveryType:         deliveryType,
		PaymentMethod:        payload.Payment.Type,
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sipariş kaydedilemedi"})
		return
	}

	// Kontör hareketi
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

	// Auto-approve: DH Middleware'e kabul bildir
	if active && rp.AutoApprove == "1" {
		go func() {
			client, err := h.newClient(&rp)
			if err != nil {
				log.Printf("YemekSepeti client oluşturulamadı: %v", err)
				return
			}
			remoteOrderId := fmt.Sprintf("YS_%d", order.ID)
			acceptanceTime := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)
			if err := client.AcceptOrder(payload.Token, remoteOrderId, acceptanceTime); err != nil {
				log.Printf("YemekSepeti otomatik onay başarısız (token: %s): %v", payload.Token, err)
			}
		}()

		if rp.Restaurant.WebhookURL != "" {
			go h.webhook.SendOrder(&order, rp.Restaurant.WebhookURL)
		}
	}

	// DH'ye remoteOrderId döndür (200 — acknowledged)
	remoteOrderId := fmt.Sprintf("YS_%d", order.ID)
	c.JSON(http.StatusOK, ysSvc.OrderDispatchAcknowledgedResponse{
		RemoteResponse: ysSvc.RemoteResponseAck{
			RemoteOrderID: remoteOrderId,
		},
	})
}

// ─────────────────────────────────────────────
// AcceptOrder — Manuel sipariş onayı
// POST /api/yemeksepeti/:integration_id/orders/:order_token/accept
// ─────────────────────────────────────────────
func (h *YemeksepetiHandler) AcceptOrder(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	orderToken := c.Param("order_token")

	var req struct {
		AcceptanceTime string `json:"acceptance_time"` // opsiyonel, yoksa 30dk sonrası
	}
	c.ShouldBindJSON(&req)
	if req.AcceptanceTime == "" {
		req.AcceptanceTime = time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)
	}

	client, err := h.newClient(rp)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// DB'de siparişi bul
	var order models.Order
	if err := h.db.Where("provider_order_id = ? AND restaurant_provider_id = ?", orderToken, rp.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sipariş bulunamadı"})
		return
	}

	remoteOrderId := fmt.Sprintf("YS_%d", order.ID)
	if err := client.AcceptOrder(orderToken, remoteOrderId, req.AcceptanceTime); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	h.db.Model(&order).Update("status", models.OrderStatusApproved)
	c.JSON(http.StatusOK, gin.H{"message": "Sipariş onaylandı"})
}

// ─────────────────────────────────────────────
// RejectOrder — Manuel sipariş reddi
// POST /api/yemeksepeti/:integration_id/orders/:order_token/reject
// Body: { "reason": "ITEM_UNAVAILABLE", "message": "..." }
// ─────────────────────────────────────────────
func (h *YemeksepetiHandler) RejectOrder(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	orderToken := c.Param("order_token")

	var req struct {
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Reason == "" {
		req.Reason = ysSvc.RejectReasonTechnicalProblem
	}

	client, err := h.newClient(rp)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.RejectOrder(orderToken, req.Reason, req.Message); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var order models.Order
	if h.db.Where("provider_order_id = ? AND restaurant_provider_id = ?", orderToken, rp.ID).First(&order).Error == nil {
		h.db.Model(&order).Update("status", models.OrderStatusCancelled)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sipariş reddedildi"})
}

// ─────────────────────────────────────────────
// PickupOrder — Kurye siparişi teslim aldı
// POST /api/yemeksepeti/:integration_id/orders/:order_token/pickup
// ─────────────────────────────────────────────
func (h *YemeksepetiHandler) PickupOrder(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	orderToken := c.Param("order_token")

	client, err := h.newClient(rp)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.PickupOrder(orderToken); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var order models.Order
	if h.db.Where("provider_order_id = ? AND restaurant_provider_id = ?", orderToken, rp.ID).First(&order).Error == nil {
		h.db.Model(&order).Update("status", models.OrderStatusDelivered)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kurye teslim aldı bildirimi gönderildi"})
}

// ─────────────────────────────────────────────
// PreparationCompleted — Yemek hazır (DH kuryesi için)
// POST /api/yemeksepeti/:integration_id/orders/:order_token/prepared
// ─────────────────────────────────────────────
func (h *YemeksepetiHandler) PreparationCompleted(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	orderToken := c.Param("order_token")

	client, err := h.newClient(rp)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.PreparationCompleted(orderToken); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var order models.Order
	if h.db.Where("provider_order_id = ? AND restaurant_provider_id = ?", orderToken, rp.ID).First(&order).Error == nil {
		h.db.Model(&order).Update("status", models.OrderStatusReady)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Yemek hazır bildirimi gönderildi"})
}

// ─────────────────────────────────────────────
// GetStatus — Restoran açık/kapalı durumu
// GET /api/yemeksepeti/:integration_id/status
// ─────────────────────────────────────────────
func (h *YemeksepetiHandler) GetStatus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}

	client, err := h.newClient(rp)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entries, err := client.GetAvailability()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": entries})
}

// ─────────────────────────────────────────────
// SetStatus — Restoran aç/kapat
// PUT /api/yemeksepeti/:integration_id/status
// Body: { "is_open": true } veya { "is_open": false, "reason": "TOO_BUSY_KITCHEN", "minutes": 30 }
// ─────────────────────────────────────────────
func (h *YemeksepetiHandler) SetStatus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}

	var req struct {
		IsOpen  bool   `json:"is_open"`
		Reason  string `json:"reason"`
		Minutes int    `json:"minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.newClient(rp)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	avReq := ysSvc.AvailabilityUpdateRequest{}
	if req.IsOpen {
		avReq.AvailabilityState = "OPEN"
	} else if req.Minutes > 0 {
		avReq.AvailabilityState = "CLOSED_UNTIL"
		avReq.ClosingMinutes = &req.Minutes
		if req.Reason != "" {
			avReq.ClosedReason = &req.Reason
		}
	} else {
		avReq.AvailabilityState = "CLOSED"
		if req.Reason != "" {
			avReq.ClosedReason = &req.Reason
		}
	}

	if err := client.SetAvailability(avReq); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	msg := "Restoran kapatıldı"
	if req.IsOpen {
		msg = "Restoran açıldı"
	}
	c.JSON(http.StatusOK, gin.H{"message": msg})
}
