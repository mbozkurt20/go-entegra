package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// JSONMap stores arbitrary key-value pairs for provider credentials
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for JSONMap")
	}
	return json.Unmarshal(bytes, j)
}

type RestaurantProvider struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	RestaurantID  uint           `gorm:"not null;index" json:"restaurant_id"`
	ProviderID    uint           `gorm:"not null;index" json:"provider_id"`
	Name          string         `gorm:"not null" json:"name"`
	Slug          string         `gorm:"not null" json:"slug"`
	Information   JSONMap        `gorm:"type:jsonb" json:"information"` // provider api keys: restaurantId, secretKey, vb.
	Status        string         `gorm:"default:'1'" json:"status"`     // "1"=aktif, "0"=pasif
	IsEcoFriendly string         `json:"is_eco_friendly"`               // örn: "Servis"
	DoNotKnock    string         `json:"do_not_knock"`                  // örn: "Zil Çalma"
	DropOffAtDoor string         `json:"drop_off_at_door"`              // örn: "Temassız Teslimat"
	AutoApprove   string         `gorm:"default:'0'" json:"auto_approve"` // "1"=açık, "0"=kapalı
	Service       string         `json:"service"`                       // örn: "15"
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	Restaurant Restaurant `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	Provider   Provider   `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}
