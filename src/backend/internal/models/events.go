package models

// MissionStatusChangedEvent is published whenever a mission status changes.
type MissionStatusChangedEvent struct {
	MissionID    string `json:"mission_id"`
	IncidentID   string `json:"incident_id"`
	RescueTeamID string `json:"rescue_team_id"`
	OldStatus    string `json:"old_status"`
	NewStatus    string `json:"new_status"`
	ChangedAt    string `json:"changed_at"`
	ChangedBy    string `json:"changed_by"`
}

// MissionBackupRequestedEvent is published when status changes to NEED_BACKUP.
type MissionBackupRequestedEvent struct {
	MissionID    string `json:"mission_id"`
	IncidentID   string `json:"incident_id"`
	RescueTeamID string `json:"rescue_team_id"`
	RequestedAt  string `json:"requested_at"`
	RequestedBy  string `json:"requested_by"`
	Location     string `json:"location,omitempty"`
}

// ImpactLevelUpdatedEvent is published when a team updates the impact level.
type ImpactLevelUpdatedEvent struct {
	MissionID    string `json:"mission_id"`
	IncidentID   string `json:"incident_id"`
	RescueTeamID string `json:"rescue_team_id"`
	OldLevel     int    `json:"old_level"`
	NewLevel     int    `json:"new_level"`
	UpdatedAt    string `json:"updated_at"`
	UpdatedBy    string `json:"updated_by"`
}
