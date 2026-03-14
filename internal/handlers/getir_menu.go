package handlers

import (
	"net/http"
	"strconv"

	getirSvc "go-entegra/internal/services/getir"
	"go-entegra/internal/middleware"
	"go-entegra/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetirMenuHandler struct {
	db *gorm.DB
}

func NewGetirMenuHandler(db *gorm.DB) *GetirMenuHandler {
	return &GetirMenuHandler{db: db}
}

// getIntegration integration_id'den RestaurantProvider çeker ve business'a ait olduğunu doğrular
func (h *GetirMenuHandler) getIntegration(c *gin.Context) (*models.RestaurantProvider, bool) {
	businessID := middleware.GetBusinessID(c)
	integrationID, _ := strconv.Atoi(c.Param("integration_id"))

	var rp models.RestaurantProvider
	err := h.db.
		Joins("JOIN restaurants ON restaurants.id = restaurant_providers.restaurant_id").
		Joins("JOIN providers ON providers.id = restaurant_providers.provider_id").
		Where("restaurant_providers.id = ? AND restaurants.business_id = ? AND providers.slug = 'getir'",
			integrationID, businessID).
		First(&rp).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Getir entegrasyonu bulunamadı"})
		return nil, false
	}
	return &rp, true
}

func (h *GetirMenuHandler) newClient(c *gin.Context, rp *models.RestaurantProvider) (*getirSvc.Client, bool) {
	client, err := getirSvc.NewClientFromInfo(map[string]interface{}(rp.Information), rp.Service)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, false
	}
	return client, true
}

// GetRestaurantInfo restoran bilgilerini getirir
// GET /api/getir/:integration_id/info
func (h *GetirMenuHandler) GetRestaurantInfo(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	info, err := client.GetRestaurantInfo()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": info})
}

// GetMenu menüyü Getir'den çeker
// GET /api/getir/:integration_id/menu
func (h *GetirMenuHandler) GetMenu(c *gin.Context) {
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

// GetCategories kategorileri Getir'den çeker
// GET /api/getir/:integration_id/categories
func (h *GetirMenuHandler) GetCategories(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	cats, err := client.GetCategories()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": cats})
}

// UpdateProductStatus ürünü aktif/pasif/günlük-pasif yapar
// PUT /api/getir/:integration_id/products/:product_id/status
// Body: {"status": 100|200|400}  — 100=aktif, 200=pasif, 400=bugün-pasif
func (h *GetirMenuHandler) UpdateProductStatus(c *gin.Context) {
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
		Status int `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status alanı gereklidir (100=aktif, 200=pasif, 400=bugün-pasif)"})
		return
	}

	if req.Status != getirSvc.ProductStatusActive && req.Status != getirSvc.ProductStatusInactive && req.Status != getirSvc.ProductStatusDailyInactive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz status. 100, 200 veya 400 olmalıdır"})
		return
	}

	if err := client.UpdateProductStatus(productID, req.Status); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	labels := map[int]string{100: "aktif", 200: "pasif", 400: "bugün pasif"}
	c.JSON(http.StatusOK, gin.H{"message": "Ürün " + labels[req.Status] + " yapıldı"})
}

// UpdateOptionStatus opsiyonu aktif/pasif yapar
// PUT /api/getir/:integration_id/options/:option_id/status
// Body: {"status": 100|200}
func (h *GetirMenuHandler) UpdateOptionStatus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	optionID := c.Param("option_id")

	var req struct {
		Status int `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status alanı gereklidir (100=aktif, 200=pasif)"})
		return
	}

	if err := client.UpdateOptionStatus(optionID, req.Status); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	label := "pasif"
	if req.Status == getirSvc.ProductStatusActive {
		label = "aktif"
	}
	c.JSON(http.StatusOK, gin.H{"message": "Opsiyon " + label + " yapıldı"})
}

// GetChainMenus zincir menü listesini getirir
// GET /api/getir/:integration_id/chain-menus
func (h *GetirMenuHandler) GetChainMenus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	menus, err := client.GetChainMenus()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": menus})
}

// GetChainMenu belirli bir zincir menüyü getirir
// GET /api/getir/:integration_id/chain-menus/:chain_menu_id
func (h *GetirMenuHandler) GetChainMenu(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	menu, err := client.GetChainMenu(c.Param("chain_menu_id"))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": menu})
}

// GetChainOptionCategories zincir opsiyon kategorilerini getirir
// GET /api/getir/:integration_id/chain-option-categories
func (h *GetirMenuHandler) GetChainOptionCategories(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	cats, err := client.GetChainOptionCategories()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cats})
}

// UpdateChainMenuPrices zincir menüdeki ürün/opsiyon fiyatlarını günceller
// POST /api/getir/:integration_id/chain-menus/:chain_menu_id/update-prices
// Body: {"chain_products": [{"id":"...","price":99.9}], "chain_options": [{"id":"...","price":5.0}]}
func (h *GetirMenuHandler) UpdateChainMenuPrices(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	chainMenuID := c.Param("chain_menu_id")

	var req struct {
		ChainProducts []getirSvc.ChainProductPriceItem `json:"chain_products"`
		ChainOptions  []getirSvc.ChainOptionPriceItem  `json:"chain_options"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.ChainProducts) == 0 && len(req.ChainOptions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "En az bir ürün veya opsiyon belirtilmelidir"})
		return
	}

	if err := client.UpdateChainMenuPrices(chainMenuID, req.ChainProducts, req.ChainOptions); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fiyatlar güncellendi"})
}

// UpdateProductPrice ürün fiyatını chain menüde bulup günceller
// PUT /api/getir/:integration_id/products/:product_id/price
// Body: {"price": 99.9}
func (h *GetirMenuHandler) UpdateProductPrice(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçerli bir fiyat girin"})
		return
	}

	if err := client.UpdateProductPrice(productID, req.Price); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fiyat güncellendi"})
}

// SetStatus restoran Getir'de açar/kapatır
// PUT /api/getir/:integration_id/status
// Body: {"is_open": true} veya {"is_open": false, "time_off_amount": 15}
func (h *GetirMenuHandler) SetStatus(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	var req struct {
		IsOpen        bool `json:"is_open"`
		TimeOffAmount int  `json:"time_off_amount"` // 15, 30, 45 (kapatma için)
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.SetRestaurantStatus(req.IsOpen, req.TimeOffAmount); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	status := "kapatıldı"
	if req.IsOpen {
		status = "açıldı"
	}
	c.JSON(http.StatusOK, gin.H{"message": "Restoran Getir'de " + status})
}

// SetBusyness restoran yoğunluk durumunu günceller
// PUT /api/getir/:integration_id/busyness
// Body: {"is_busy": true, "duration": 15}
func (h *GetirMenuHandler) SetBusyness(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	var req struct {
		IsBusy   bool `json:"is_busy"`
		Duration int  `json:"duration"` // 15, 30, 45
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.SetBusyness(req.IsBusy, req.Duration); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Yoğunluk güncellendi"})
}

// SetCourier kurye servisini açar/kapatır
// PUT /api/getir/:integration_id/courier
// Body: {"is_enabled": true} veya {"is_enabled": false, "time_off_amount": 15}
func (h *GetirMenuHandler) SetCourier(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	var req struct {
		IsEnabled     bool `json:"is_enabled"`
		TimeOffAmount int  `json:"time_off_amount"` // 15, 30, 45
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error
	if req.IsEnabled {
		err = client.EnableCourier()
	} else {
		err = client.DisableCourier(req.TimeOffAmount)
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	status := "devre dışı bırakıldı"
	if req.IsEnabled {
		status = "etkinleştirildi"
	}
	c.JSON(http.StatusOK, gin.H{"message": "Kurye " + status})
}

// GetWorkingHours çalışma saatlerini getirir
// GET /api/getir/:integration_id/working-hours
func (h *GetirMenuHandler) GetWorkingHours(c *gin.Context) {
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

// GetOptionProducts restoran opsiyon ürünlerini getirir
// GET /api/getir/:integration_id/option-products
func (h *GetirMenuHandler) GetOptionProducts(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	products, err := client.GetOptionProducts()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": products})
}

// GetAllPaymentMethods Getir'deki tüm ödeme yöntemlerini getirir
// GET /api/getir/:integration_id/all-payment-methods
func (h *GetirMenuHandler) GetAllPaymentMethods(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	methods, err := client.GetAllPaymentMethods()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": methods})
}

// GetPaymentMethods restoran ödeme yöntemlerini getirir
// GET /api/getir/:integration_id/payment-methods
func (h *GetirMenuHandler) GetPaymentMethods(c *gin.Context) {
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

// AddPaymentMethod restorana ödeme yöntemi ekler
// POST /api/getir/:integration_id/payment-methods
func (h *GetirMenuHandler) AddPaymentMethod(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	var req struct {
		PaymentMethodID string `json:"paymentMethodId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paymentMethodId alanı gereklidir"})
		return
	}

	if err := client.AddPaymentMethod(req.PaymentMethodID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Ödeme yöntemi eklendi"})
}

// DeletePaymentMethod restorandan ödeme yöntemi siler
// DELETE /api/getir/:integration_id/payment-methods
func (h *GetirMenuHandler) DeletePaymentMethod(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	var req struct {
		PaymentMethodID string `json:"paymentMethodId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paymentMethodId alanı gereklidir"})
		return
	}

	if err := client.DeletePaymentMethod(req.PaymentMethodID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Ödeme yöntemi silindi"})
}

// SetWorkingHours çalışma saatlerini günceller
// PUT /api/getir/:integration_id/working-hours
// Body: [{"day":0,"workingHours":{"startTime":"09:00","endTime":"22:00"},"closed":false}, ...]
func (h *GetirMenuHandler) SetWorkingHours(c *gin.Context) {
	rp, ok := h.getIntegration(c)
	if !ok {
		return
	}
	client, ok := h.newClient(c, rp)
	if !ok {
		return
	}

	var days []getirSvc.WorkingHourDay
	if err := c.ShouldBindJSON(&days); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := client.SetWorkingHours(days); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Çalışma saatleri güncellendi"})
}
