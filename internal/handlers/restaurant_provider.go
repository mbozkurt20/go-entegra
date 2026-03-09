package handlers

import (
	"net/http"
	"strconv"

	"go-entegra/internal/middleware"
	"go-entegra/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RestaurantProviderHandler struct {
	db *gorm.DB
}

func NewRestaurantProviderHandler(db *gorm.DB) *RestaurantProviderHandler {
	return &RestaurantProviderHandler{db: db}
}

type CreateRestaurantProviderRequest struct {
	ProviderID    uint                   `json:"provider_id" binding:"required"`
	Name          string                 `json:"name" binding:"required"`
	Slug          string                 `json:"slug" binding:"required"`
	Information   map[string]interface{} `json:"information"`
	Status        string                 `json:"status"`
	IsEcoFriendly string                 `json:"is_eco_friendly"`
	DoNotKnock    string                 `json:"do_not_knock"`
	DropOffAtDoor string                 `json:"drop_off_at_door"`
	AutoApprove   string                 `json:"auto_approve"`
	Service       string                 `json:"service"`
}

type UpdateRestaurantProviderRequest struct {
	Name          string                 `json:"name"`
	Information   map[string]interface{} `json:"information"`
	Status        string                 `json:"status"`
	IsEcoFriendly string                 `json:"is_eco_friendly"`
	DoNotKnock    string                 `json:"do_not_knock"`
	DropOffAtDoor string                 `json:"drop_off_at_door"`
	AutoApprove   string                 `json:"auto_approve"`
	Service       string                 `json:"service"`
}

func (h *RestaurantProviderHandler) ownsRestaurant(businessID uint, restaurantID int) (*models.Restaurant, bool) {
	var restaurant models.Restaurant
	if err := h.db.Where("id = ? AND business_id = ?", restaurantID, businessID).First(&restaurant).Error; err != nil {
		return nil, false
	}
	return &restaurant, true
}

// List godoc
// GET /restaurants/:id/providers
func (h *RestaurantProviderHandler) List(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	restaurantID, _ := strconv.Atoi(c.Param("id"))

	if _, ok := h.ownsRestaurant(businessID, restaurantID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	var rps []models.RestaurantProvider
	h.db.Where("restaurant_id = ?", restaurantID).Preload("Provider").Find(&rps)

	c.JSON(http.StatusOK, gin.H{"data": rps})
}

// Get godoc
// GET /restaurants/:id/providers/:pid
func (h *RestaurantProviderHandler) Get(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	restaurantID, _ := strconv.Atoi(c.Param("id"))
	pid, _ := strconv.Atoi(c.Param("pid"))

	if _, ok := h.ownsRestaurant(businessID, restaurantID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	var rp models.RestaurantProvider
	if err := h.db.Where("id = ? AND restaurant_id = ?", pid, restaurantID).
		Preload("Provider").First(&rp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rp})
}

// Create godoc
// POST /restaurants/:id/providers
func (h *RestaurantProviderHandler) Create(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	restaurantID, _ := strconv.Atoi(c.Param("id"))

	if _, ok := h.ownsRestaurant(businessID, restaurantID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	var req CreateRestaurantProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var provider models.Provider
	if err := h.db.First(&provider, req.ProviderID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider not found"})
		return
	}

	// Her restoran için aynı provider'dan sadece 1 entegrasyon olabilir
	var existing models.RestaurantProvider
	if err := h.db.Where("restaurant_id = ? AND provider_id = ?", restaurantID, req.ProviderID).
		First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": provider.Name + " entegrasyonu bu restoran için zaten ekli"})
		return
	}

	status := req.Status
	if status == "" {
		status = "1"
	}
	autoApprove := req.AutoApprove
	if autoApprove == "" {
		autoApprove = "0"
	}

	rp := models.RestaurantProvider{
		RestaurantID:  uint(restaurantID),
		ProviderID:    req.ProviderID,
		Name:          req.Name,
		Slug:          req.Slug,
		Information:   models.JSONMap(req.Information),
		Status:        status,
		IsEcoFriendly: req.IsEcoFriendly,
		DoNotKnock:    req.DoNotKnock,
		DropOffAtDoor: req.DropOffAtDoor,
		AutoApprove:   autoApprove,
		Service:       req.Service,
	}

	if err := h.db.Create(&rp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create integration"})
		return
	}

	h.db.Preload("Provider").First(&rp, rp.ID)
	c.JSON(http.StatusCreated, gin.H{"data": rp})
}

// Update godoc
// PUT /restaurants/:id/providers/:pid
func (h *RestaurantProviderHandler) Update(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	restaurantID, _ := strconv.Atoi(c.Param("id"))
	pid, _ := strconv.Atoi(c.Param("pid"))

	if _, ok := h.ownsRestaurant(businessID, restaurantID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	var rp models.RestaurantProvider
	if err := h.db.Where("id = ? AND restaurant_id = ?", pid, restaurantID).First(&rp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	var req UpdateRestaurantProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		rp.Name = req.Name
	}
	if req.Information != nil {
		// Mevcut bilgileri koru, sadece gönderilen key'leri güncelle (secretKey gibi alanlar silinmez)
		if rp.Information == nil {
			rp.Information = make(models.JSONMap)
		}
		for k, v := range req.Information {
			if v != nil && v != "" {
				rp.Information[k] = v
			}
		}
	}
	if req.Status != "" {
		rp.Status = req.Status
	}
	if req.IsEcoFriendly != "" {
		rp.IsEcoFriendly = req.IsEcoFriendly
	}
	if req.DoNotKnock != "" {
		rp.DoNotKnock = req.DoNotKnock
	}
	if req.DropOffAtDoor != "" {
		rp.DropOffAtDoor = req.DropOffAtDoor
	}
	if req.AutoApprove != "" {
		rp.AutoApprove = req.AutoApprove
	}
	if req.Service != "" {
		rp.Service = req.Service
	}

	h.db.Save(&rp)
	h.db.Preload("Provider").First(&rp, rp.ID)
	c.JSON(http.StatusOK, gin.H{"data": rp})
}

// Delete godoc
// DELETE /restaurants/:id/providers/:pid
func (h *RestaurantProviderHandler) Delete(c *gin.Context) {
	businessID := middleware.GetBusinessID(c)
	restaurantID, _ := strconv.Atoi(c.Param("id"))
	pid, _ := strconv.Atoi(c.Param("pid"))

	if _, ok := h.ownsRestaurant(businessID, restaurantID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Restaurant not found"})
		return
	}

	var rp models.RestaurantProvider
	if err := h.db.Where("id = ? AND restaurant_id = ?", pid, restaurantID).First(&rp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	h.db.Delete(&rp)
	c.JSON(http.StatusOK, gin.H{"message": "Integration deleted"})
}
