package trendyol

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURLProd  = "https://api.tgoapis.com"
	BaseURLStage = "https://stageapi.tgoapis.com"
)

type Client struct {
	baseURL      string
	supplierID   string
	storeID      string
	apiKey       string
	apiSecretKey string
	httpClient   *http.Client
}

func NewClient(supplierID, storeID, apiKey, apiSecretKey, env string) *Client {
	baseURL := BaseURLProd
	if env == "stage" || env == "0" {
		baseURL = BaseURLStage
	}
	return &Client{
		baseURL:      baseURL,
		supplierID:   supplierID,
		storeID:      storeID,
		apiKey:       apiKey,
		apiSecretKey: apiSecretKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func NewClientFromInfo(info map[string]interface{}) (*Client, error) {
	supplierID, _ := info["supplierId"].(string)
	// storeId öncelikli, yoksa restaurantId (eski alan adı) dene
	storeID, _ := info["storeId"].(string)
	if storeID == "" {
		storeID, _ = info["restaurantId"].(string)
	}
	apiKey, _ := info["apiKey"].(string)
	apiSecretKey, _ := info["apiSecretKey"].(string)
	env, _ := info["env"].(string)

	if supplierID == "" {
		return nil, fmt.Errorf("supplierId bilgisi eksik")
	}
	if storeID == "" {
		return nil, fmt.Errorf("storeId bilgisi eksik")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey bilgisi eksik")
	}
	if apiSecretKey == "" {
		return nil, fmt.Errorf("apiSecretKey bilgisi eksik")
	}
	return NewClient(supplierID, storeID, apiKey, apiSecretKey, env), nil
}

func (c *Client) authHeader() string {
	creds := c.apiKey + ":" + c.apiSecretKey
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
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
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func checkResp(resp *http.Response, action string) error {
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("trendyol %s hatası %d: %s", action, resp.StatusCode, string(b))
	}
	return nil
}

// --- Restoran ---

func (c *Client) GetStoreInfo() (*StoreInfo, error) {
	path := fmt.Sprintf("/integrator/store/meal/suppliers/%s/stores/%s", c.supplierID, c.storeID)
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "mağaza bilgisi"); err != nil {
		return nil, err
	}
	var wrapper struct {
		Data StoreInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("mağaza bilgisi decode hatası: %w", err)
	}
	return &wrapper.Data, nil
}

func (c *Client) SetRestaurantStatus(isOpen bool) error {
	path := fmt.Sprintf("/integrator/store/meal/suppliers/%s/stores/%s/status", c.supplierID, c.storeID)
	resp, err := c.do("PUT", path, RestaurantStatusRequest{IsOpen: isOpen})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "restoran durumu")
}

func (c *Client) UpdateWorkingHours(slots []WorkingHoursSlot) error {
	path := fmt.Sprintf("/integrator/store/meal/suppliers/%s/stores/%s/working-hours", c.supplierID, c.storeID)
	resp, err := c.do("PUT", path, WorkingHoursRequest{WorkingHours: slots})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "çalışma saatleri")
}

func (c *Client) UpdateDeliveryTime(minutes int) error {
	path := fmt.Sprintf("/integrator/store/meal/suppliers/%s/stores/%s/delivery-time", c.supplierID, c.storeID)
	resp, err := c.do("PUT", path, DeliveryTimeRequest{DeliveryTime: minutes})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "teslimat süresi")
}

func (c *Client) UpdateDeliveryZones(zones []DeliveryZoneRequest) error {
	path := fmt.Sprintf("/integrator/store/meal/suppliers/%s/stores/%s/delivery-zones", c.supplierID, c.storeID)
	resp, err := c.do("PUT", path, DeliveryZonesRequest{DeliveryZones: zones})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "teslimat bölgeleri")
}

// --- Menü / Bölüm ---

func (c *Client) UpdateSectionStatus(sectionID string, available bool) error {
	path := fmt.Sprintf("/integrator/product/meal/suppliers/%s/stores/%s/sections/%s/status",
		c.supplierID, c.storeID, sectionID)
	status := "PASSIVE"
	if available {
		status = "ACTIVE"
	}
	resp, err := c.do("PUT", path, SectionStatusRequest{Status: status})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "kategori durum güncelleme")
}

func (c *Client) GetBatchStatus(batchID string) (*BatchStatusResponse, error) {
	path := fmt.Sprintf("/integrator/product/meal/suppliers/%s/stores/%s/batch-requests/%s",
		c.supplierID, c.storeID, batchID)
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "batch durum"); err != nil {
		return nil, err
	}
	var result BatchStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("batch decode hatası: %w", err)
	}
	return &result, nil
}

func (c *Client) GetMenu() (*MenuResponse, error) {
	path := fmt.Sprintf("/integrator/product/meal/suppliers/%s/stores/%s/products", c.supplierID, c.storeID)
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResp(resp, "menü çekme"); err != nil {
		return nil, err
	}

	var result MenuResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("menü decode hatası: %w", err)
	}
	return &result, nil
}

func (c *Client) UpdateItemStatus(itemID string, available bool) error {
	path := fmt.Sprintf("/integrator/product/meal/suppliers/%s/stores/%s/products/%s/status",
		c.supplierID, c.storeID, itemID)
	status := "PASSIVE"
	if available {
		status = "ACTIVE"
	}
	resp, err := c.do("PUT", path, UpdateItemStatusRequest{Status: status})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "ürün durum güncelleme")
}

func (c *Client) UpdateItemPrice(itemID string, price float64) error {
	path := fmt.Sprintf("/integrator/product/meal/suppliers/%s/stores/%s/products/%s/price",
		c.supplierID, c.storeID, itemID)
	resp, err := c.do("PUT", path, UpdateItemPriceRequest{Price: price})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "ürün fiyat güncelleme")
}

// --- Siparişler ---

func (c *Client) UpdateOrderStatus(orderID string, status string) error {
	path := fmt.Sprintf("/integrator/order/meal/suppliers/%s/orders/%s/status", c.supplierID, orderID)
	resp, err := c.do("PUT", path, OrderStatusRequest{Status: status})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş durum güncelleme")
}

func (c *Client) CancelOrder(orderID string, reasonID int, note string) error {
	path := fmt.Sprintf("/integrator/order/meal/suppliers/%s/orders/%s/cancel", c.supplierID, orderID)
	resp, err := c.do("PUT", path, CancelOrderRequest{
		CancelReasonID:   reasonID,
		CancelReasonNote: note,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş iptal")
}
