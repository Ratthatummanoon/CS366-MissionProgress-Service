package models

// IncidentDetail represents data fetched from the IncidentTracking Service.
type IncidentDetail struct {
	IncidentID   string `json:"incident_id"`
	Description  string `json:"description"`
	Location     string `json:"location"`
	IncidentType string `json:"incident_type"`
}
