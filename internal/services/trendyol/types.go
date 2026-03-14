package trendyol

// --- Restaurant ---

type RestaurantStatusRequest struct {
	IsOpen bool `json:"isOpen"`
}

type DeliveryHour struct {
	DayOfWeek    string `json:"dayOfWeek"`
	OpeningTime  string `json:"openingTime"`
	ClosingTime  string `json:"closingTime"`
	DeliveryType string `json:"deliveryType"`
}

type StoreInfo struct {
	ID                   int64          `json:"id"`
	Name                 string         `json:"name"`
	Status               string         `json:"status"`        // "ACTIVE" | "PASSIVE"
	WorkingStatus        string         `json:"workingStatus"` // "OPEN" | "CLOSED"
	Address              string         `json:"address"`
	Description          string         `json:"description"`
	DeliveryHours        []DeliveryHour `json:"deliveryHours"`
	MinDeliveryTimeInMin int            `json:"minDeliveryTimeInMin"`
	MaxDeliveryTimeInMin int            `json:"maxDeliveryTimeInMin"`
	MinBasketPrice       float64        `json:"minBasketPrice"`
}

type WorkingHoursSlot struct {
	DayOfWeek    string `json:"dayOfWeek"`    // "MONDAY" .. "SUNDAY"
	OpeningTime  string `json:"openingTime"`  // "09:00:00"
	ClosingTime  string `json:"closingTime"`  // "22:00:00"
	DeliveryType string `json:"deliveryType"` // "GO"
}

type WorkingHoursRequest struct {
	WorkingHours []WorkingHoursSlot `json:"workingHours"`
}

type DeliveryTimeRequest struct {
	DeliveryTime int `json:"deliveryTime"` // dakika
}

type DeliveryZoneRequest struct {
	Name         string  `json:"name"`
	MinBasket    float64 `json:"minBasket"`
	DeliveryFee  float64 `json:"deliveryFee"`
	DeliveryTime int     `json:"deliveryTime"`
}

type DeliveryZonesRequest struct {
	DeliveryZones []DeliveryZoneRequest `json:"deliveryZones"`
}

// --- Batch ---

type BatchStatusResponse struct {
	BatchRequestID string `json:"batchRequestId"`
	Status         string `json:"status"`
	FailedItems    []struct {
		ItemID  string `json:"itemId"`
		Message string `json:"message"`
	} `json:"failedItems"`
}

// --- Section (Category) ---

type SectionStatusRequest struct {
	Status string `json:"status"` // "ACTIVE" | "PASSIVE"
}

// --- Webhook / Incoming Order ---

type WebhookOrderCustomer struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	PhoneNumber     string `json:"phoneNumber"`
	DeliveryAddress string `json:"deliveryAddress"`
}

type WebhookOrderItem struct {
	ID           int64                  `json:"id"`
	Name         string                 `json:"name"`
	Quantity     int                    `json:"quantity"`
	Price        float64                `json:"price"`
	ModifierList []WebhookOrderModifier `json:"modifierList"`
}

type WebhookOrderModifier struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type WebhookOrder struct {
	ID          string               `json:"id"`
	Status      string               `json:"status"`
	Customer    WebhookOrderCustomer `json:"customer"`
	OrderItems  []WebhookOrderItem   `json:"orderItems"`
	TotalAmount float64              `json:"totalAmount"`
	PaymentType string               `json:"paymentType"`
	Note        string               `json:"note"`
	StoreID     string               `json:"storeId"`
}

// --- Orders ---

type OrderStatusRequest struct {
	Status string `json:"status"`
}

type CancelOrderRequest struct {
	CancelReasonID   int    `json:"cancelReasonId"`
	CancelReasonNote string `json:"cancelReasonNote"`
}

// --- Menu ---

type MenuIngredient struct {
	ID     int64   `json:"id"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Status string  `json:"status"` // "ACTIVE" | "PASSIVE"
}

type MenuModifierProduct struct {
	ID       int64   `json:"id"`
	Position int     `json:"position"`
	Price    float64 `json:"price"`
}

type MenuModifierGroup struct {
	ID               int64                 `json:"id"`
	Name             string                `json:"name"`
	Min              int                   `json:"min"`
	Max              int                   `json:"max"`
	ModifierProducts []MenuModifierProduct `json:"modifierProducts"`
}

type MenuProductModifierRef struct {
	ID       int64 `json:"id"`
	Position int   `json:"position"`
}

type MenuProduct struct {
	ID               int64                    `json:"id"`
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	OriginalPrice    float64                  `json:"originalPrice"`
	SellingPrice     float64                  `json:"sellingPrice"`
	Status           string                   `json:"status"` // "ACTIVE" | "PASSIVE"
	Ingredients      []int64                  `json:"ingredients"`
	ExtraIngredients []int64                  `json:"extraIngredients"`
	ModifierGroups   []MenuProductModifierRef `json:"modifierGroups"`
}

type MenuSectionProduct struct {
	ID       int64 `json:"id"`
	Position int   `json:"position"`
}

type MenuSection struct {
	ID       int64                `json:"id"`
	Name     string               `json:"name"`
	Position int                  `json:"position"`
	Status   string               `json:"status"` // "ACTIVE" | "PASSIVE"
	Products []MenuSectionProduct `json:"products"`
}

type MenuResponse struct {
	Ingredients    []MenuIngredient    `json:"ingredients"`
	ModifierGroups []MenuModifierGroup `json:"modifierGroups"`
	Products       []MenuProduct       `json:"products"`
	Sections       []MenuSection       `json:"sections"`
}

type UpdateItemPriceRequest struct {
	Price float64 `json:"price"`
}

type UpdateItemStatusRequest struct {
	Status string `json:"status"` // "ACTIVE" | "PASSIVE"
}
