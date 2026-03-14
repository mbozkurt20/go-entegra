package models

import (
	"time"

	"gorm.io/gorm"
)

type Restaurant struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	BusinessID  uint           `gorm:"not null;index" json:"business_id"`
	Name        string         `gorm:"not null" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null" json:"slug"`
	Phone       string         `json:"phone"`
	Address     string         `json:"address"`
	WebhookURL  string         `json:"webhook_url"`
	Status      string         `gorm:"default:active" json:"status"`
	Credits     int            `gorm:"default:0" json:"credits"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Business            Business             `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	RestaurantProviders []RestaurantProvider `gorm:"foreignKey:RestaurantID" json:"restaurant_providers,omitempty"`
	Orders              []Order              `gorm:"foreignKey:RestaurantID" json:"orders,omitempty"`
}
