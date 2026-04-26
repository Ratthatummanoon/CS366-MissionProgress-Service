package models

// MissionAssignment represents a mission assignment record in DynamoDB.
type MissionAssignment struct {
	MissionID         string `json:"mission_id"          dynamodbav:"mission_id"`
	DispatchID        string `json:"dispatch_id"         dynamodbav:"dispatch_id"`
	RequestID         string `json:"request_id"          dynamodbav:"request_id"`
	IncidentID        string `json:"incident_id"         dynamodbav:"incident_id"`
	RescueTeamID      string `json:"rescue_team_id"      dynamodbav:"rescue_team_id"`
	PriorityLevel     int    `json:"priority_level"      dynamodbav:"priority_level"`
	CurrentStatus     string `json:"current_status"      dynamodbav:"current_status"`
	LatestImpactLevel int    `json:"latest_impact_level" dynamodbav:"latest_impact_level"`
	StartedAt         string `json:"started_at"          dynamodbav:"started_at"`
	LastUpdatedAt     string `json:"last_updated_at"     dynamodbav:"last_updated_at"`
}
