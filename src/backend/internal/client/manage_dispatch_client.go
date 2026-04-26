package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
)

const (
	mdMaxRetries        = 2
	mdBackoffBase       = 100 * time.Millisecond
	mdPerRequestTimeout = 800 * time.Millisecond
)

// ManageDispatchClient calls the Manage Dispatch Service.
type ManageDispatchClient struct {
	baseURL     string
	bearerToken string
	httpClient  *http.Client
}

// NewManageDispatchClient creates a new client.
// Reads MANAGE_DISPATCH_SERVICE_URL and MANAGE_DISPATCH_SERVICE_TOKEN from env.
func NewManageDispatchClient() *ManageDispatchClient {
	baseURL := os.Getenv("MANAGE_DISPATCH_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9997"
	}
	token := os.Getenv("MANAGE_DISPATCH_SERVICE_TOKEN")
	return &ManageDispatchClient{
		baseURL:     baseURL,
		bearerToken: token,
		httpClient: &http.Client{
			Timeout: mdPerRequestTimeout,
		},
	}
}

// GetDispatchByTeamAndRequest ดึงข้อมูล dispatch record สำหรับทีมที่ระบุ
// Endpoint: GET /v1/dispatches?teamId={teamId}
// Auth: Authorization: Bearer <token>
// Returns nil on failure (degraded mode).
func (c *ManageDispatchClient) GetDispatchByTeamAndRequest(ctx context.Context, teamID string) *models.DispatchListResponse {
	url := fmt.Sprintf("%s/v1/dispatches?teamId=%s", c.baseURL, teamID)

	var lastErr error
	for attempt := 0; attempt <= mdMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := mdBackoffBase * (1 << (attempt - 1))
			log.Printf("INFO: Retry %d/%d for ManageDispatchService after %v", attempt, mdMaxRetries, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				log.Printf("WARNING: ManageDispatchService retry aborted — context cancelled")
				return nil
			}
		}

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Accept", "application/json")
		if c.bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.bearerToken)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("WARNING: ManageDispatchService attempt %d failed (network): %v", attempt+1, err)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return nil
		}
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("ManageDispatchService returned %d", resp.StatusCode)
			log.Printf("WARNING: ManageDispatchService attempt %d failed (5xx): status %d", attempt+1, resp.StatusCode)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("WARN: ManageDispatchService returned unexpected status %d (non-retryable)", resp.StatusCode)
			return nil
		}

		var result models.DispatchListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			log.Printf("WARN: decode ManageDispatchService response: %v", err)
			return nil
		}
		resp.Body.Close()
		return &result
	}

	log.Printf("WARN: ManageDispatchService unavailable after %d retries: %v", mdMaxRetries, lastErr)
	return nil
}
