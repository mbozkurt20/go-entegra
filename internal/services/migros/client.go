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
	secretKey    string
	storeID      int64
	storeGroupID int64
	httpClient   *http.Client
}

func NewClient(apiKey, secretKey string, storeID, storeGroupID int64, env string) *Client {
	baseURL := BaseURLProd
	if env != "prod" && env != "1" {
		baseURL = BaseURLTest
	}
	return &Client{
		baseURL:      baseURL,
		apiKey:       apiKey,
		secretKey:    secretKey,
		storeID:      storeID,
		storeGroupID: storeGroupID,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func NewClientFromInfo(info map[string]interface{}) (*Client, error) {
	apiKey, _ := info["apiKey"].(string)
	secretKey, _ := info["secretKey"].(string)
	env, _ := info["env"].(string)

	// storeId falls back to restaurantId
	storeIDStr, _ := info["storeId"].(string)
	if storeIDStr == "" {
		storeIDStr, _ = info["restaurantId"].(string)
	}

	// storeGroupId falls back to chainId
	storeGroupIDStr, _ := info["storeGroupId"].(string)
	if storeGroupIDStr == "" {
		storeGroupIDStr, _ = info["chainId"].(string)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("apiKey bilgisi eksik")
	}

	storeID, err := strconv.ParseInt(storeIDStr, 10, 64)
	if err != nil || storeID == 0 {
		return nil, fmt.Errorf("storeId/restaurantId bilgisi eksik veya geçersiz")
	}

	storeGroupID, err := strconv.ParseInt(storeGroupIDStr, 10, 64)
	if err != nil || storeGroupID == 0 {
		return nil, fmt.Errorf("storeGroupId/chainId bilgisi eksik veya geçersiz")
	}

	return NewClient(apiKey, secretKey, storeID, storeGroupID, env), nil
}

// SecretKey returns the AES secret key for webhook verification
func (c *Client) SecretKey() string {
	return c.secretKey
}

// encryptionKey returns the AES key to use: secretKey if set, otherwise apiKey
func (c *Client) encryptionKey() string {
	if c.secretKey != "" {
		return c.secretKey
	}
	return c.apiKey
}

// marshalSpaced encodes v as JSON with spaces after ':' and ',' (Python json.dumps style).
// Migros API's .NET decryptor requires this exact byte layout for valid PKCS7 padding.
func marshalSpaced(v interface{}) ([]byte, error) {
	compact, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	inString := false
	escaped := false
	for _, c := range compact {
		if escaped {
			buf.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' && inString {
			buf.WriteByte(c)
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
		}
		buf.WriteByte(c)
		if !inString && (c == ':' || c == ',') {
			buf.WriteByte(' ')
		}
	}
	return buf.Bytes(), nil
}

func (c *Client) do(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		plain, err := marshalSpaced(body)
		if err != nil {
			return nil, err
		}
		encrypted, err := AESEncrypt(c.encryptionKey(), plain)
		if err != nil {
			return nil, fmt.Errorf("migros istek şifreleme hatası: %w", err)
		}
		wrapped, _ := json.Marshal(map[string]string{"value": encrypted})
		bodyReader = bytes.NewReader(wrapped)
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

// GetOrder sipariş detayını getirir
func (c *Client) GetOrder(orderID int64) (*IncomingOrder, error) {
	path := fmt.Sprintf("/Order/v2/GetOrderDetail?orderId=%d&storeId=%d", orderID, c.storeID)
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	type orderResp struct {
		Data    IncomingOrder `json:"data"`
		Success bool          `json:"success"`
	}
	var result orderResp
	if err := decodeAndCheck(resp, "sipariş detay", &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
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

// GetCancelReasons iptal sebeplerini getirir (şifreleme yok — GET)
func (c *Client) GetCancelReasons() ([]CancelReasonDTO, error) {
	resp, err := c.do("GET", "/Mapping/v2/GetCancelReasons", nil)
	if err != nil {
		return nil, err
	}
	var result CancelReasonsResponse
	if err := decodeAndCheck(resp, "iptal sebepleri", &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetStoreViewStatus mağazanın geçici kapalılık durumunu getirir
func (c *Client) GetStoreViewStatus() (*StoreViewStatus, error) {
	resp, err := c.do("POST", "/Store/GetStoreViewStatus", GetStoreViewStatusRequest{
		StoreID: c.storeID,
	})
	if err != nil {
		return nil, err
	}
	var result StoreViewStatusResponse
	if err := decodeAndCheck(resp, "mağaza görünüm durumu", &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// AddStoreOffDate mağazayı geçici kapatır
func (c *Client) AddStoreOffDate(option string) error {
	resp, err := c.do("POST", "/Store/AddStoreOffDate", AddStoreOffDateRequest{
		StoreID:            c.storeID,
		StoreGroupID:       c.storeGroupID,
		StoreOffDateOption: option,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "geçici kapatma")
}

// RemoveStoreOffDate geçici kapatmayı kaldırır
func (c *Client) RemoveStoreOffDate() error {
	resp, err := c.do("POST", "/Store/RemoveStoreOffDate", RemoveStoreOffDateRequest{
		StoreID:      c.storeID,
		StoreGroupID: c.storeGroupID,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "geçici kapatma kaldırma")
}

// GetWorkingHours çalışma saatlerini getirir
func (c *Client) GetWorkingHours() ([]WorkingHourDTO, error) {
	resp, err := c.do("POST", "/WorkingHour/GetStoreTimeSlot", GetWorkingHoursRequest{
		StoreID: c.storeID,
	})
	if err != nil {
		return nil, err
	}
	var result WorkingHoursResponse
	if err := decodeAndCheck(resp, "çalışma saatleri", &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// UpdateWorkingHours çalışma saatlerini günceller
func (c *Client) UpdateWorkingHours(slots []WorkingHourItem) error {
	resp, err := c.do("POST", "/WorkingHour/UpsertStoreTimeSlot", UpsertWorkingHoursRequest{
		StoreID:     c.storeID,
		TimeSlotIds: slots,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "çalışma saatleri güncelleme")
}

// GetPaymentMethods ödeme yöntemlerini getirir
func (c *Client) GetPaymentMethods() (*PaymentMethods, error) {
	resp, err := c.do("POST", "/PaymentMethod/GetPaymentMethodsByStoreId", GetPaymentMethodsRequest{
		StoreID: c.storeID,
	})
	if err != nil {
		return nil, err
	}
	var result PaymentMethodsResponse
	if err := decodeAndCheck(resp, "ödeme yöntemleri", &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdatePaymentMethods ödeme yöntemlerini günceller
func (c *Client) UpdatePaymentMethods(updates []PaymentStatusUpdate) error {
	resp, err := c.do("POST", "/PaymentMethod/UpdatePaymentMethodStatus", UpdatePaymentMethodsRequest{
		StoreID:             c.storeID,
		UpdatePaymentStatus: updates,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "ödeme yöntemi güncelleme")
}

// UpdateOptionItemStatus opsiyon durumunu günceller
func (c *Client) UpdateOptionItemStatus(optionItemID int64, activate bool) error {
	path := "/OptionItem/DeActivateOptionItemStatusByStore"
	if activate {
		path = "/OptionItem/ActivateOptionItemStatusByStore"
	}
	resp, err := c.do("POST", path, UpdateOptionItemStatusRequest{
		StoreID:      c.storeID,
		OptionItemID: optionItemID,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "opsiyon durum güncelleme")
}
