package models

// ReportProgressRequest is the body of POST /incidents/{id}/progress.
type ReportProgressRequest struct {
	Status          string `json:"new_status"`
	Note            string `json:"note,omitempty"`
	CurrentLocation string `json:"current_location,omitempty"`
	NewImpactLevel  *int   `json:"new_impact_level,omitempty"`
	ImageKey        string `json:"image_key,omitempty"`
}

// ReportProgressResponse is the response for a successful progress report.
type ReportProgressResponse struct {
	Message    string `json:"message"`
	MissionID  string `json:"mission_id"`
	IncidentID string `json:"incident_id"`
	OldStatus  string `json:"old_status"`
	NewStatus  string `json:"new_status"`
	UpdatedAt  string `json:"updated_at"`
}

// GetMissionResponse is the response for GET /incidents/{id}.
type GetMissionResponse struct {
	IncidentID        string          `json:"incident_id"`
	MissionID         string          `json:"mission_id"`
	RescueTeamID      string          `json:"rescue_team_id"`
	CurrentStatus     string          `json:"current_status"`
	LatestImpactLevel int             `json:"latest_impact_level"`
	StartedAt         string          `json:"started_at"`
	LastUpdatedAt     string          `json:"last_updated_at"`
	Description       string          `json:"description,omitempty"`
	Location          string          `json:"location,omitempty"`
	IncidentType      string          `json:"incident_type,omitempty"`
	Timeline          []TimelineEntry `json:"timeline"`
	DataSource        string          `json:"data_source"`
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"traceId,omitempty"`
}

// PresignedURLRequest is the body of POST /incidents/{id}/presigned-url.
type PresignedURLRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
}

// PresignedURLResponse is the response for a successful presigned URL generation.
type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	ImageKey  string `json:"image_key"`
	ExpiresIn int    `json:"expires_in"`
	Message   string `json:"message"`
}

// ListMissionsResponse is the response for GET /incidents (list missions by team).
type ListMissionsResponse struct {
	TeamID        string              `json:"team_id"`
	TotalMissions int                 `json:"total_missions"`
	Missions      []MissionAssignment `json:"missions"`
}
