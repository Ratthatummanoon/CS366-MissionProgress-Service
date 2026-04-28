package models

// TimelineEntry represents a single timeline/action log entry.
type TimelineEntry struct {
	MissionID   string `json:"mission_id" dynamodbav:"mission_id"`
	Timestamp   string `json:"timestamp" dynamodbav:"timestamp"`
	LogID       string `json:"log_id" dynamodbav:"log_id"`
	ActionType  string `json:"action_type" dynamodbav:"action_type"`
	Description string `json:"description" dynamodbav:"description"`
	PerformedBy string `json:"performed_by" dynamodbav:"performed_by"`
	OldStatus   string `json:"old_status,omitempty" dynamodbav:"old_status,omitempty"`
	NewStatus   string `json:"new_status,omitempty" dynamodbav:"new_status,omitempty"`
	Note        string `json:"note,omitempty" dynamodbav:"note,omitempty"`
	GPSLocation string `json:"location,omitempty" dynamodbav:"gps_location,omitempty"`
	ImageKey    string `json:"image_key,omitempty" dynamodbav:"image_key,omitempty"`
}
