package handlers

import (
	"net/http"
	"strconv"

	"go-entegra/internal/middleware"
	"go-entegra/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)


type RestaurantHandler struct {
	db *gorm.DB
}

func NewRestaurantHandler(db *gorm.DB) *RestaurantHandler {
	return &RestaurantHandler{db: db}
}

type CreateRestaurantRequest struct {
	Name       string `json:"name" binding:"required"`
	Slug       string `json:"slug" binding:"required"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	WebhookURL string `json:"webhook_url"`
	Credits    int    `json:"credits"`
}

type UpdateRestaurantRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	WebhookURL string `json:"webhook_url"`
	Status     string `json:"status"`
}

// List godoc
// GET /restaurants
func (h *RestaurantHandler) List(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)

	var restaurants []models.Restaurant
	h.db.Where("business_id = ?", businessID).Find(&restaurants)

	c.JSON(http.StatusOK, gin.H{"data": restaurants})
}

// Get godoc
// GET /restaurants/:id
func (h *RestaurantHandler) Get(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var restaurant models.Restaurant
	if err := h.db.Where("id = ? AND business_id = ?", id, businessID).
		Preload("RestaurantProviders.Provider").
		First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": restaurant})
}

// Create godoc
// POST /restaurants
func (h *RestaurantHandler) Create(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)

	var req CreateRestaurantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.Restaurant
	if err := h.db.Where("slug = ?", req.Slug).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Slug already in use"})
		return
	}

	restaurant := models.Restaurant{
		BusinessID: businessID,
		Name:       req.Name,
		Slug:       req.Slug,
		Phone:      req.Phone,
		Address:    req.Address,
		WebhookURL: req.WebhookURL,
		Status:     "active",
		Credits:    req.Credits,
	}

	if err := h.db.Create(&restaurant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create restaurant"})
		return
	}

	// Başlangıç kontörü varsa hareket kaydı
	if restaurant.Credits > 0 {
		tx := models.CreditTransaction{
			RestaurantID: restaurant.ID,
			Amount:       restaurant.Credits,
			Balance:      restaurant.Credits,
			Type:         models.CreditTxAdd,
			Source:       models.CreditSrcManual,
			Description:  "Başlangıç kontörü",
		}
		h.db.Create(&tx)
	}

	c.JSON(http.StatusCreated, gin.H{"data": restaurant})
}

// Update godoc
// PUT /restaurants/:id
func (h *RestaurantHandler) Update(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var restaurant models.Restaurant
	if err := h.db.Where("id = ? AND business_id = ?", id, businessID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	var req UpdateRestaurantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		restaurant.Name = req.Name
	}
	if req.Phone != "" {
		restaurant.Phone = req.Phone
	}
	if req.Address != "" {
		restaurant.Address = req.Address
	}
	if req.WebhookURL != "" {
		restaurant.WebhookURL = req.WebhookURL
	}
	if req.Status != "" {
		restaurant.Status = req.Status
	}

	h.db.Save(&restaurant)
	c.JSON(http.StatusOK, gin.H{"data": restaurant})
}

// AddCredits godoc
// POST /restaurants/:id/credits
func (h *RestaurantHandler) AddCredits(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var restaurant models.Restaurant
	if err := h.db.Where("id = ? AND business_id = ?", id, businessID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	var req struct {
		Amount int `json:"amount" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.db.Model(&restaurant).UpdateColumn("credits", gorm.Expr("credits + ?", req.Amount))
	h.db.First(&restaurant, id)

	// Hareket kaydı
	tx := models.CreditTransaction{
		RestaurantID: restaurant.ID,
		Amount:       req.Amount,
		Balance:      restaurant.Credits,
		Type:         models.CreditTxAdd,
		Source:       models.CreditSrcManual,
		Description:  "Manuel kontör ekleme",
	}
	h.db.Create(&tx)

	c.JSON(http.StatusOK, gin.H{"data": restaurant})
}

// ListCreditTransactions godoc
// GET /restaurants/:id/credit-transactions
func (h *RestaurantHandler) ListCreditTransactions(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	id, _ := strconv.Atoi(c.Param("id"))

	// Restoranın bu business'a ait olduğunu doğrula
	var restaurant models.Restaurant
	if err := h.db.Where("id = ? AND business_id = ?", id, businessID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	var txs []models.CreditTransaction
	h.db.Where("restaurant_id = ?", id).
		Order("created_at DESC").
		Limit(100).
		Find(&txs)

	c.JSON(http.StatusOK, gin.H{"data": txs, "restaurant": restaurant})
}

// Delete godoc
// DELETE /restaurants/:id
func (h *RestaurantHandler) Delete(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var restaurant models.Restaurant
	if err := h.db.Where("id = ? AND business_id = ?", id, businessID).First(&restaurant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	h.db.Delete(&restaurant)
	c.JSON(http.StatusOK, gin.H{"message": "Restaurant deleted"})
}
