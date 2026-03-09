package getir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	BaseURLProduction  = "https://food-external-api-gateway.getirapi.com"
	BaseURLDevelopment = "https://food-external-api-gateway.development.getirapi.com"
)

type Client struct {
	baseURL             string
	appSecretKey        string
	restaurantSecretKey string
	httpClient          *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

// NewClient creates a Getir API client.
// restaurantID maps to Getir's appSecretKey, secretKey maps to restaurantSecretKey.
func NewClient(restaurantID, secretKey, service string) *Client {
	baseURL := BaseURLProduction
	if service != "1" {
		baseURL = BaseURLDevelopment
	}
	return &Client{
		baseURL:             baseURL,
		appSecretKey:        restaurantID,
		restaurantSecretKey: secretKey,
		httpClient:          &http.Client{Timeout: 15 * time.Second},
	}
}

// GetToken token alır, süresi dolmamışsa cache'den döner
func (c *Client) GetToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	body, _ := json.Marshal(TokenRequest{
		AppSecretKey:        c.appSecretKey,
		RestaurantSecretKey: c.restaurantSecretKey,
	})

	resp, err := c.httpClient.Post(c.baseURL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("getir token isteği başarısız: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("getir token hatası %d: %s", resp.StatusCode, string(b))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("getir token decode hatası: %w", err)
	}

	c.token = tokenResp.Token
	c.tokenExpiry = time.Now().Add(55 * time.Minute) // API expiry bilgisi dönmüyor, 55dk cache
	return c.token, nil
}

func (c *Client) do(method, path string, body interface{}) (*http.Response, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("token", token)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func checkResp(resp *http.Response, action string) error {
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("getir %s hatası %d: %s", action, resp.StatusCode, string(b))
	}
	return nil
}

// --- Sipariş ---

func (c *Client) ApproveOrder(orderID string) error {
	resp, err := c.do("POST", fmt.Sprintf("/food-orders/%s/verify", orderID), VerifyOrderRequest{Status: "Approved"})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş onay")
}

func (c *Client) PrepareOrder(orderID string) error {
	resp, err := c.do("POST", fmt.Sprintf("/food-orders/%s/prepare", orderID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş hazırlama")
}

func (c *Client) HandoverOrder(orderID string) error {
	resp, err := c.do("POST", fmt.Sprintf("/food-orders/%s/handover", orderID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş kurye teslim")
}

func (c *Client) DeliverOrder(orderID string) error {
	resp, err := c.do("POST", fmt.Sprintf("/food-orders/%s/deliver", orderID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş müşteri teslim")
}

func (c *Client) CancelOrder(orderID string, reasonID int, note string) error {
	resp, err := c.do("POST", fmt.Sprintf("/food-orders/%s/cancel", orderID), CancelOrderRequest{
		CancelReasonID: reasonID,
		CancelNote:     note,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş iptal")
}

// InquireOrder Getir'den sipariş detaylarını sorgular
func (c *Client) InquireOrder(orderID string) (*OrderInquiryResponse, error) {
	resp, err := c.do("GET", fmt.Sprintf("/food-orders/%s", orderID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "sipariş sorgulama"); err != nil {
		return nil, err
	}
	var result OrderInquiryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("sipariş sorgulama decode hatası: %w", err)
	}
	return &result, nil
}

// ApproveScheduledOrder ileri tarihli siparişi onaylar
func (c *Client) ApproveScheduledOrder(orderID string) error {
	resp, err := c.do("POST", fmt.Sprintf("/food-orders/%s/verify-scheduled", orderID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "ileri tarihli sipariş onay")
}

// GetCancelOptions siparişin iptal nedenlerini getirir
func (c *Client) GetCancelOptions(orderID string) ([]CancelOption, error) {
	resp, err := c.do("GET", fmt.Sprintf("/food-orders/%s/cancel-options", orderID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "iptal nedenleri"); err != nil {
		return nil, err
	}
	var result []CancelOption
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("iptal nedenleri decode hatası: %w", err)
	}
	return result, nil
}

// --- Restoran ---

// GetRestaurantInfo restoran bilgilerini getirir
func (c *Client) GetRestaurantInfo() (*RestaurantInfo, error) {
	resp, err := c.do("GET", "/restaurants", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "restoran bilgisi"); err != nil {
		return nil, err
	}
	var result RestaurantInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("restoran bilgisi decode hatası: %w", err)
	}
	return &result, nil
}

// SetRestaurantStatus restoranı Getir'de açar veya kapatır.
// timeOffAmount: kapatma süresi (0=süresiz, 15/30/45=geçici).
func (c *Client) SetRestaurantStatus(isOpen bool, timeOffAmount ...int) error {
	if isOpen {
		resp, err := c.do("PUT", "/restaurants/status/open", nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return checkResp(resp, "restoran açma")
	}
	var body interface{}
	if len(timeOffAmount) > 0 && timeOffAmount[0] > 0 {
		body = CloseRestaurantRequest{TimeOffAmount: timeOffAmount[0]}
	}
	resp, err := c.do("PUT", "/restaurants/status/close", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "restoran kapatma")
}

// SetBusyness restoranın yoğunluk durumunu günceller
func (c *Client) SetBusyness(isBusy bool, durationMin int) error {
	body := BusynessRequest{IsBusy: isBusy, BusynessDifferenceDuration: durationMin}
	resp, err := c.do("PUT", "/restaurants/delivery-duration/busyness", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "yoğunluk güncelleme")
}

// EnableCourier kurye servisini etkinleştirir
func (c *Client) EnableCourier() error {
	resp, err := c.do("POST", "/restaurants/courier/enable", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "kurye etkinleştirme")
}

// DisableCourier kurye servisini devre dışı bırakır (timeOffAmount: 15/30/45)
func (c *Client) DisableCourier(timeOffAmount int) error {
	resp, err := c.do("POST", "/restaurants/courier/disable", CourierDisableRequest{TimeOffAmount: timeOffAmount})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "kurye devre dışı")
}

// GetWorkingHours çalışma saatlerini getirir
func (c *Client) GetWorkingHours() (*WorkingHoursResponse, error) {
	resp, err := c.do("GET", "/restaurants/working-hours", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "çalışma saatleri"); err != nil {
		return nil, err
	}
	var result WorkingHoursResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("çalışma saatleri decode hatası: %w", err)
	}
	return &result, nil
}

// SetWorkingHours çalışma saatlerini günceller (7 günlük dizi)
func (c *Client) SetWorkingHours(days []WorkingHourDay) error {
	body := struct {
		RestaurantWorkingHours []WorkingHourDay `json:"restaurantWorkingHours"`
	}{RestaurantWorkingHours: days}
	resp, err := c.do("PUT", "/restaurants/working-hours", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "çalışma saatleri güncelleme")
}

// SetPosStatus restoranın POS entegrasyonunu açar/kapatır.
// Kapalı statüsüyle gelen POS entegrasyonunu aktif hale getirmek için kullanılır.
// Bu endpoint token gerektirmez; appSecretKey + restaurantSecretKey + posStatus (100=açık, 200=kapalı) gönderilir.
func (c *Client) SetPosStatus(isOpen bool) error {
	type posStatusReq struct {
		AppSecretKey        string `json:"appSecretKey"`
		RestaurantSecretKey string `json:"restaurantSecretKey"`
		PosStatus           int    `json:"posStatus"` // 100=açık, 200=kapalı
	}
	status := 200
	if isOpen {
		status = 100
	}
	body := posStatusReq{
		AppSecretKey:        c.appSecretKey,
		RestaurantSecretKey: c.restaurantSecretKey,
		PosStatus:           status,
	}
	// Bu endpoint token gerektirmez, doğrudan http client kullan
	b, _ := json.Marshal(body)
	resp, err := c.httpClient.Do(func() *http.Request {
		req, _ := http.NewRequest("PUT", c.baseURL+"/restaurants/pos-status", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		return req
	}())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "POS durumu")
}

// --- Menü ---

// GetMenu restoranın tüm ürünlerini ve kategorilerini Getir'den çeker
func (c *Client) GetMenu() (*MenuResponse, error) {
	resp, err := c.do("GET", "/restaurants/menu", nil)
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

// GetCategories restoran kategorilerini çeker
func (c *Client) GetCategories() ([]MenuCategory, error) {
	menu, err := c.GetMenu()
	if err != nil {
		return nil, err
	}
	return menu.ProductCategories, nil
}

// UpdateProductStatus ürün durumunu günceller.
// status: 100=ACTIVE, 200=INACTIVE, 400=DAILY_INACTIVE
func (c *Client) UpdateProductStatus(productID string, status int) error {
	resp, err := c.do("PUT",
		fmt.Sprintf("/products/%s/status", productID),
		UpdateProductStatusRequest{Status: status},
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "ürün durum güncelleme")
}

// UpdateOptionStatus opsiyon durumunu günceller.
// status: 100=ACTIVE, 200=INACTIVE
func (c *Client) UpdateOptionStatus(optionID string, status int) error {
	resp, err := c.do("PUT",
		fmt.Sprintf("/options/%s/status", optionID),
		UpdateOptionStatusRequest{Status: status},
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "opsiyon durum güncelleme")
}

// --- Chain Menus ---

// GetChainMenus zincir menü listesini getirir
func (c *Client) GetChainMenus() ([]ChainMenu, error) {
	resp, err := c.do("GET", "/chain-menus", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "chain menü listesi"); err != nil {
		return nil, err
	}
	var result []ChainMenu
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("chain menü decode hatası: %w", err)
	}
	return result, nil
}

// GetChainMenu belirli bir zincir menüyü getirir
func (c *Client) GetChainMenu(chainMenuOID string) (*ChainMenu, error) {
	resp, err := c.do("GET", fmt.Sprintf("/chain-menus/%s", chainMenuOID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "chain menü detayı"); err != nil {
		return nil, err
	}
	var result ChainMenu
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("chain menü detayı decode hatası: %w", err)
	}
	return &result, nil
}

// GetChainOptionCategories zincir opsiyon kategorilerini getirir
func (c *Client) GetChainOptionCategories() ([]ChainOptionCategory, error) {
	resp, err := c.do("GET", "/chain-menus/chain-option-categories", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "chain opsiyon kategorileri"); err != nil {
		return nil, err
	}
	var result []ChainOptionCategory
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("chain opsiyon kategorileri decode hatası: %w", err)
	}
	return result, nil
}

// UpdateChainMenuPrices zincir menüdeki ürün ve opsiyon fiyatlarını günceller
func (c *Client) UpdateChainMenuPrices(chainMenuOID string, products []ChainPriceItem, options []ChainPriceItem) error {
	body := UpdateChainPricesRequest{
		ChainProducts: products,
		ChainOptions:  options,
	}
	resp, err := c.do("POST", fmt.Sprintf("/chain-menus/%s/update-prices", chainMenuOID), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "chain fiyat güncelleme")
}

// NewClientFromInfo RestaurantProvider.Information map'inden client oluşturur
func NewClientFromInfo(info map[string]interface{}, service string) (*Client, error) {
	// Getir credentials stored as restaurantId/secretKey but sent as appSecretKey/restaurantSecretKey
	restaurantID, ok := info["restaurantId"].(string)
	if !ok || restaurantID == "" {
		return nil, fmt.Errorf("restaurantId bilgisi eksik")
	}
	secretKey, ok := info["secretKey"].(string)
	if !ok || secretKey == "" {
		return nil, fmt.Errorf("secretKey bilgisi eksik")
	}
	return NewClient(restaurantID, secretKey, service), nil
}
