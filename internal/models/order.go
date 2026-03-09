package models

import (
	"time"

	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusApproved  OrderStatus = "approved"
	OrderStatusPreparing OrderStatus = "preparing"
	OrderStatusReady     OrderStatus = "ready"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	RestaurantID         uint           `gorm:"not null;index" json:"restaurant_id"`
	RestaurantProviderID uint           `gorm:"not null;index" json:"restaurant_provider_id"`
	ProviderOrderID      string         `gorm:"index" json:"provider_order_id"`
	Status               OrderStatus    `gorm:"default:pending" json:"status"`
	RawPayload           JSONMap        `gorm:"type:jsonb" json:"raw_payload"`
	TotalAmount          float64        `json:"total_amount"`
	DiscountAmount       float64        `json:"discount_amount"`
	CustomerName         string         `json:"customer_name"`
	CustomerPhone        string         `json:"customer_phone"`
	CustomerAddress      string         `json:"customer_address"`
	Note                 string         `json:"note"`
	DeliveryType         int            `json:"delivery_type"`          // 1=Getir Getirsin, 2=Restoran Getirsin
	PaymentMethod        string         `json:"payment_method"`         // Ödeme yöntemi
	VerificationCode     string         `json:"verification_code"`      // Sipariş doğrulama kodu
	ScheduledAt          *time.Time     `json:"scheduled_at"`           // İleri tarihli sipariş
	WebhookSent          bool           `gorm:"default:false" json:"webhook_sent"`
	WebhookSentAt        *time.Time     `json:"webhook_sent_at"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`

	Restaurant         Restaurant         `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	RestaurantProvider RestaurantProvider `gorm:"foreignKey:RestaurantProviderID" json:"restaurant_provider,omitempty"`
}
