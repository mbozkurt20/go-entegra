package handlers

import (
	"net/http"

	"go-entegra/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProviderHandler struct {
	db *gorm.DB
}

func NewProviderHandler(db *gorm.DB) *ProviderHandler {
	return &ProviderHandler{db: db}
}

// List godoc
// GET /providers
func (h *ProviderHandler) List(c *gin.Context) {
	var providers []models.Provider
	h.db.Where("status = ?", "active").Find(&providers)
	c.JSON(http.StatusOK, gin.H{"data": providers})
}
