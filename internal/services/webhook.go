package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go-entegra/internal/models"

	"gorm.io/gorm"
)

type WebhookService struct {
	db *gorm.DB
}

func NewWebhookService(db *gorm.DB) *WebhookService {
	return &WebhookService{db: db}
}

func (s *WebhookService) SendOrder(order *models.Order, webhookURL string) error {
	if webhookURL == "" {
		return nil
	}

	payload, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("Webhook failed for order %d: %v", order.ID, err)
		return err
	}
	defer resp.Body.Close()

	now := time.Now()
	s.db.Model(order).Updates(map[string]interface{}{
		"webhook_sent":    true,
		"webhook_sent_at": now,
	})

	log.Printf("Webhook sent for order %d to %s, status: %d", order.ID, webhookURL, resp.StatusCode)
	return nil
}
