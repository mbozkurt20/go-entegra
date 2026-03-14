package yemeksepeti

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	BaseURLStaging    = "https://integration-middleware.stg.restaurant-partners.com"
	BaseURLProduction = "https://integration-middleware.restaurant-partners.com"
)

type Client struct {
	baseURL     string
	username    string
	password    string
	chainCode   string
	posVendorId string
	httpClient  *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewClient(username, password, chainCode, posVendorId, baseURL string) *Client {
	if baseURL == "" {
		baseURL = BaseURLStaging
	}
	return &Client{
		baseURL:     baseURL,
		username:    username,
		password:    password,
		chainCode:   chainCode,
		posVendorId: posVendorId,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

// Login Middleware API'ye giriş yapar ve token cache'ler
func (c *Client) Login() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", c.baseURL+"/v2/login", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("yemeksepeti login isteği başarısız: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("yemeksepeti login hatası %d: %s", resp.StatusCode, string(b))
	}

	var lr LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return "", fmt.Errorf("yemeksepeti login decode hatası: %w", err)
	}

	c.accessToken = lr.AccessToken
	// ExpiresIn saniye, biraz erken yenile
	expiry := time.Duration(lr.ExpiresIn-60) * time.Second
	if expiry < 0 {
		expiry = 30 * time.Second
	}
	c.tokenExpiry = time.Now().Add(expiry)
	return c.accessToken, nil
}

func (c *Client) do(method, path string, body interface{}) (*http.Response, error) {
	token, err := c.Login()
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

func checkResp(resp *http.Response, op string) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("yemeksepeti %s hatası %d: %s", op, resp.StatusCode, string(b))
}

// AcceptOrder siparişi kabul eder
func (c *Client) AcceptOrder(orderToken, remoteOrderId, acceptanceTime string) error {
	req := OrderStatusRequest{
		Status:         StatusOrderAccepted,
		RemoteOrderId:  &remoteOrderId,
		AcceptanceTime: &acceptanceTime,
	}
	resp, err := c.do("POST", "/v2/order/status/"+orderToken, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş kabul")
}

// RejectOrder siparişi reddeder
func (c *Client) RejectOrder(orderToken, reason, message string) error {
	req := OrderStatusRequest{
		Status:  StatusOrderRejected,
		Reason:  &reason,
		Message: &message,
	}
	resp, err := c.do("POST", "/v2/order/status/"+orderToken, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş red")
}

// PickupOrder siparişin kurye tarafından alındığını bildirir
func (c *Client) PickupOrder(orderToken string) error {
	req := OrderStatusRequest{Status: StatusOrderPickedUp}
	resp, err := c.do("POST", "/v2/order/status/"+orderToken, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "sipariş teslim alındı")
}

// PreparationCompleted yemeğin hazır olduğunu kuryeye bildirir
func (c *Client) PreparationCompleted(orderToken string) error {
	resp, err := c.do("POST", "/v2/orders/"+orderToken+"/preparation-completed", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "hazırlık tamamlandı")
}

// GetAvailability mevcut açık/kapalı durumunu getirir
func (c *Client) GetAvailability() ([]AvailabilityEntry, error) {
	path := fmt.Sprintf("/v2/chains/%s/remoteVendors/%s/availability", c.chainCode, c.posVendorId)
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkResp(resp, "availability"); err != nil {
		return nil, err
	}
	var result []AvailabilityEntry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("availability decode hatası: %w", err)
	}
	return result, nil
}

// SetAvailability restoranı açar veya kapatır
func (c *Client) SetAvailability(req AvailabilityUpdateRequest) error {
	path := fmt.Sprintf("/v2/chains/%s/remoteVendors/%s/availability", c.chainCode, c.posVendorId)
	resp, err := c.do("PUT", path, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "availability güncelle")
}

// SetReachability POS sisteminin erişilebilir olup olmadığını bildirir
func (c *Client) SetReachability(reachable bool) error {
	path := fmt.Sprintf("/v2/chains/%s/remoteVendors/%s/posReachabilityStatus", c.chainCode, c.posVendorId)
	status := "REACHABLE"
	if !reachable {
		status = "UNREACHABLE"
	}
	resp, err := c.do("PUT", path, ReachabilityRequest{PosReachabilityStatus: status})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkResp(resp, "reachability güncelle")
}
