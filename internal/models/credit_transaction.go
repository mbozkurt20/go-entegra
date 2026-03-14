package models

import "time"

// CreditTransactionType hareketin türü
type CreditTransactionType string

const (
	CreditTxAdd     CreditTransactionType = "add"     // Kontör ekleme (manuel veya ödeme ile)
	CreditTxUse     CreditTransactionType = "use"     // Sipariş ile tüketim
	CreditTxPayment CreditTransactionType = "payment" // Ödeme ile satın alma (ilerisi için)
)

// CreditTransactionSource hareketin kaynağı
type CreditTransactionSource string

const (
	CreditSrcManual  CreditTransactionSource = "manual"  // Panel üzerinden manuel ekleme
	CreditSrcOrder   CreditTransactionSource = "order"   // Sipariş gelince otomatik tüketim
	CreditSrcPayment CreditTransactionSource = "payment" // Ödeme sistemi (ilerisi için)
)

type CreditTransaction struct {
	ID           uint                    `gorm:"primaryKey" json:"id"`
	RestaurantID uint                    `gorm:"not null;index" json:"restaurant_id"`
	Amount       int                     `gorm:"not null" json:"amount"`        // + ekleme, - tüketim
	Balance      int                     `gorm:"not null" json:"balance"`       // işlem sonrası bakiye
	Type         CreditTransactionType   `gorm:"not null" json:"type"`
	Source       CreditTransactionSource `gorm:"not null" json:"source"`
	Description  string                  `json:"description"`                   // "Manuel ekleme", "Sipariş #123" vb.
	OrderID      *uint                   `gorm:"index" json:"order_id"`         // use tipinde sipariş ID
	Reference    string                  `json:"reference"`                     // ödeme referansı (ilerisi için)
	CreatedAt    time.Time               `json:"created_at"`

	Restaurant Restaurant `gorm:"foreignKey:RestaurantID" json:"restaurant,omitempty"`
	Order      *Order     `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}
