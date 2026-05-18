package client

import (
	"bytes"
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
	rrMaxRetries        = 2
	rrBackoffBase       = 100 * time.Millisecond
	rrPerRequestTimeout = 800 * time.Millisecond
)

// RescueRequestClient calls the RescueRequest Service.
type RescueRequestClient struct {
	baseURL     string
	bearerToken string
	httpClient  *http.Client
}

// NewRescueRequestClient creates a new client with retry support.
// Reads RESCUE_REQUEST_SERVICE_URL and RESCUE_REQUEST_SERVICE_TOKEN from env.
func NewRescueRequestClient() *RescueRequestClient {
	baseURL := os.Getenv("RESCUE_REQUEST_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9998"
	}
	token := os.Getenv("RESCUE_REQUEST_SERVICE_TOKEN")
	return &RescueRequestClient{
		baseURL:     baseURL,
		bearerToken: token,
		httpClient: &http.Client{
			Timeout: rrPerRequestTimeout,
		},
	}
}

// GetRequestDetail fetches rescue request details from the RescueRequest Service.
// Endpoint: GET /v1/rescue-requests/{requestId}
// Auth: Authorization: Bearer <token>
// Retries up to 2 times on network errors and 5xx.
// Returns nil on failure (degraded mode).
func (c *RescueRequestClient) GetRequestDetail(ctx context.Context, requestID string) *models.RescueRequestDetail {
	url := fmt.Sprintf("%s/v1/rescue-requests/%s", c.baseURL, requestID)

	var lastErr error
	for attempt := 0; attempt <= rrMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := rrBackoffBase * (1 << (attempt - 1)) // 100ms, 200ms
			log.Printf("INFO: Retry %d/%d for RescueRequestService after %v", attempt, rrMaxRetries, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				log.Printf("WARNING: RescueRequestService retry aborted — context cancelled")
				return nil
			}
		}

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Printf("WARNING: RescueRequestService build request failed: %v", err)
			return nil
		}
		if c.bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.bearerToken)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("WARNING: RescueRequestService attempt %d failed (network): %v", attempt+1, err)
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			log.Printf("WARNING: RescueRequestService attempt %d failed (5xx): status %d", attempt+1, resp.StatusCode)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			log.Printf("WARNING: RescueRequestService returned 404 for requestId=%s", requestID)
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("WARNING: RescueRequestService returned status %d (non-retryable)", resp.StatusCode)
			return nil
		}

		var detail models.RescueRequestDetail
		if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
			resp.Body.Close()
			log.Printf("WARNING: Failed to decode RescueRequestService response: %v", err)
			return nil
		}
		resp.Body.Close()
		return &detail
	}

	log.Printf("WARNING: RescueRequestService all retries exhausted for requestId=%s: %v", requestID, lastErr)
	return nil
}

// postRescueCommand sends a POST to a RescueRequest command endpoint with optional JSON body.
// Retries up to rrMaxRetries times on network errors and 5xx responses.
func (c *RescueRequestClient) postRescueCommand(ctx context.Context, url string, body interface{}) error {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= rrMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := rrBackoffBase * (1 << (attempt - 1))
			log.Printf("INFO: Retry %d/%d for RescueRequestService command after %v", attempt, rrMaxRetries, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				log.Printf("WARNING: RescueRequestService command retry aborted — context cancelled")
				return fmt.Errorf("context cancelled")
			}
		}

		var req *http.Request
		var err error
		if len(bodyBytes) > 0 {
			req, err = http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
		} else {
			req, err = http.NewRequest(http.MethodPost, url, nil)
		}
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		if len(bodyBytes) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.bearerToken)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("WARNING: RescueRequestService command attempt %d failed (network): %v", attempt+1, err)
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			log.Printf("WARNING: RescueRequestService command attempt %d failed (5xx): status %d", attempt+1, resp.StatusCode)
			continue
		}

		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		// 4xx are non-retryable (409 = already in that state, etc.)
		log.Printf("WARNING: RescueRequestService command returned non-retryable status %d for url=%s", resp.StatusCode, url)
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	log.Printf("WARNING: RescueRequestService command all retries exhausted for url=%s: %v", url, lastErr)
	return lastErr
}

// StartRescueRequest calls POST /rescue-requests/{requestId}/start
// Transitions RescueRequest from ASSIGNED → IN_PROGRESS.
// Called when MissionProgress transitions to EN_ROUTE.
func (c *RescueRequestClient) StartRescueRequest(ctx context.Context, requestID string) error {
	url := fmt.Sprintf("%s/v1/rescue-requests/%s/start", c.baseURL, requestID)
	return c.postRescueCommand(ctx, url, nil)
}

// ResolveRescueRequest calls POST /rescue-requests/{requestId}/resolve
// Transitions RescueRequest from IN_PROGRESS → RESOLVED.
// Called when MissionProgress transitions to RESOLVED.
func (c *RescueRequestClient) ResolveRescueRequest(ctx context.Context, requestID string) error {
	url := fmt.Sprintf("%s/v1/rescue-requests/%s/resolve", c.baseURL, requestID)
	return c.postRescueCommand(ctx, url, nil)
}

// CancelRescueRequest calls POST /rescue-requests/{requestId}/cancel
// Transitions RescueRequest to CANCELLED from any non-terminal state.
// reason is required per the API spec.
func (c *RescueRequestClient) CancelRescueRequest(ctx context.Context, requestID string, reason string) error {
	url := fmt.Sprintf("%s/v1/rescue-requests/%s/cancel", c.baseURL, requestID)
	return c.postRescueCommand(ctx, url, map[string]string{"reason": reason})
}

// FormatLocation formats the flat location fields from the master record into a human-readable string.
// e.g., "123 ม.2 ถ.ห้วยแก้ว, สุเทพ, เมืองเชียงใหม่, เชียงใหม่"
func FormatLocation(master models.RescueRequestMaster) string {
	parts := []string{}
	if master.AddressLine != "" {
		parts = append(parts, master.AddressLine)
	}
	if master.Subdistrict != "" {
		parts = append(parts, master.Subdistrict)
	}
	if master.District != "" {
		parts = append(parts, master.District)
	}
	if master.Province != "" {
		parts = append(parts, master.Province)
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
