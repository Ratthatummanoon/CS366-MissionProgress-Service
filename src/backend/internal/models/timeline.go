package models

// TimelineEntry represents a single timeline/action log entry.
type TimelineEntry struct {
	MissionID   string `json:"mission_id" dynamodbav:"mission_id"`
	Timestamp   string `json:"timestamp" dynamodbav:"timestamp"`
	LogID       string `json:"log_id" dynamodbav:"log_id"`
	ActionType  string `json:"action_type" dynamodbav:"action_type"`
	Description string `json:"description" dynamodbav:"description"`
	PerformedBy string `json:"performed_by" dynamodbav:"performed_by"`
	GPSLocation string `json:"gps_location,omitempty" dynamodbav:"gps_location,omitempty"`
}
