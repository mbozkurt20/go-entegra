package yemeksepeti

import "time"

// ---- Middleware API Auth ----

type LoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	GrantType string `json:"grant_type"` // "client_credentials"
}

type LoginResponse struct {
	AccessToken string  `json:"access_token"`
	ExpiresIn   float64 `json:"expires_in"` // seconds
	TokenType   string  `json:"token_type"`
}

// ---- Order Status Update ----

type OrderStatusRequest struct {
	Status         string  `json:"status"`
	AcceptanceTime *string `json:"acceptanceTime,omitempty"` // RFC3339, for order_accepted
	RemoteOrderId  *string `json:"remoteOrderId,omitempty"`
	Reason         *string `json:"reason,omitempty"`  // for order_rejected
	Message        *string `json:"message,omitempty"` // for order_rejected
}

type OrderStatusResponse struct {
	Message string `json:"message"`
}

// Reject reasons (enum)
const (
	RejectReasonItemUnavailable    = "ITEM_UNAVAILABLE"
	RejectReasonTechnicalProblem   = "TECHNICAL_PROBLEM"
	RejectReasonTooBusy            = "TOO_BUSY"
	RejectReasonClosed             = "CLOSED"
	RejectReasonNoResponse         = "NO_RESPONSE"
	RejectReasonTestOrder          = "TEST_ORDER"
	RejectReasonBadWeather         = "BAD_WEATHER"
	RejectReasonNoCourier          = "NO_COURIER"
)

// Order status values
const (
	StatusOrderAccepted  = "order_accepted"
	StatusOrderRejected  = "order_rejected"
	StatusOrderPickedUp  = "order_picked_up"
)

// ---- Availability ----

type AvailabilityUpdateRequest struct {
	AvailabilityState string  `json:"availabilityState"`         // OPEN | CLOSED | CLOSED_UNTIL
	ClosedReason      *string `json:"closedReason,omitempty"`    // TOO_BUSY_KITCHEN | TECHNICAL_PROBLEM | ...
	ClosingMinutes    *int    `json:"closingMinutes,omitempty"`  // minutes until reopen (CLOSED_UNTIL)
}

type AvailabilityEntry struct {
	AvailabilityState   string   `json:"availabilityState"`
	AvailabilityStates  []string `json:"availabilityStates"`
	Changeable          bool     `json:"changeable"`
	ClosedReason        string   `json:"closedReason"`
	ClosingMinutes      []int    `json:"closingMinutes"`
	ClosingReasons      []string `json:"closingReasons"`
	PlatformID          string   `json:"platformId"`
	PlatformKey         string   `json:"platformKey"`
	PlatformRestaurantID string  `json:"platformRestaurantId"`
	PlatformType        string   `json:"platformType"`
}

// ---- POS Reachability ----

type ReachabilityRequest struct {
	PosReachabilityStatus string `json:"posReachabilityStatus"` // REACHABLE | UNREACHABLE
}

// ---- Incoming Order (Plugin API — DH sends this to us) ----

type IncomingOrder struct {
	Token           string          `json:"token"`           // DH unique order ID (orderToken for callbacks)
	Code            string          `json:"code"`            // Platform order code e.g. "n0s1-w0k1"
	ShortCode       string          `json:"shortCode"`       // Rider-friendly short code
	Comments        OrderComments   `json:"comments"`
	CreatedAt       time.Time       `json:"createdAt"`
	ExpiryDate      time.Time       `json:"expiryDate"`
	ExpeditionType  string          `json:"expeditionType"`  // "pickup" | "delivery"
	Test            bool            `json:"test"`
	PreOrder        bool            `json:"preOrder"`
	Customer        Customer        `json:"customer"`
	Delivery        *Delivery       `json:"delivery"`
	Pickup          *PickupInfo     `json:"pickup"`
	Payment         Payment         `json:"payment"`
	Price           Price           `json:"price"`
	Products        []Product       `json:"products"`
	Discounts       []Discount      `json:"discounts"`
	Vouchers        []interface{}   `json:"vouchers"`
	PlatformRestaurant PlatformRestaurant `json:"platformRestaurant"`
	LocalInfo       LocalInfo       `json:"localInfo"`
	CallbackUrls    *CallbackUrls   `json:"callbackUrls"`
	PreparationTimeAdjustments *PreparationTimeAdjustments `json:"preparationTimeAdjustments"`
	CorporateTaxId  string          `json:"corporateTaxId"`
}

type OrderComments struct {
	CustomerComment string `json:"customerComment"`
}

type Customer struct {
	Email             string   `json:"email"`
	FirstName         string   `json:"firstName"`
	LastName          string   `json:"lastName"`
	MobilePhone       string   `json:"mobilePhone"`
	Flags             []string `json:"flags"`
}

type Delivery struct {
	Address              *DeliveryAddress `json:"address"`
	ExpectedDeliveryTime string           `json:"expectedDeliveryTime"`
	ExpressDelivery      bool             `json:"expressDelivery"`
	RiderPickupTime      string           `json:"riderPickupTime"`
}

type DeliveryAddress struct {
	Building            string  `json:"building"`
	City                string  `json:"city"`
	Street              string  `json:"street"`
	Number              string  `json:"number"`
	FlatNumber          string  `json:"flatNumber"`
	Floor               string  `json:"floor"`
	Postcode            string  `json:"postcode"`
	DeliveryArea        string  `json:"deliveryArea"`
	DeliveryMainArea    string  `json:"deliveryMainArea"`
	DeliveryInstructions string `json:"deliveryInstructions"`
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
}

type PickupInfo struct {
	PickupTime string `json:"pickupTime"`
	PickupCode string `json:"pickupCode"`
}

type Payment struct {
	Status string `json:"status"` // "pending" | "paid"
	Type   string `json:"type"`
}

type Price struct {
	GrandTotal            string         `json:"grandTotal"`
	TotalNet              string         `json:"totalNet"`
	VatTotal              string         `json:"vatTotal"`
	PayRestaurant         string         `json:"payRestaurant"`
	RiderTip              *string        `json:"riderTip"`
	CollectFromCustomer   string         `json:"collectFromCustomer"`
	DeliveryFees          []DeliveryFee  `json:"deliveryFees"`
	MinimumDeliveryValue  string         `json:"minimumDeliveryValue"`
	SubTotal              string         `json:"subTotal"`
}

type DeliveryFee struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type Product struct {
	ID                       string    `json:"id"`
	Name                     string    `json:"name"`
	CategoryName             string    `json:"categoryName"`
	PaidPrice                string    `json:"paidPrice"`
	UnitPrice                string    `json:"unitPrice"`
	Quantity                 string    `json:"quantity"`
	RemoteCode               string    `json:"remoteCode"`
	SKU                      string    `json:"sku"`
	Comment                  string    `json:"comment"`
	ItemUnavailabilityHandling string  `json:"itemUnavailabilityHandling"`
	Variation                *Variation `json:"variation"`
	SelectedToppings         []Topping `json:"selectedToppings"`
	Discounts                []Discount `json:"discounts"`
}

type Variation struct {
	Name string `json:"name"`
}

type Topping struct {
	ID                       string    `json:"id"`
	Name                     string    `json:"name"`
	Price                    string    `json:"price"`
	Quantity                 float64   `json:"quantity"`
	RemoteCode               string    `json:"remoteCode"`
	SKU                      string    `json:"sku"`
	Type                     string    `json:"type"` // PRODUCT | VARIANT | EXTRA
	ItemUnavailabilityHandling string  `json:"itemUnavailabilityHandling"`
	Discounts                []Discount `json:"discounts"`
	Children                 []Topping  `json:"children"`
}

type Discount struct {
	Name         string        `json:"name"`
	Amount       string        `json:"amount"`
	Sponsorships []Sponsorship `json:"sponsorships"`
}

type Sponsorship struct {
	Sponsor string `json:"sponsor"` // PLATFORM | VENDOR | THIRD_PARTY
	Amount  string `json:"amount"`
}

type PlatformRestaurant struct {
	ID string `json:"id"`
}

type LocalInfo struct {
	CountryCode    string `json:"countryCode"`
	CurrencySymbol string `json:"currencySymbol"`
	Platform       string `json:"platform"`
	PlatformKey    string `json:"platformKey"`
}

type CallbackUrls struct {
	OrderAcceptedUrl              string `json:"orderAcceptedUrl"`
	OrderRejectedUrl              string `json:"orderRejectedUrl"`
	OrderProductModificationUrl   string `json:"orderProductModificationUrl"`
	OrderPickedUpUrl              string `json:"orderPickedUpUrl"`
	OrderPreparedUrl              string `json:"orderPreparedUrl"`
	OrderPreparationTimeAdjustmentUrl string `json:"orderPreparationTimeAdjustmentUrl"`
}

type PreparationTimeAdjustments struct {
	MaxPickUpTimestamp                    string  `json:"maxPickUpTimestamp"`
	MinPickUpTimestamp                    string  `json:"minPickUpTimestamp"`
	PreparationTimeChangeIntervalsInMinutes []int `json:"preparationTimeChangeIntervalsInMinutes"`
}

// ---- Plugin API Responses (we send these back to DH) ----

// 200 — order acknowledged synchronously
type OrderDispatchAcknowledgedResponse struct {
	RemoteResponse RemoteResponseAck `json:"remoteResponse"`
}

type RemoteResponseAck struct {
	RemoteOrderID string `json:"remoteOrderId"`
}

// 202 — order accepted asynchronously (direct flow)
type OrderDispatchAcceptedResponse struct {
	RemoteResponse RemoteResponseAccepted `json:"remoteResponse"`
}

type RemoteResponseAccepted struct {
	RemoteOrderID  string `json:"remoteOrderId"`
	AcceptanceTime string `json:"acceptanceTime"` // RFC3339, at least 2 min in future
}

// 400 — order rejected
type OrderDispatchRejectResponse struct {
	Reason  string `json:"reason"`
	Message string `json:"message,omitempty"`
}

// ---- Adjust Preparation Time ----

type AdjustPreparationTimeRequest struct {
	ExpectedPickupAt string `json:"expectedPickupAt"` // RFC3339
}
