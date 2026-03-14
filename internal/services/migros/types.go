package migros

// --- Store ---

type StoreDetailResponse struct {
	Data    StoreDetail `json:"data"`
	Success bool        `json:"success"`
}

type StoreDetail struct {
	IntegrationIsActive bool   `json:"integrationIsActive"`
	ID                  int64  `json:"id"`
	WarehouseID         int64  `json:"warehouseId"`
	StoreGroupID        int64  `json:"storeGroupId"`
	PrepareDuration     int64  `json:"prepareDuration"`
	InWorkingHours      bool   `json:"inWorkingHours"`
	StoreName           string `json:"storeName"`
	Active              bool   `json:"active"`
}

type ActivateStoreRequest struct {
	StoreID     int64 `json:"storeId"`
	WarehouseID int64 `json:"warehouseId"`
}

type DeActivateStoreRequest struct {
	StoreID     int64 `json:"storeId"`
	WarehouseID int64 `json:"warehouseId"`
}

// GetStoreViewStatus — geçici kapalılık durumunu getirir
type GetStoreViewStatusRequest struct {
	StoreID int64 `json:"storeId"`
}

type StoreViewStatus struct {
	Status      string `json:"status"`      // "OPEN" | "CLOSED"
	OpeningTime int64  `json:"openingTime"` // unix timestamp
}

type StoreViewStatusResponse struct {
	Data    StoreViewStatus `json:"data"`
	Success bool            `json:"success"`
}

// Geçici kapatma süre seçenekleri
const (
	StoreOffDateOneHour   = "ONE_HOUR"
	StoreOffDateFourHour  = "FOUR_HOUR"
	StoreOffDateNextShift = "NEXT_SHIFT_START"
)

type AddStoreOffDateRequest struct {
	StoreID            int64  `json:"storeId"`
	StoreGroupID       int64  `json:"storeGroupId"`
	StoreOffDateOption string `json:"storeOffDateOption"`
}

type RemoveStoreOffDateRequest struct {
	StoreID      int64 `json:"storeId"`
	StoreGroupID int64 `json:"storeGroupId"`
}

// --- Menu ---

type MenuDetailsRequest struct {
	StoreID      int64 `json:"storeId"`
	StoreGroupID int64 `json:"storeGroupId"`
}

type FoodMenuItemDTO struct {
	ID                            int64   `json:"id"`
	MenuID                        int64   `json:"menuId"`
	ProductID                     int64   `json:"productId"`
	ProductName                   string  `json:"productName"`
	ProductDescription            string  `json:"productDescription"`
	DefaultPrimaryPrice           float64 `json:"defaultPrimaryPrice"`
	DefaultPrimaryDiscountedPrice float64 `json:"defaultPrimaryDiscountedPrice"`
	Status                        string  `json:"status"`
}

type MenuHeaderInfo struct {
	ID                      int64             `json:"id"`
	MenuID                  int64             `json:"menuId"`
	Name                    string            `json:"name"`
	Status                  string            `json:"status"`
	FoodMenuItemDetailsDTOs []FoodMenuItemDTO `json:"foodMenuItemDetailsDTOs"`
}

type MenuData struct {
	ID              int64            `json:"id"`
	StoreGroupID    int64            `json:"storeGroupId"`
	Name            string           `json:"name"`
	Status          string           `json:"status"`
	MenuHeaderInfos []MenuHeaderInfo `json:"menuHeaderInfos"`
}

type MenuResponse struct {
	Data    MenuData `json:"data"`
	Success bool     `json:"success"`
}

type UpdateProductStatusRequest struct {
	StoreID   int64  `json:"storeId"`
	ProductID int64  `json:"productId"`
	Status    string `json:"status"` // "ACTIVE" | "PASSIVE"
}

type PriceItem struct {
	MenuItemID                    int64   `json:"menuItemId"`
	ProductID                     int64   `json:"productId"`
	DefaultPrimaryPrice           float64 `json:"defaultPrimaryPrice"`
	DefaultPrimaryDiscountedPrice float64 `json:"defaultPrimaryDiscountedPrice"`
}

type UpdateBatchPriceRequest struct {
	StoreID      int64       `json:"storeId"`
	StoreGroupID int64       `json:"storeGroupId"`
	MenuID       int64       `json:"menuId"`
	Prices       []PriceItem `json:"prices"`
}

// OptionItem — opsiyon aktif/pasif
type UpdateOptionItemStatusRequest struct {
	StoreID      int64 `json:"storeId"`
	OptionItemID int64 `json:"optionItemId"`
}

// --- Working Hours ---

type WorkingHourItem struct {
	StartDay  string `json:"startDay"`  // PAZARTESİ, SALI, ÇARŞAMBA, PERŞEMBE, CUMA, CUMARTESİ, PAZAR
	StartHour string `json:"startHour"` // "11:00"
	EndHour   string `json:"endHour"`   // "22:00"
	Active    bool   `json:"active"`
}

type UpsertWorkingHoursRequest struct {
	StoreID     int64             `json:"storeId"`
	TimeSlotIds []WorkingHourItem `json:"timeSlotIds"`
}

type GetWorkingHoursRequest struct {
	StoreID int64 `json:"storeId"`
}

type WorkingHourDTO struct {
	StoreID       int64  `json:"storeId"`
	StartDay      string `json:"startDay"`
	StartHour     string `json:"startHour"`
	EndHour       string `json:"endHour"`
	Active        bool   `json:"active"`
	CreatedAt     int64  `json:"createdAt"`
	LastUpdatedAt int64  `json:"lastUpdatedAt"`
}

type WorkingHoursResponse struct {
	Data    []WorkingHourDTO `json:"data"`
	Success bool             `json:"success"`
}

// --- Payment Methods ---

type GetPaymentMethodsRequest struct {
	StoreID int64 `json:"storeId"`
}

type PaymentTypeDTO struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      bool   `json:"status"`
}

type PaymentMethods struct {
	OnlinePayments     []PaymentTypeDTO `json:"onlinePayments"`
	OnDeliveryPayments []PaymentTypeDTO `json:"onDeliveryPayments"`
}

type PaymentMethodsResponse struct {
	Data    PaymentMethods `json:"data"`
	Success bool           `json:"success"`
}

type PaymentStatusUpdate struct {
	Name   string `json:"name"`
	Status bool   `json:"status"`
}

type UpdatePaymentMethodsRequest struct {
	StoreID             int64                 `json:"storeId"`
	UpdatePaymentStatus []PaymentStatusUpdate `json:"updatePaymentStatus"`
}

// --- Cancel Reasons ---

type CancelReasonDTO struct {
	ReasonID    int64  `json:"reasonId"`
	Description string `json:"description"`
}

type CancelReasonsResponse struct {
	Data    []CancelReasonDTO `json:"data"`
	Success bool              `json:"success"`
}

// --- Orders ---

type UpdateOrderStatusRequest struct {
	OrderID        int64  `json:"orderId"`
	StoreID        int64  `json:"storeId"`
	OrderStatus    string `json:"orderStatus"`
	CancelReasonID int64  `json:"cancelReasonId,omitempty"`
}

type CancelOrderRequest struct {
	OrderID        int64 `json:"orderId"`
	StoreID        int64 `json:"storeId"`
	NotifyUser     bool  `json:"notifyUser"`
	CancelReasonID int64 `json:"cancelReasonId"`
}

// --- Webhook / Incoming Order ---

type IncomingOrder struct {
	ID               int64                    `json:"id"`
	Description      string                   `json:"description"`
	Status           string                   `json:"status"`
	DeliveryProvider string                   `json:"deliveryProvider"`
	Store            IncomingOrderStore        `json:"store"`
	Customer         IncomingOrderCustomer     `json:"customer"`
	Prices           IncomingOrderPrices       `json:"prices"`
	Items            []IncomingOrderItem       `json:"items"`
	Payment          IncomingOrderPayment      `json:"payment"`
	ExtendedProps    IncomingOrderExtendedProps `json:"extendedProperties"`
	Log              IncomingOrderLog          `json:"log"`
}

type IncomingOrderStore struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Group struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"group"`
}

type IncomingOrderCustomer struct {
	ID              int64                   `json:"id"`
	FirstName       string                  `json:"firstName"`
	LastName        string                  `json:"lastName"`
	FullName        string                  `json:"fullName"`
	PhoneNumber     string                  `json:"phoneNumber"`
	DeliveryAddress IncomingOrderAddress    `json:"deliveryAddress"`
}

type IncomingOrderAddress struct {
	ID        int64  `json:"id"`
	Direction string `json:"direction"`
	Detail    string `json:"detail"`
	City      struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"city"`
	Town struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"town"`
	District struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"district"`
	GeoLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"geoLocation"`
}

type IncomingOrderPriceAmount struct {
	AmountAsPenny int64  `json:"amountAsPenny"`
	Text          string `json:"text"`
}

type IncomingOrderPrices struct {
	Total               IncomingOrderPriceAmount `json:"total"`
	Discounted          IncomingOrderPriceAmount `json:"discounted"`
	RestaurantDiscounted IncomingOrderPriceAmount `json:"restaurantDiscounted"`
	MigrosDiscounted    IncomingOrderPriceAmount `json:"migrosDiscounted"`
}

type IncomingOrderItem struct {
	ID        int64                `json:"id"`
	ProductID int64                `json:"productId"`
	Name      string               `json:"name"`
	Price     int64                `json:"price"` // kuruş
	Amount    int                  `json:"amount"`
	Note      string               `json:"note"`
	Options   []IncomingOrderOption `json:"options"`
}

type IncomingOrderOption struct {
	OptionHeaderID   int64  `json:"optionHeaderId"`
	OptionItemID     int64  `json:"optionItemId"`
	HeaderName       string `json:"headerName"`
	ItemNames        string `json:"itemNames"`
	PrimaryPrice     int64  `json:"primaryPrice"` // kuruş
	Quantity         int    `json:"quantity"`
	Excluded         bool   `json:"excluded"`
}

type IncomingOrderPayment struct {
	Type struct {
		HashedNameID   int64  `json:"hashedNameId"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		IsOnlinePayment bool  `json:"isOnlinePayment"`
	} `json:"type"`
}

type IncomingOrderExtendedProps struct {
	OrderNote           string `json:"orderNote"`
	SaveGreen           bool   `json:"saveGreen"`
	ContactlessDelivery bool   `json:"contactlessDelivery"`
	RingDoorBell        bool   `json:"ringDoorBell"`
}

type IncomingOrderLog struct {
	CreatedAsMs int64 `json:"createdAsMs"`
}

// Sipariş durum sabitleri — Migros API v2 değerleri
const (
	OrderStatusApproved  = "Approved"
	OrderStatusRejected  = "Rejected"
	OrderStatusPrepared  = "Prepared"
	OrderStatusDelivery  = "Delivery"
	OrderStatusCompleted = "Completed"
)

// --- Common ---

type BaseResponse struct {
	Success      bool          `json:"success"`
	ErrorMessage *ErrorMessage `json:"errorMessage"`
}

type ErrorMessage struct {
	ErrorCode   string `json:"errorCode"`
	ErrorTitle  string `json:"errorTitle"`
	ErrorDetail string `json:"errorDetail"`
}
