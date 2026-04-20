package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
)

const (
	maxRetries     = 2             // total 3 attempts
	backoffBase    = 100 * time.Millisecond
	perRequestTimeout = 800 * time.Millisecond
)

// IncidentClient calls the IncidentTracking Service.
type IncidentClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewIncidentClient creates a new client with retry support.
func NewIncidentClient() *IncidentClient {
	baseURL := os.Getenv("INCIDENT_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9999"
	}
	return &IncidentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: perRequestTimeout,
		},
	}
}

// GetIncidentDetail fetches incident details from the IncidentTracking Service.
// Retries up to 2 times with exponential backoff on network errors and 5xx.
// Returns nil on failure (degraded mode).
func (c *IncidentClient) GetIncidentDetail(incidentID string) *models.IncidentDetail {
	url := fmt.Sprintf("%s/incidents/%s", c.baseURL, incidentID)

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := backoffBase * (1 << (attempt - 1)) // 100ms, 200ms
			log.Printf("INFO: Retry %d/%d for IncidentTracking after %v", attempt, maxRetries, backoff)
			time.Sleep(backoff)
		}

		resp, err := c.httpClient.Get(url)
		if err != nil {
			lastErr = err
			log.Printf("WARNING: IncidentTracking attempt %d failed (network): %v", attempt+1, err)
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			log.Printf("WARNING: IncidentTracking attempt %d failed (5xx): status %d", attempt+1, resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("WARNING: IncidentTracking returned status %d (non-retryable)", resp.StatusCode)
			return nil
		}

		var detail models.IncidentDetail
		if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
			resp.Body.Close()
			log.Printf("WARNING: Failed to decode IncidentTracking response: %v", err)
			return nil
		}
		resp.Body.Close()
		return &detail
	}

	log.Printf("WARNING: IncidentTracking Service unavailable after %d attempts: %v", maxRetries+1, lastErr)
	return nil
}
