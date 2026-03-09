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

// IncidentClient calls the IncidentTracking Service.
type IncidentClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewIncidentClient creates a new client with a 3-second timeout.
func NewIncidentClient() *IncidentClient {
	baseURL := os.Getenv("INCIDENT_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9999"
	}
	return &IncidentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// GetIncidentDetail fetches incident details from the IncidentTracking Service.
// Returns nil on failure (degraded mode).
func (c *IncidentClient) GetIncidentDetail(incidentID string) *models.IncidentDetail {
	url := fmt.Sprintf("%s/incidents/%s", c.baseURL, incidentID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		log.Printf("WARNING: IncidentTracking Service unavailable: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("WARNING: IncidentTracking Service returned status %d", resp.StatusCode)
		return nil
	}

	var detail models.IncidentDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		log.Printf("WARNING: Failed to decode IncidentTracking response: %v", err)
		return nil
	}
	return &detail
}
