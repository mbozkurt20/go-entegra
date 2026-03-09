package models

import "time"

type Provider struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Slug      string    `gorm:"uniqueIndex;not null" json:"slug"`
	Status    string    `gorm:"default:active" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	RestaurantProviders []RestaurantProvider `gorm:"foreignKey:ProviderID" json:"restaurant_providers,omitempty"`
}
