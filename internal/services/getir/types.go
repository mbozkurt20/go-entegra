package getir

import "time"

// --- Auth ---

type TokenRequest struct {
	AppSecretKey        string `json:"appSecretKey"`
	RestaurantSecretKey string `json:"restaurantSecretKey"`
}

type TokenResponse struct {
	Token        string `json:"token"`
	RestaurantID string `json:"restaurantId"`
}

// --- Incoming Order (Getir → bizim webhook) ---

type IncomingOrder struct {
	ID               string      `json:"id"`
	RestaurantID     string      `json:"restaurantId"`
	Client           OrderClient `json:"client"`
	DeliveryTime     int         `json:"deliveryTime"` // dakika
	Products         []Product   `json:"products"`
	TotalPrice       float64     `json:"totalPrice"`
	DiscountAmount   float64     `json:"discountAmount"`   // İndirim tutarı
	Currency         string      `json:"currency"`
	Note             string      `json:"note"`
	Status           string      `json:"status"`
	DeliveryType     int         `json:"deliveryType"` // 1=Getir Getirsin, 2=Restoran Getirsin
	Address          Address     `json:"address"`
	PaymentMethod    string      `json:"paymentMethod"`    // Ödeme yöntemi
	VerificationCode string      `json:"verificationCode"` // Sipariş doğrulama kodu (ör. h593)
	ScheduledAt      *time.Time  `json:"scheduledAt"`      // İleri tarihli sipariş
	Promotions       []Promotion `json:"promotions"`       // Kampanyalar
	CreatedAt        time.Time   `json:"createdAt"`
}

type OrderClient struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
}

type Address struct {
	Description string  `json:"description"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}

type Product struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Count   int      `json:"count"`
	Price   float64  `json:"price"`
	Note    string   `json:"note"` // Ürün notu
	Options []Option `json:"options"`
}

type Option struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type Promotion struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Type   string  `json:"type"` // "ORTAKKAMPANYA", vb.
}

// --- Order Verify/Cancel ---

type VerifyOrderRequest struct {
	Status string `json:"status"` // "Approved"
}

type CancelOrderRequest struct {
	CancelReasonID int    `json:"cancelReasonId"`
	CancelNote     string `json:"cancelNote"`
}

type CancelOption struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// --- Gelen Statü Değişikliği (Getir → biz) ---

type IncomingStatusChange struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurantId"`
	Status       string `json:"status"` // "Cancelled", "Delivered", vb.
	CancelReason string `json:"cancelReason,omitempty"`
}

// --- Sipariş Sorgulama (Inquiry) ---

type OrderInquiryResponse struct {
	ID           string      `json:"id"`
	RestaurantID string      `json:"restaurantId"`
	Status       string      `json:"status"`
	TotalPrice   float64     `json:"totalPrice"`
	Client       OrderClient `json:"client"`
	Products     []Product   `json:"products"`
	DeliveryType int         `json:"deliveryType"`
	CreatedAt    string      `json:"createdAt"`
}

// --- Restoran ---

type RestaurantInfo struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Status                  int    `json:"status"`
	AveragePreparationTime  int    `json:"averagePreparationTime"`
	IsCourierAvailable      bool   `json:"isCourierAvailable"`
}

// timeOffAmount: 15, 30, 45 (dakika) — sadece kapatma için
type CloseRestaurantRequest struct {
	TimeOffAmount int `json:"timeOffAmount,omitempty"` // 15, 30, 45
}

type BusynessRequest struct {
	IsBusy                   bool `json:"isBusy"`
	BusynessDifferenceDuration int  `json:"busynessDifferenceDuration,omitempty"` // 15, 30, 45
}

type CourierDisableRequest struct {
	TimeOffAmount int `json:"timeOffAmount"` // 15, 30, 45
}

// --- Çalışma Saatleri ---

type WorkingHourDay struct {
	Day          int          `json:"day"` // 0=Pazar, 1=Pazartesi, ..., 6=Cumartesi
	WorkingHours WorkingHours `json:"workingHours"`
	Closed       bool         `json:"closed"`
}

type WorkingHours struct {
	StartTime string `json:"startTime"` // "09:00"
	EndTime   string `json:"endTime"`   // "22:00"
}

type WorkingHoursResponse struct {
	RestaurantWorkingHours []WorkingHourDay `json:"restaurantWorkingHours"`
	RestaurantCourierHours []WorkingHourDay `json:"restaurantCourierHours"`
}

// --- Menu / Products ---

// ProductStatus: 100=ACTIVE, 200=INACTIVE, 400=DAILY_INACTIVE
const (
	ProductStatusActive      = 100
	ProductStatusInactive    = 200
	ProductStatusDailyInactive = 400
)

type UpdateProductStatusRequest struct {
	Status int `json:"status"` // 100, 200, 400
}

type UpdateOptionStatusRequest struct {
	Status int `json:"status"` // 100, 200
}

// LocalizedName is a multilingual name object returned by Getir API
type LocalizedName struct {
	TR string `json:"tr"`
	EN string `json:"en"`
}

func (l LocalizedName) String() string {
	if l.TR != "" {
		return l.TR
	}
	return l.EN
}

type MenuOption struct {
	ID     string        `json:"id"`
	Name   LocalizedName `json:"name"`
	Price  float64       `json:"price"`
	Status int           `json:"status"` // 100=aktif, 200=pasif
}

type MenuOptionCategory struct {
	ID      string        `json:"id"`
	Name    LocalizedName `json:"name"`
	Options []MenuOption  `json:"options"`
}

type MenuProduct struct {
	ID               string               `json:"id"`
	Name             LocalizedName        `json:"name"`
	Description      LocalizedName        `json:"description"`
	Price            float64              `json:"price"`
	ImageURL         string               `json:"imageURL"`
	Status           int                  `json:"status"` // 100=active, 200=inactive, 400=daily_inactive
	OptionCategories []MenuOptionCategory `json:"optionCategories,omitempty"`
}

// IsAvailable returns true if product status is active
func (p MenuProduct) IsAvailable() bool { return p.Status == ProductStatusActive }

type MenuCategory struct {
	ID       string        `json:"id"`
	Name     LocalizedName `json:"name"`
	Products []MenuProduct `json:"products"`
}

type MenuResponse struct {
	ProductCategories []MenuCategory `json:"productCategories"`
}

type UpdateProductPriceRequest struct {
	Price float64 `json:"price"`
}

// --- Chain Menus ---

type ChainMenu struct {
	ID       string         `json:"id"`
	Name     LocalizedName  `json:"name"`
	Products []ChainProduct `json:"products,omitempty"`
}

type ChainProduct struct {
	ID    string  `json:"id"`
	Name  LocalizedName `json:"name"`
	Price float64 `json:"price"`
}

type ChainOptionCategory struct {
	ID      string        `json:"id"`
	Name    LocalizedName `json:"name"`
	Options []ChainOption `json:"options"`
}

type ChainOption struct {
	ID    string  `json:"id"`
	Name  LocalizedName `json:"name"`
	Price float64 `json:"price"`
}

type UpdateChainPricesRequest struct {
	ChainProducts []ChainPriceItem `json:"chainProducts"`
	ChainOptions  []ChainPriceItem `json:"chainOptions"`
}

type ChainPriceItem struct {
	ID    string  `json:"id"`
	Price float64 `json:"price"`
}
