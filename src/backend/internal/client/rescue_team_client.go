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
	rtMaxRetries        = 2
	rtBackoffBase       = 100 * time.Millisecond
	rtPerRequestTimeout = 800 * time.Millisecond
)

// RescueTeamClient calls the RescueTeam Service.
type RescueTeamClient struct {
	baseURL     string
	bearerToken string
	httpClient  *http.Client
}

// NewRescueTeamClient creates a new client.
// Reads RESCUE_TEAM_SERVICE_URL and RESCUE_TEAM_SERVICE_TOKEN from env.
func NewRescueTeamClient() *RescueTeamClient {
	baseURL := os.Getenv("RESCUE_TEAM_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9996"
	}
	token := os.Getenv("RESCUE_TEAM_SERVICE_TOKEN")
	return &RescueTeamClient{
		baseURL:     baseURL,
		bearerToken: token,
		httpClient: &http.Client{
			Timeout: rtPerRequestTimeout,
		},
	}
}

// GetTeamDetail fetches rescue team details from the RescueTeam Service.
// Endpoint: GET /v1/teams/{team_id}
// Auth: Authorization: Bearer <token>
// Retries up to 2 times on network errors and 5xx.
// Returns nil on failure (degraded mode — caller ยังคืน response ได้โดยไม่มีข้อมูลทีม).
func (c *RescueTeamClient) GetTeamDetail(ctx context.Context, teamID string) *models.RescueTeamDetail {
	url := fmt.Sprintf("%s/v1/teams/%s", c.baseURL, teamID)

	var lastErr error
	for attempt := 0; attempt <= rtMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := rtBackoffBase * (1 << (attempt - 1)) // 100ms, 200ms
			log.Printf("INFO: Retry %d/%d for RescueTeamService.GetTeamDetail after %v", attempt, rtMaxRetries, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				log.Printf("WARNING: RescueTeamService.GetTeamDetail retry aborted — context cancelled")
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
			log.Printf("WARNING: RescueTeamService.GetTeamDetail attempt %d failed (network): %v", attempt+1, err)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			log.Printf("WARN: RescueTeamService team_id=%s not found (404)", teamID)
			return nil
		}
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("RescueTeamService returned %d", resp.StatusCode)
			log.Printf("WARNING: RescueTeamService.GetTeamDetail attempt %d failed (5xx): status %d", attempt+1, resp.StatusCode)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("WARN: RescueTeamService returned unexpected status %d (non-retryable)", resp.StatusCode)
			return nil
		}

		var detail models.RescueTeamDetail
		if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
			resp.Body.Close()
			log.Printf("WARN: decode RescueTeamService response: %v", err)
			return nil
		}
		resp.Body.Close()
		return &detail
	}

	log.Printf("WARN: RescueTeamService unavailable after %d retries: %v", rtMaxRetries, lastErr)
	return nil
}

// UpdateTeamStatus calls PATCH /v1/teams/{team_id}/status to update team availability.
// Endpoint: PATCH /v1/teams/{team_id}/status
// Auth: Authorization: Bearer <token>
// เรียกใช้ตอน mission RESOLVED เพื่อ free team กลับไปเป็น AVAILABLE.
// Best-effort — ถ้าล้มเหลว log แล้วผ่าน (ไม่ block response กลับ caller).
func (c *RescueTeamClient) UpdateTeamStatus(ctx context.Context, teamID, status string) error {
	url := fmt.Sprintf("%s/v1/teams/%s/status", c.baseURL, teamID)

	body, _ := json.Marshal(map[string]string{"status": status})

	var lastErr error
	for attempt := 0; attempt <= rtMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := rtBackoffBase * (1 << (attempt - 1))
			log.Printf("INFO: Retry %d/%d for RescueTeamService.UpdateTeamStatus after %v", attempt, rtMaxRetries, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				log.Printf("WARNING: RescueTeamService.UpdateTeamStatus retry aborted — context cancelled")
				return fmt.Errorf("context cancelled during UpdateTeamStatus retry")
			}
		}

		req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if c.bearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.bearerToken)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("INFO: RescueTeamService team_id=%s status updated to %s", teamID, status)
			return nil
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("RescueTeamService returned %d on status update", resp.StatusCode)
			continue
		}
		// 4xx — validation error หรือ not found → ไม่ retry
		log.Printf("WARN: RescueTeamService status update failed: HTTP %d for team_id=%s", resp.StatusCode, teamID)
		return fmt.Errorf("RescueTeamService rejected status update: HTTP %d", resp.StatusCode)
	}
	return fmt.Errorf("RescueTeamService unavailable after retries: %w", lastErr)
}
