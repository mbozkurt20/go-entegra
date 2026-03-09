package migros

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	BaseURLProd = "https://gourmet.migrosonline.com"
	BaseURLTest = "https://test.gourmet.migrosonline.com"
)

type Client struct {
	baseURL      string
	apiKey       string
	storeID      int64
	storeGroupID int64
	httpClient   *http.Client
}

func NewClient(apiKey string, storeID, storeGroupID int64, env string) *Client {
	baseURL := BaseURLProd
	if env != "prod" && env != "1" {
		baseURL = BaseURLTest
	}
	return &Client{
		baseURL:      baseURL,
		apiKey:       apiKey,
		storeID:      storeID,
		storeGroupID: storeGroupID,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func NewClientFromInfo(info map[string]interface{}) (*Client, error) {
	apiKey, _ := info["apiKey"].(string)
	storeIDStr, _ := info["storeId"].(string)
	storeGroupIDStr, _ := info["storeGroupId"].(string)
	env, _ := info["env"].(string)

	if apiKey == "" {
		return nil, fmt.Errorf("apiKey bilgisi eksik")
	}

	storeID, err := strconv.ParseInt(storeIDStr, 10, 64)
	if err != nil || storeID == 0 {
		return nil, fmt.Errorf("storeId bilgisi eksik veya geçersiz")
	}

	storeGroupID, err := strconv.ParseInt(storeGroupIDStr, 10, 64)
	if err != nil || storeGroupID == 0 {
		return nil, fmt.Errorf("storeGroupId bilgisi eksik veya geçersiz")
	}

	return NewClient(apiKey, storeID, storeGroupID, env), nil
}

func (c *Client) do(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header["XApiKey"] = []string{c.apiKey} // Go normalizes Set() to "Xapikey", bypass it
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func checkResp(resp *http.Response, action string) error {
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("migros %s hatası %d: %s", action, resp.StatusCode, string(b))
	}
	return nil
}

func decodeAndCheck(resp *http.Response, action string, result interface{}) error {
	defer resp.Body.Close()
	if err := checkResp(resp, action); err != nil {
		return err
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("migros %s decode hatası: %w", action, err)
	}
	return nil
}

// GetStoreDetail mağaza detayını getirir (warehouseId ve storeGroupId için)
func (c *Client) GetStoreDetail() (*StoreDetail, error) {
	resp, err := c.do("GET", "/Store/GetStoreDetail", nil)
	if err != nil {
		return nil, err
	}
	var result StoreDetailResponse
	if err := decodeAndCheck(resp, "store detay", &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// SetStoreStatus mağazayı açar veya kapatır
func (c *Client) SetStoreStatus(isOpen bool) error {
	store, err := c.GetStoreDetail()
	if err != nil {
		return err
	}

	if isOpen {
		resp, err := c.do("POST", "/Store/ActivateStore", ActivateStoreRequest{
			StoreID:     c.storeID,
			WarehouseID: store.WarehouseID,
		})
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return checkResp(resp, "mağaza açma")
	}

	resp, err := c.do("POST", "/Store/DeActivateStore", DeActivateStoreRequest{
		StoreID:     c.storeID,
		WarehouseID: store.WarehouseID,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "mağaza kapatma")
}

// GetMenu menü detayını getirir
func (c *Client) GetMenu() (*MenuData, error) {
	resp, err := c.do("POST", "/Menu/GetMenuDetailsByStoreAndStoreGroupId", MenuDetailsRequest{
		StoreID:      c.storeID,
		StoreGroupID: c.storeGroupID,
	})
	if err != nil {
		return nil, err
	}
	var result MenuResponse
	if err := decodeAndCheck(resp, "menü çekme", &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateProductStatus ürün durumunu günceller
func (c *Client) UpdateProductStatus(productID int64, available bool) error {
	status := "PASSIVE"
	if available {
		status = "ACTIVE"
	}
	resp, err := c.do("POST", "/Menu/UpdateProductStatusByStoreId", UpdateProductStatusRequest{
		StoreID:   c.storeID,
		ProductID: productID,
		Status:    status,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "ürün durum güncelleme")
}

// UpdateProductPrice ürün fiyatını günceller
func (c *Client) UpdateProductPrice(menuItemID, productID int64, price float64) error {
	menu, err := c.GetMenu()
	if err != nil {
		return err
	}

	resp, err := c.do("POST", "/Menu/UpdateBatchPrice", UpdateBatchPriceRequest{
		StoreID:      c.storeID,
		StoreGroupID: c.storeGroupID,
		MenuID:       menu.ID,
		Prices: []PriceItem{
			{
				MenuItemID:                    menuItemID,
				ProductID:                     productID,
				DefaultPrimaryPrice:           price,
				DefaultPrimaryDiscountedPrice: price,
			},
		},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "ürün fiyat güncelleme")
}

// UpdateOrderStatus sipariş durumunu günceller
func (c *Client) UpdateOrderStatus(orderID int64, status string) error {
	resp, err := c.do("POST", "/Order/v2/UpdateOrderStatus", UpdateOrderStatusRequest{
		OrderID:     orderID,
		StoreID:     c.storeID,
		OrderStatus: status,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş durum güncelleme")
}

// CancelOrder siparişi iptal eder
func (c *Client) CancelOrder(orderID, cancelReasonID int64, notifyUser bool) error {
	resp, err := c.do("POST", "/Order/v2/CancelOrder", CancelOrderRequest{
		OrderID:        orderID,
		StoreID:        c.storeID,
		NotifyUser:     notifyUser,
		CancelReasonID: cancelReasonID,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş iptal")
}
