package trendyol

// --- Restaurant Status ---

type RestaurantStatusRequest struct {
	IsOpen bool `json:"isOpen"`
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
