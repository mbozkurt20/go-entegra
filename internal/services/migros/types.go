package migros

// --- Store ---

type StoreDetailResponse struct {
	Data    StoreDetail `json:"data"`
	Success bool        `json:"success"`
}

type StoreDetail struct {
	ID           int64  `json:"id"`
	WarehouseID  int64  `json:"warehouseId"`
	StoreGroupID int64  `json:"storeGroupId"`
	StoreName    string `json:"storeName"`
	Active       bool   `json:"active"`
}

type ActivateStoreRequest struct {
	StoreID     int64 `json:"storeId"`
	WarehouseID int64 `json:"warehouseId"`
}

type DeActivateStoreRequest struct {
	StoreID     int64 `json:"storeId"`
	WarehouseID int64 `json:"warehouseId"`
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
